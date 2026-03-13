package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ZoneDS struct{ client *APIClient }

type ZoneDSModel struct {
	Name types.String `tfsdk:"name"`
	ID   types.String `tfsdk:"id"`
}

func NewZoneDataSource() datasource.DataSource { return &ZoneDS{} }

func (d *ZoneDS) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "ruckus_zone"
}

func (d *ZoneDS) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{Required: true},
			"id":   schema.StringAttribute{Computed: true},
		},
	}
}

func (d *ZoneDS) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*APIClient)
	}
}

type zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type zonesResp struct {
	List []zone `json:"list"`
}

func (d *ZoneDS) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("not configured", "missing API client")
		return
	}
	var state ZoneDSModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// GET {base}/wsg/api/public/{ver}/rkszones?serviceTicket=...&name={name}
	q := url.Values{}
	q.Set("serviceTicket", d.client.ServiceTicket)
	q.Set("name", state.Name.ValueString())

	endpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones?%s",
		d.client.BaseURL, d.client.APIVersion, q.Encode())

	var zr zonesResp
	if err := doGET(ctx, d.client.HTTP, endpoint, &zr); err != nil { // <-- pass *http.Client
		resp.Diagnostics.AddError("read zone failed", err.Error())
		return
	}
	for _, z := range zr.List {
		if z.Name == state.Name.ValueString() {
			state.ID = types.StringValue(z.ID)
			resp.State.Set(ctx, &state)
			return
		}
	}
	resp.Diagnostics.AddError("not found", "zone not found by name")
}
