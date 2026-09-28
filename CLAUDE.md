# CLAUDE.md

This repository is the Laminar Terraform provider (`registry: lmnr-ai/lmnr`). It's built on Terraform Plugin Framework and wraps the Laminar project API that `app-server` serves in [lmnr-ai/lmnr](https://github.com/lmnr-ai/lmnr).

## Layout

- `internal/client/`: HTTP client, one file per API resource. `contract_test.go` checks JSON tags and paths against `openapi/openapi.yaml`.
- `internal/provider/`: resources, data sources, and tests.
  - `fake_api_test.go` is an in-memory Laminar API used by the acceptance tests.
  - `live_test.go` runs against a real server.
- The API also serves `/v1/datasets`, but datasets are intentionally not a resource (see "Open decisions" in `CONTRIBUTING.md`).
- `examples/`, `docs/`: `docs/` is generated from schema descriptions and examples by `make generate`. Never edit `docs/` by hand; CI fails if it is stale.

## Commands

```shell
go test ./...                                 # unit + OpenAPI contract tests
TF_ACC=1 go test ./internal/...               # + Terraform CLI acceptance tests (needs terraform on PATH)
golangci-lint run ./...                       # CI lint (v2)
make generate                                 # terraform fmt examples + regenerate docs/
make sync-openapi                             # refresh vendored spec from ../docs (lmnr-ai/docs)
```

`resource.Test` skips without `TF_ACC=1`. The validation tests use `resource.UnitTest` and always run.

Live test: `LMNR_TF_LIVE_TEST=1 LMNR_PROJECT_API_KEY=... LMNR_BASE_URL=... TF_ACC=1 go test ./internal/provider -run TestAccLive -count=1 -timeout 30m -v`.
- Add `LMNR_TF_LIVE_SELF_HOSTED=1` only against a server without `LAMINAR_CLOUD`. Self-hosted requires `llmProfileId` and `model` on Signals; Cloud rejects them.
- Afterwards, check that no `tf-acc-*` resources are left.

## Gotchas

- **Hand-written, not generated.** `tfplugingen-openapi` can't parse the spec's `allOf`/`oneOf` and generates no CRUD logic. The contract test is the drift guard.
- **`provider` is a reserved root attribute name**, and the schema fails to load if you use it. The LLM profile field is `llm_provider`.
- **Signal DELETE purges ClickHouse data synchronously.** It took about 62s on staging, which is why the client timeout is 5 minutes.
- **The API doesn't preserve `structuredOutput` key order.** `canonicalJSON` re-marshals it with sorted keys, like `jsonencode`, or import-verify diffs.
- **LLM profile secrets:**
  - The API returns only masks, merges secrets on update, and prunes header secrets whose names aren't in `headerNames`.
  - State keeps the configured values, marked sensitive.
  - `llmProfileToModel` nulls a secret the server reports absent.
- **Server-side validation mirrored at plan time.** Keep these in sync with `lmnr`:
  - `signalFilterColumns` ↔ `FILTER_COLUMNS` in `app-server/src/signals/service.rs`.
  - `llmProviderFields` ↔ `app-server/src/llm/profiles/service/provider_fields.rs`. The server rejects config fields a provider doesn't use.
- The Signal list endpoint is an ILIKE substring match on `name`, so the data source filters for an exact match.
- The docs spec's `LlmProfileProvider` enum lacks `custom_responses`. The server supports it.

## Style

- Comments explain a WHY that names can't: a constraint, an invariant, or a mirrored server rule. Keep them to a line or two.
- Run `gofmt` on changed files; `gofmt` is at `$(go env GOROOT)/bin/gofmt` if it's not on PATH.
