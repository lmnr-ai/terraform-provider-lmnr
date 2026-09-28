package provider

import (
	"fmt"
	"net/url"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAPIEndpoint(t *testing.T) {
	t.Parallel()
	cases := []struct {
		baseURL string
		port    int64
		want    string
	}{
		{"https://api.lmnr.ai", 0, "https://api.lmnr.ai:443"},
		{"http://localhost:8000", 0, "http://localhost:8000"},
		{"http://localhost", 8000, "http://localhost:8000"},
		{"http://localhost:8000", 9000, "http://localhost:9000"},
		{"https://lmnr.example.com/api", 0, "https://lmnr.example.com:443/api"},
		{"http://[::1]", 8000, "http://[::1]:8000"},
	}
	for _, tc := range cases {
		got, err := apiEndpoint(tc.baseURL, tc.port)
		if err != nil || got != tc.want {
			t.Errorf("apiEndpoint(%q, %d) = %q, %v; want %q", tc.baseURL, tc.port, got, err, tc.want)
		}
	}
	if _, err := apiEndpoint("localhost:8000", 0); err == nil {
		t.Error("expected an error for a URL without a scheme")
	}
}

func TestAccProviderHTTPPortOverride(t *testing.T) {
	_, baseURL := newFakeAPI(t)
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	withoutPort := parsed.Scheme + "://" + parsed.Hostname()

	t.Setenv("LMNR_HTTP_PORT", parsed.Port())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`provider "lmnr" {
  project_api_key = "test-key"
  base_url        = %q
}
data "lmnr_project" "current" {}`, withoutPort),
			Check: resource.TestCheckResourceAttr("data.lmnr_project.current", "id", fakeProjectID),
		}},
	})
}

func TestProviderInvalidHTTPPortEnv(t *testing.T) {
	t.Setenv("LMNR_HTTP_PORT", "https")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      providerConfig("http://127.0.0.1") + `data "lmnr_project" "current" {}`,
			ExpectError: regexp.MustCompile("Invalid LMNR_HTTP_PORT"),
		}},
	})
}
