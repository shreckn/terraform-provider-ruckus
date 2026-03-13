package provider

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ProviderModel struct {
	Host               types.String `tfsdk:"host"` // e.g., https://sz.example.com:8443
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	Domain             types.String `tfsdk:"domain"`               // e.g., "System"
	APIVersion         types.String `tfsdk:"api_version"`          // e.g., v13_1 (SZ 7.1.1)
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"` // labs only
	TimeoutSeconds     types.Int64  `tfsdk:"timeout_seconds"`
}

type ruckusProvider struct{}

func New() provider.Provider { return &ruckusProvider{} }

func (p *ruckusProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ruckus"
}

func (p *ruckusProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host":                 schema.StringAttribute{Required: true},
			"username":             schema.StringAttribute{Optional: true, Sensitive: true},
			"password":             schema.StringAttribute{Optional: true, Sensitive: true},
			"domain":               schema.StringAttribute{Optional: true},
			"api_version":          schema.StringAttribute{Optional: true},
			"insecure_skip_verify": schema.BoolAttribute{Optional: true},
			"timeout_seconds":      schema.Int64Attribute{Optional: true},
		},
	}
}

func (p *ruckusProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg ProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify.ValueBool()}}
	timeout := 30 * time.Second
	if !cfg.TimeoutSeconds.IsNull() {
		timeout = time.Duration(cfg.TimeoutSeconds.ValueInt64()) * time.Second
	}
	httpClient := &http.Client{Transport: tr, Timeout: timeout}

	apiVersion := "v13_1" // reasonable default for SZ 7.1.1 (overridable). [3](https://docs.ruckuswireless.com/smartzone/7.1.1/vsze-public-api-reference-guide-711.html)
	if !cfg.APIVersion.IsNull() && !cfg.APIVersion.IsUnknown() && cfg.APIVersion.ValueString() != "" {
		apiVersion = cfg.APIVersion.ValueString()
	}

	ticket, err := LoginForServiceTicket(
		ctx, httpClient,
		cfg.Host.ValueString(), apiVersion,
		cfg.Username.ValueString(), cfg.Password.ValueString(), cfg.Domain.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Ruckus SmartZone login failed", err.Error())
		return
	}

	client := &APIClient{
		BaseURL:       cfg.Host.ValueString(),
		APIVersion:    apiVersion,
		ServiceTicket: ticket,
		HTTP:          httpClient,
	}
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ruckusProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewWLANResource,
	}
}

func (p *ruckusProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewZoneDataSource,
	}
}
