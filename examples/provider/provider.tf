terraform {
  required_providers {
    laminar = {
      source = "lmnr-ai/laminar"
    }
  }
}

# Reads LMNR_PROJECT_API_KEY, LMNR_BASE_URL and LMNR_HTTP_PORT from the
# environment. Self-hosted deployments can set base_url and http_port here.
provider "laminar" {}
