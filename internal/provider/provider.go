package provider

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lmnr-ai/terraform-provider-lmnr/internal/client"
)

var _ provider.Provider = &LaminarProvider{}

type LaminarProvider struct{ version string }

type LaminarProviderModel struct {
	ProjectAPIKey types.String `tfsdk:"project_api_key"`
	BaseURL       types.String `tfsdk:"base_url"`
	HTTPPort      types.Int64  `tfsdk:"http_port"`
}

func (p *LaminarProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "lmnr"
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
				Optional:            true,
				MarkdownDescription: "Laminar API base URL, usually without a port. Defaults to `https://api.lmnr.ai`; may also be set with `LMNR_BASE_URL`.",
			},
			"http_port": schema.Int64Attribute{
				Optional: true,
				MarkdownDescription: "Laminar API HTTP port. May also be set with `LMNR_HTTP_PORT`. " +
					"Defaults to the port in `base_url` if it has one, otherwise `443`.",
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
	if config.ProjectAPIKey.IsUnknown() || config.BaseURL.IsUnknown() || config.HTTPPort.IsUnknown() {
		return
	}
	port := config.HTTPPort.ValueInt64()
	if config.HTTPPort.IsNull() {
		if raw := os.Getenv("LMNR_HTTP_PORT"); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || parsed < 1 || parsed > 65535 {
				resp.Diagnostics.AddError("Invalid LMNR_HTTP_PORT", fmt.Sprintf("LMNR_HTTP_PORT must be a port number between 1 and 65535, got %q.", raw))
				return
			}
			port = parsed
		}
	}
	endpoint, err := apiEndpoint(baseURL, port)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Laminar base URL", err.Error())
		return
	}
	if apiKey == "" {
		resp.Diagnostics.AddError("Missing Laminar project API key", "Set project_api_key in the provider configuration or LMNR_PROJECT_API_KEY in the environment.")
		return
	}

	api, err := client.New(endpoint, apiKey, "terraform-provider-lmnr/"+p.version, nil)
	if err != nil {
		resp.Diagnostics.AddError("Unable to configure Laminar client", err.Error())
		return
	}
	resp.ResourceData = api
	resp.DataSourceData = api
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

// apiEndpoint applies the SDKs' port rule: an explicit port wins, then a port
// already in the base URL, then 443.
func apiEndpoint(baseURL string, port int64) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Hostname() == "" {
		return "", fmt.Errorf("base URL must be an absolute URL such as https://api.lmnr.ai, got %q", baseURL)
	}
	switch {
	case port != 0:
		parsed.Host = net.JoinHostPort(parsed.Hostname(), strconv.FormatInt(port, 10))
	case parsed.Port() == "":
		parsed.Host = net.JoinHostPort(parsed.Hostname(), "443")
	}
	return parsed.String(), nil
}
