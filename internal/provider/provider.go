package provider

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lmnr-ai/terraform-provider-laminar/internal/client"
)

var _ provider.Provider = &LaminarProvider{}

type LaminarProvider struct{ version string }

type LaminarProviderModel struct {
	ProjectAPIKey types.String `tfsdk:"project_api_key"`
	BaseURL       types.String `tfsdk:"base_url"`
	HTTPPort      types.Int64  `tfsdk:"http_port"`
}

func (p *LaminarProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "laminar"
	resp.Version = p.version
}

func (p *LaminarProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Laminar resources through the Laminar project API. A project API key scopes the provider to one project; " +
			"use one provider alias per project. LLM profiles belong to the project's workspace.",
		Attributes: map[string]schema.Attribute{
			"project_api_key": schema.StringAttribute{
				Optional: true, Sensitive: true,
				MarkdownDescription: "Laminar project API key. May also be set with `LMNR_PROJECT_API_KEY`.",
			},
			"base_url": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Laminar API base URL without the port, the same value the Laminar SDKs use. " +
					"Defaults to `https://api.lmnr.ai`; may also be set with `LMNR_BASE_URL`. Set the port with `http_port`.",
			},
			"http_port": schema.Int64Attribute{
				Optional: true,
				MarkdownDescription: "Laminar API HTTP port. Defaults to `443`, like the Laminar SDKs, even for `http://` base URLs; " +
					"when unset, a port in `base_url` is used, then `LMNR_HTTP_PORT`.",
				Validators: []validator.Int64{int64validator.Between(1, 65535)},
			},
		},
	}
}

func (p *LaminarProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config LaminarProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("LMNR_PROJECT_API_KEY")
	if !config.ProjectAPIKey.IsNull() && !config.ProjectAPIKey.IsUnknown() {
		apiKey = config.ProjectAPIKey.ValueString()
	}
	baseURL := os.Getenv("LMNR_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.lmnr.ai"
	}
	if !config.BaseURL.IsNull() && !config.BaseURL.IsUnknown() {
		baseURL = config.BaseURL.ValueString()
	}
	port := ""
	if known(config.HTTPPort) {
		port = strconv.FormatInt(config.HTTPPort.ValueInt64(), 10)
	}
	if config.ProjectAPIKey.IsUnknown() || config.BaseURL.IsUnknown() || config.HTTPPort.IsUnknown() {
		return
	}
	if apiKey == "" {
		resp.Diagnostics.AddError("Missing Laminar project API key", "Set project_api_key in the provider configuration or LMNR_PROJECT_API_KEY in the environment.")
		return
	}

	endpoint, err := apiEndpoint(baseURL, port, os.Getenv("LMNR_HTTP_PORT"))
	if err != nil {
		resp.Diagnostics.AddError("Invalid Laminar API endpoint", err.Error())
		return
	}
	api, err := client.New(endpoint, apiKey, "terraform-provider-laminar/"+p.version, nil)
	if err != nil {
		resp.Diagnostics.AddError("Unable to configure Laminar client", err.Error())
		return
	}
	resp.ResourceData = api
	resp.DataSourceData = api
}

// apiEndpoint resolves the port like the Laminar SDKs, which default to 443
// regardless of scheme. Precedence: http_port, a port in baseURL, LMNR_HTTP_PORT.
func apiEndpoint(baseURL, port, envPort string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		return "", fmt.Errorf("base URL %q must be an http(s) URL such as https://api.lmnr.ai", baseURL)
	}
	for _, candidate := range []string{port, parsed.Port(), envPort, "443"} {
		if candidate != "" {
			port = candidate
			break
		}
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("HTTP port %q must be a number between 1 and 65535", port)
	}
	parsed.Host = net.JoinHostPort(parsed.Hostname(), port)
	return parsed.String(), nil
}

func (p *LaminarProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewSignalResource, NewLlmProfileResource}
}

func (p *LaminarProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSignalDataSource, NewLlmProfileDataSource, NewProjectDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &LaminarProvider{version: version} }
}
