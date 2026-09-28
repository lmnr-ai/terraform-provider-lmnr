package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectDataSource(t *testing.T) {
	_, baseURL := newFakeAPI(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: providerConfig(baseURL) + `data "laminar_project" "current" {}`,
			Check:  resource.TestCheckResourceAttr("data.laminar_project.current", "id", fakeProjectID),
		}},
	})
}
