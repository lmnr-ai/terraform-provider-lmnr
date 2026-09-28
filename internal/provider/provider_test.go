package provider

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"laminar": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("acceptance tests skipped with -short")
	}
}

func testAccLivePreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("LMNR_TF_LIVE_TEST") == "" {
		t.Skip("set LMNR_TF_LIVE_TEST=1 to run against a live Laminar API")
	}
	if os.Getenv("LMNR_PROJECT_API_KEY") == "" {
		t.Fatal("LMNR_PROJECT_API_KEY must be set for live acceptance tests")
	}
}

func TestAPIEndpoint(t *testing.T) {
	tests := map[string]struct {
		baseURL, port, envPort, want, err string
	}{
		"cloud default":         {baseURL: "https://api.lmnr.ai", want: "https://api.lmnr.ai:443"},
		"http without port":     {baseURL: "http://10.0.0.1", want: "http://10.0.0.1:443"},
		"http_port":             {baseURL: "http://localhost", port: "8000", want: "http://localhost:8000"},
		"env port":              {baseURL: "http://localhost", envPort: "8000", want: "http://localhost:8000"},
		"port in base url":      {baseURL: "http://localhost:8000/", want: "http://localhost:8000"},
		"http_port beats url":   {baseURL: "http://localhost:8000", port: "9000", want: "http://localhost:9000"},
		"url port beats env":    {baseURL: "http://localhost:8000", envPort: "443", want: "http://localhost:8000"},
		"http_port beats env":   {baseURL: "http://localhost", port: "8000", envPort: "443", want: "http://localhost:8000"},
		"path kept":             {baseURL: "https://example.com/laminar/", want: "https://example.com:443/laminar"},
		"ipv6":                  {baseURL: "http://[::1]", port: "8000", want: "http://[::1]:8000"},
		"missing scheme":        {baseURL: "api.lmnr.ai", err: "must be an http(s) URL"},
		"non-numeric env port":  {baseURL: "http://localhost", envPort: "http", err: "between 1 and 65535"},
		"out of range env port": {baseURL: "http://localhost", envPort: "70000", err: "between 1 and 65535"},
		"out of range url port": {baseURL: "http://localhost:0", err: "between 1 and 65535"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := apiEndpoint(tt.baseURL, tt.port, tt.envPort)
			if tt.err != "" {
				if err == nil || !strings.Contains(err.Error(), tt.err) {
					t.Fatalf("error = %v, want %q", err, tt.err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}
