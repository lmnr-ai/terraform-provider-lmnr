# Terraform Provider for Laminar

Manage [Laminar](https://laminar.sh) Signals and LLM profiles with Terraform.

| Type | Name | Notes |
|---|---|---|
| Resource | `laminar_signal` | Prompt, structured output, trigger, filters, sampling, mode, LLM profile routing (self-hosted) |
| Resource | `laminar_llm_profile` | Workspace-scoped provider credentials and models |
| Data source | `laminar_signal`, `laminar_llm_profile` | Look up by `id` or exact `name` |
| Data source | `laminar_project` | The project that owns the API key |

## Example

```hcl
terraform {
  required_providers {
    laminar = {
      source = "lmnr-ai/laminar"
    }
  }
}

# Reads LMNR_PROJECT_API_KEY; self-hosted users also set LMNR_BASE_URL or base_url.
provider "laminar" {}

resource "laminar_signal" "failure_detector" {
  name   = "Failure detector"
  prompt = "Identify failed or abandoned runs."

  structured_output = jsonencode({
    type       = "object"
    properties = { failed = { type = "boolean" } }
    required   = ["failed"]
  })
}
```

A project API key scopes the provider to one project. To manage several projects, configure one provider alias per project key. LLM profiles belong to the project's workspace.

Full reference: [`docs/`](docs/). Examples: [`examples/`](examples/).

## Behavior worth knowing

- Destroying a `laminar_signal` deletes its events, which can take a minute on large projects. Use `lifecycle { prevent_destroy = true }` for Signals you care about, or set `disabled = true` to pause one.
- Omitting `trigger` or `filters` on a Signal applies the server defaults (`rootSpanFinished`, `total_token_count > 1000`). Set `filters = []` to evaluate every trace.
- LLM profile credentials are write-only in the API. Terraform stores the configured values in state as sensitive, so keep state encrypted. After `terraform import`, set the credentials in configuration; the next apply writes them.
- The LLM profile provider attribute is `llm_provider`, because `provider` is a reserved Terraform meta-argument.
- `llm_profile_id` and `model` on a Signal only apply to self-hosted deployments. Laminar Cloud rejects them.

## Development

Requirements: Go 1.25+ and Terraform 1.0+.

```shell
make build                       # go build ./...
go test ./...                    # unit and OpenAPI contract tests
TF_ACC=1 go test ./internal/...  # adds Terraform CLI acceptance tests against an in-memory Laminar API
make generate                    # terraform fmt the examples and regenerate docs/
make sync-openapi                # copy docs/openapi/openapi.yaml from the docs repo and re-run the contract test
```

The live test creates and destroys real resources:

```shell
LMNR_TF_LIVE_TEST=1 LMNR_PROJECT_API_KEY=... LMNR_BASE_URL=http://localhost:8000 \
  TF_ACC=1 go test ./internal/provider -run TestAccLive -count=1 -timeout 30m -v
```

Add `LMNR_TF_LIVE_SELF_HOSTED=1` on a self-hosted server to also create an LLM profile and route the Signal through it. The profile uses a dummy key; nothing calls the LLM.

The provider is written by hand against the API. `tfplugingen-openapi` doesn't support the spec's `allOf`/`oneOf` shapes and generates no CRUD logic. `internal/client/contract_test.go` checks the client's request and response types against the vendored `openapi/openapi.yaml`, so API drift fails the tests.

## Maintaining

The provider wraps the project API served by `app-server` in [lmnr-ai/lmnr](https://github.com/lmnr-ai/lmnr). When that API changes:

- Run `make sync-openapi` (with the `lmnr-ai/docs` repo checked out next to this one, or `OPENAPI_SPEC=<path>`). The contract test fails if a client type no longer matches the spec.
- Update the plan-time validation that mirrors the server:
  - `signalFilterColumns` mirrors `FILTER_COLUMNS` in `app-server/src/signals/service.rs`.
  - `llmProviderFields` mirrors `app-server/src/llm/profiles/service/provider_fields.rs`.

The docs spec's `LlmProfileProvider` enum is missing `custom_responses`, which the server supports. The provider accepts it.

### Releasing

`release.yml` runs GoReleaser on `v*` tags. It needs the `GPG_PRIVATE_KEY` and `PASSPHRASE` repository secrets. The public key must be registered with the Terraform Registry. The Registry only publishes public repositories named `terraform-provider-<name>`.

### Open decisions

- LLM profile secrets are sensitive values in state. With Terraform ≥ 1.11 as the minimum version, they could become write-only attributes, with a version attribute that triggers rotation.
- Not managed yet: Signal alerts, LLM feature routes, custom model costs. Each needs a project API endpoint in app-server first.
- Datasets are left out on purpose. A name-only resource adds little, and destroying one deletes its datapoints; revisit when another resource references datasets.
