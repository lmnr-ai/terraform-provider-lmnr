terraform {
  required_providers {
    laminar = {
      source = "lmnr-ai/laminar"
    }
  }
}

provider "laminar" {
  # Configure with LMNR_PROJECT_API_KEY. Self-hosted users also set LMNR_BASE_URL
  # (no port, as in the SDKs) and LMNR_HTTP_PORT, or base_url and http_port here.
  # The port defaults to 443, even for http:// URLs.
}
