package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WLANResource struct{ client *APIClient }

type WLANModel struct {
	ID          types.String `tfsdk:"id"`          // computed
	ZoneID      types.String `tfsdk:"zone_id"`     // required
	Name        types.String `tfsdk:"name"`        // required
	SSID        types.String `tfsdk:"ssid"`        // required
	Description types.String `tfsdk:"description"` // optional
	// add other fields as needed (auth, encryption, vlan, etc.)
}

func NewWLANResource() resource.Resource { return &WLANResource{} }

func (r *WLANResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "ruckus_wlan"
}

func (r *WLANResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true},
			"zone_id":     schema.StringAttribute{Required: true},
			"name":        schema.StringAttribute{Required: true},
			"ssid":        schema.StringAttribute{Required: true},
			"description": schema.StringAttribute{Optional: true},
		},
	}
}

func (r *WLANResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*APIClient)
	}
}

// ---- API payloads (simplified) ----
type createWLANReq struct {
	Name        string `json:"name"`
	SSID        string `json:"ssid"`
	Description string `json:"description,omitempty"`
}
type createWLANResp struct {
	ID string `json:"id"`
}

func (r *WLANResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("not configured", "missing API client")
		return
	}
	var plan WLANModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	q := url.Values{}
	q.Set("serviceTicket", r.client.ServiceTicket)

	endpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlans?%s",
		r.client.BaseURL, r.client.APIVersion, plan.ZoneID.ValueString(), q.Encode())

	body, _ := json.Marshal(createWLANReq{
		Name: plan.Name.ValueString(), SSID: plan.SSID.ValueString(),
		Description: plan.Description.ValueString(),
	})
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json;charset=UTF-8")

	httpResp, err := r.client.HTTP.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("create failed", err.Error())
		return
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
		resp.Diagnostics.AddError("create failed", fmt.Sprintf("status %d", httpResp.StatusCode))
		return
	}
	var cr createWLANResp
	if err := json.NewDecoder(httpResp.Body).Decode(&cr); err != nil {
		resp.Diagnostics.AddError("decode failed", err.Error())
		return
	}
	plan.ID = types.StringValue(cr.ID)
	resp.State.Set(ctx, &plan)
}

func (r *WLANResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WLANModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// GET /rkszones/{zoneId}/wlans/{id}
	q := url.Values{}
	q.Set("serviceTicket", r.client.ServiceTicket)
	endpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlans/%s?%s",
		r.client.BaseURL, r.client.APIVersion, state.ZoneID.ValueString(), state.ID.ValueString(), q.Encode())

	var out struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		SSID        string `json:"ssid"`
		Description string `json:"description,omitempty"`
	}
	if err := doGET(ctx, r.client, endpoint, &out); err != nil {
		// If 404, mark resource gone
		resp.State.RemoveResource(ctx)
		return
	}
	state.Name = types.StringValue(out.Name)
	state.SSID = types.StringValue(out.SSID)
	state.Description = types.StringPointerValue(&out.Description)
	resp.State.Set(ctx, &state)
}

func (r *WLANResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WLANModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	q := url.Values{}
	q.Set("serviceTicket", r.client.ServiceTicket)
	endpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlans/%s?%s",
		r.client.BaseURL, r.client.APIVersion, plan.ZoneID.ValueString(), plan.ID.ValueString(), q.Encode())

	body, _ := json.Marshal(createWLANReq{
		Name: plan.Name.ValueString(), SSID: plan.SSID.ValueString(),
		Description: plan.Description.ValueString(),
	})
	reqHTTP, _ := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json;charset=UTF-8")

	httpResp, err := r.client.HTTP.Do(reqHTTP)
	if err != nil {
		resp.Diagnostics.AddError("update failed", err.Error())
		return
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
		resp.Diagnostics.AddError("update failed", fmt.Sprintf("status %d", httpResp.StatusCode))
		return
	}
	resp.State.Set(ctx, &plan)
}

func (r *WLANResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WLANModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	q := url.Values{}
	q.Set("serviceTicket", r.client.ServiceTicket)
	endpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlans/%s?%s",
		r.client.BaseURL, r.client.APIVersion, state.ZoneID.ValueString(), state.ID.ValueString(), q.Encode())

	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	httpReq.Header.Set("Content-Type", "application/json;charset=UTF-8")
	httpResp, err := r.client.HTTP.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("delete failed", err.Error())
		return
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode >= 400 && httpResp.StatusCode != 404 {
		resp.Diagnostics.AddError("delete failed", fmt.Sprintf("status %d", httpResp.StatusCode))
	}
}
