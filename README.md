# Terraform Provider for Laminar

Manage [Laminar](https://laminar.sh) Signals and LLM profiles with Terraform or OpenTofu.

| Type | Name | Notes |
|---|---|---|
| Resource | `lmnr_signal` | Prompt, structured output, trigger, filters, sampling, mode, LLM profile routing (self-hosted) |
| Resource | `lmnr_llm_profile` | Workspace-scoped provider credentials and models |
| Data source | `lmnr_signal`, `lmnr_llm_profile` | Look up by `id` or exact `name` |
| Data source | `lmnr_project` | The project that owns the API key |

## Example

```hcl
terraform {
  required_providers {
    lmnr = {
      source = "lmnr-ai/lmnr"
    }
  }
}

# Reads LMNR_PROJECT_API_KEY from the environment.
provider "lmnr" {}

resource "lmnr_signal" "failure_detector" {
  name   = "Failure detector"
  prompt = "Identify failed or abandoned runs."

  structured_output = jsonencode({
    type       = "object"
    properties = { failed = { type = "boolean" } }
    required   = ["failed"]
  })
}
```

Run `terraform init` to install the provider from the Terraform Registry, then `terraform apply`.

Full reference: [Terraform Registry](https://registry.terraform.io/providers/lmnr-ai/lmnr/latest/docs) (source in [`docs/`](docs/)). More examples: [`examples/`](examples/).

## Configuration

| Argument | Environment variable | Default |
|---|---|---|
| `project_api_key` | `LMNR_PROJECT_API_KEY` | (required) |
| `base_url` | `LMNR_BASE_URL` | `https://api.lmnr.ai` |
| `http_port` | `LMNR_HTTP_PORT` | The port in `base_url`, or `443` |

Arguments in the provider block take precedence over environment variables. Create a project API key in the Laminar project's settings.

A self-hosted deployment usually needs only the URL and port:

```hcl
provider "lmnr" {
  base_url  = "http://laminar.internal"
  http_port = 8000
}
```

A project API key scopes the provider to one project. To manage several projects, configure one provider alias per project key:

```hcl
provider "lmnr" {
  alias           = "staging"
  project_api_key = var.staging_project_api_key
}

resource "lmnr_signal" "staging_failures" {
  provider = lmnr.staging
  # ...
}
```

LLM profiles belong to the project's workspace, so every project in that workspace can use them.

## Importing existing resources

Every resource can be imported by UUID, which you can copy from the Laminar UI:

```shell
terraform import lmnr_signal.failure_detector <signal-uuid>
terraform import lmnr_llm_profile.openai <llm-profile-uuid>
```

After importing an LLM profile, set its credentials in configuration. The API never returns them, and the next apply writes them.

## Behavior worth knowing

- Destroying a `lmnr_signal` deletes its events, which can take a minute on large projects. Use `lifecycle { prevent_destroy = true }` for Signals you care about, or set `disabled = true` to pause one.
- Omitting `trigger` or `filters` on a Signal applies the server defaults (`rootSpanFinished`, `total_token_count > 1000`). Set `filters = []` to evaluate every trace.
- LLM profile credentials are write-only in the API. Terraform stores the configured values in state as sensitive, so keep state encrypted.
- The LLM profile provider attribute is `llm_provider`, because `provider` is a reserved Terraform meta-argument.
- `llm_profile_id` and `model` on a Signal only apply to self-hosted deployments. Laminar Cloud rejects them.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[Apache 2.0](LICENSE)
