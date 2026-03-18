package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WLANResource struct{ client *APIClient }

type WLANSecurityModel struct {
	Mode          types.String `tfsdk:"mode"`
	Passphrase    types.String `tfsdk:"passphrase"`
	AuthProfileID types.String `tfsdk:"auth_profile_id"`
	Encryption    types.String `tfsdk:"encryption"`
}
type WLANVLANModel struct {
	AccessVLAN types.Int64 `tfsdk:"access_vlan"`
}
type WLANRadioModel struct {
	Band            types.String `tfsdk:"band"`
	ClientIsolation types.Bool   `tfsdk:"client_isolation"`
}
type WLANTunnelModel struct {
	Type      types.String `tfsdk:"type"`
	ProfileID types.String `tfsdk:"profile_id"`
}
type WLANAdvancedModel struct {
	MinBSSRate types.Int64 `tfsdk:"min_bss_rate"`
	OFDMA      types.Bool  `tfsdk:"ofdma"`
}

type WLANModel struct {
	ID          types.String `tfsdk:"id"`
	ZoneID      types.String `tfsdk:"zone_id"`
	Name        types.String `tfsdk:"name"`
	SSID        types.String `tfsdk:"ssid"`
	Description types.String `tfsdk:"description"`

	Security *WLANSecurityModel `tfsdk:"security"`
	VLAN     *WLANVLANModel     `tfsdk:"vlan"`
	Radio    *WLANRadioModel    `tfsdk:"radio"`
	Tunnel   *WLANTunnelModel   `tfsdk:"tunnel"`
	Advanced *WLANAdvancedModel `tfsdk:"advanced"`
}

func buildCreateWLANReq(plan *WLANModel) createWLANReq {
	req := createWLANReq{
		Name: plan.Name.ValueString(),
		SSID: plan.SSID.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		req.Description = plan.Description.ValueString()
	}

	if plan.Security != nil {
		s := &wlanSecurity{}
		if !plan.Security.Mode.IsNull() {
			s.Mode = plan.Security.Mode.ValueString()
		}
		if !plan.Security.Passphrase.IsNull() {
			s.Passphrase = plan.Security.Passphrase.ValueString()
		}
		if !plan.Security.AuthProfileID.IsNull() {
			s.AuthProfileID = plan.Security.AuthProfileID.ValueString()
		}
		if !plan.Security.Encryption.IsNull() {
			s.Encryption = plan.Security.Encryption.ValueString()
		}
		req.Security = s
	}

	if plan.VLAN != nil {
		v := &wlanVLAN{}
		if !plan.VLAN.AccessVLAN.IsNull() {
			av := int(plan.VLAN.AccessVLAN.ValueInt64())
			v.AccessVLAN = &av
		}
		req.VLAN = v
	}

	if plan.Radio != nil {
		r := &wlanRadio{}
		if !plan.Radio.Band.IsNull() {
			r.Band = plan.Radio.Band.ValueString()
		}
		if !plan.Radio.ClientIsolation.IsNull() {
			ci := plan.Radio.ClientIsolation.ValueBool()
			r.ClientIsolation = &ci
		}
		req.Radio = r
	}

	if plan.Tunnel != nil {
		t := &wlanTunnel{}
		if !plan.Tunnel.Type.IsNull() {
			t.Type = plan.Tunnel.Type.ValueString()
		}
		if !plan.Tunnel.ProfileID.IsNull() {
			t.ProfileID = plan.Tunnel.ProfileID.ValueString()
		}
		req.Tunnel = t
	}

	if plan.Advanced != nil {
		a := &wlanAdvanced{}
		if !plan.Advanced.MinBSSRate.IsNull() {
			mbr := int(plan.Advanced.MinBSSRate.ValueInt64())
			a.MinBSSRate = &mbr
		}
		if !plan.Advanced.OFDMA.IsNull() {
			o := plan.Advanced.OFDMA.ValueBool()
			a.OFDMA = &o
		}
		req.Advanced = a
	}

	return req
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
		Blocks: map[string]schema.Block{
			"security": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					// e.g. "open", "wep", "wpa2_psk", "wpa3_sae", "wpa2_wpa3_mixed", "8021x", "wpa3_enterprise", "webauth", "wispr", "owe"
					"mode": schema.StringAttribute{
						Optional: true,
						Validators: []validator.String{
							stringvalidator.OneOf("open", "wep", "wpa2_psk", "wpa3_sae", "wpa2_wpa3_mixed", "8021x", "wpa3_enterprise", "webauth", "wispr", "owe"),
						},
					},
					// PSK for *_psk modes
					"passphrase": schema.StringAttribute{Optional: true, Sensitive: true},
					// RADIUS / AAA profile id for 802.1X, or auth server reference
					"auth_profile_id": schema.StringAttribute{Optional: true},
					// encryption hints if your firmware expects them (e.g., "ccmp", "tkip_ccmp", "sae", "owe")
					"encryption": schema.StringAttribute{Optional: true},
				},
			},
			"vlan": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"access_vlan": schema.Int64Attribute{Optional: true}, // static access VLAN
				},
			},
			"radio": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					// "2.4", "5", "6", "both" (consult your API version; some expose per‑band flags)
					"band":             schema.StringAttribute{Optional: true},
					"client_isolation": schema.BoolAttribute{Optional: true},
				},
			},
			"tunnel": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					// "none", "ruckus_gre", "soft_gre", "ipsec"
					"type":       schema.StringAttribute{Optional: true},
					"profile_id": schema.StringAttribute{Optional: true},
				},
			},
			"advanced": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"min_bss_rate": schema.Int64Attribute{Optional: true}, // minimum basic rate (kbps)
					"ofdma":        schema.BoolAttribute{Optional: true},  // if supported by firmware/band
				},
			},
		},
	}
}

func (r *WLANResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*APIClient)
	}
}

// ---- API payloads (example fields; verify against your controller OpenAPI) ----
// Security
type wlanSecurity struct {
	// e.g., "open", "wep", "wpa2_psk", "wpa3_sae", "wpa2_wpa3_mixed", "8021x", "wpa3_enterprise", "webauth", "wispr", "owe"
	Mode string `json:"method,omitempty"`
	// For PSK modes
	Passphrase string `json:"passphrase,omitempty"`
	// AAA / RADIUS profile
	AuthProfileID string `json:"authServerId,omitempty"`
	// e.g., "ccmp", "tkip_ccmp", "sae", "owe" (depends on mode)
	Encryption string `json:"encryption,omitempty"`
}

// VLAN
type wlanVLAN struct {
	AccessVLAN *int `json:"accessVlan,omitempty"`
}

// Radio/band
type wlanRadio struct {
	Band            string `json:"band,omitempty"` // "2.4","5","6","both" (verify)
	ClientIsolation *bool  `json:"clientIsolation,omitempty"`
}

// Tunneling
type wlanTunnel struct {
	Type      string `json:"type,omitempty"`      // "ruckus_gre","soft_gre","ipsec"
	ProfileID string `json:"profileId,omitempty"` // pre-created tunnel profile id
}

// Advanced
type wlanAdvanced struct {
	MinBSSRate *int  `json:"minBssRate,omitempty"` // kbps
	OFDMA      *bool `json:"ofdma,omitempty"`
}

type createWLANReq struct {
	Name        string        `json:"name"`
	SSID        string        `json:"ssid"`
	Description string        `json:"description,omitempty"`
	Security    *wlanSecurity `json:"security,omitempty"`
	VLAN        *wlanVLAN     `json:"vlan,omitempty"`
	Radio       *wlanRadio    `json:"radio,omitempty"`
	Tunnel      *wlanTunnel   `json:"tunnel,omitempty"`
	Advanced    *wlanAdvanced `json:"advanced,omitempty"`
}

type createWLANResp struct {
	ID string `json:"id"`
}

type wlanResponse struct {
	ID          string        `json:"id"`
	ZoneID      string        `json:"zoneId,omitempty"`
	Name        string        `json:"name"`
	SSID        string        `json:"ssid"`
	Description string        `json:"description,omitempty"`
	Security    *wlanSecurity `json:"security,omitempty"`
	VLAN        *wlanVLAN     `json:"vlan,omitempty"`
	Radio       *wlanRadio    `json:"radio,omitempty"`
	Tunnel      *wlanTunnel   `json:"tunnel,omitempty"`
	Advanced    *wlanAdvanced `json:"advanced,omitempty"`
}

func (r *WLANResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("provider not configured", "missing API client")
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

	payload := buildCreateWLANReq(&plan)
	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		resp.Diagnostics.AddError("create failed", err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json;charset=UTF-8")

	httpResp, err := r.client.HTTP.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("create failed", err.Error())
		return
	}
	defer func() {
		if cerr := httpResp.Body.Close(); cerr != nil {
			resp.Diagnostics.AddWarning("response close failed", cerr.Error())
		}
	}()

	if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("create failed", fmt.Sprintf("status %d: %s", httpResp.StatusCode, string(bodyBytes)))
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
	if r.client == nil {
		resp.Diagnostics.AddError("provider not configured", "missing API client")
		return
	}
	var state WLANModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	q := url.Values{}
	q.Set("serviceTicket", r.client.ServiceTicket)
	endpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlans/%s?%s",
		r.client.BaseURL, r.client.APIVersion, state.ZoneID.ValueString(), state.ID.ValueString(), q.Encode())

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		resp.Diagnostics.AddError("read failed", err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json;charset=UTF-8")

	httpResp, err := r.client.HTTP.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("read failed", err.Error())
		return
	}
	defer func() {
		if cerr := httpResp.Body.Close(); cerr != nil {
			resp.Diagnostics.AddWarning("response close failed", cerr.Error())
		}
	}()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("read failed", fmt.Sprintf("status %d: %s", httpResp.StatusCode, string(bodyBytes)))
		return
	}

	var out wlanResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&out); err != nil {
		resp.Diagnostics.AddError("decode failed", err.Error())
		return
	}
	state.ID = types.StringValue(out.ID)
	if out.ZoneID != "" {
		state.ZoneID = types.StringValue(out.ZoneID)
	}
	state.Name = types.StringValue(out.Name)
	state.SSID = types.StringValue(out.SSID)
	if out.Description != "" {
		state.Description = types.StringValue(out.Description)
	} else {
		state.Description = types.StringNull()
	}

	if out.Security != nil {
		state.Security = &WLANSecurityModel{}
		if out.Security.Mode != "" {
			state.Security.Mode = types.StringValue(out.Security.Mode)
		} else {
			state.Security.Mode = types.StringNull()
		}
		if out.Security.Passphrase != "" {
			state.Security.Passphrase = types.StringValue(out.Security.Passphrase)
		} else {
			state.Security.Passphrase = types.StringNull()
		}
		if out.Security.AuthProfileID != "" {
			state.Security.AuthProfileID = types.StringValue(out.Security.AuthProfileID)
		} else {
			state.Security.AuthProfileID = types.StringNull()
		}
		if out.Security.Encryption != "" {
			state.Security.Encryption = types.StringValue(out.Security.Encryption)
		} else {
			state.Security.Encryption = types.StringNull()
		}
	} else {
		state.Security = nil
	}

	if out.VLAN != nil {
		state.VLAN = &WLANVLANModel{}
		if out.VLAN.AccessVLAN != nil {
			state.VLAN.AccessVLAN = types.Int64Value(int64(*out.VLAN.AccessVLAN))
		} else {
			state.VLAN.AccessVLAN = types.Int64Null()
		}
	} else {
		state.VLAN = nil
	}

	if out.Radio != nil {
		state.Radio = &WLANRadioModel{}
		if out.Radio.Band != "" {
			state.Radio.Band = types.StringValue(out.Radio.Band)
		} else {
			state.Radio.Band = types.StringNull()
		}
		if out.Radio.ClientIsolation != nil {
			state.Radio.ClientIsolation = types.BoolValue(*out.Radio.ClientIsolation)
		} else {
			state.Radio.ClientIsolation = types.BoolNull()
		}
	} else {
		state.Radio = nil
	}

	if out.Tunnel != nil {
		state.Tunnel = &WLANTunnelModel{}
		if out.Tunnel.Type != "" {
			state.Tunnel.Type = types.StringValue(out.Tunnel.Type)
		} else {
			state.Tunnel.Type = types.StringNull()
		}
		if out.Tunnel.ProfileID != "" {
			state.Tunnel.ProfileID = types.StringValue(out.Tunnel.ProfileID)
		} else {
			state.Tunnel.ProfileID = types.StringNull()
		}
	} else {
		state.Tunnel = nil
	}

	if out.Advanced != nil {
		state.Advanced = &WLANAdvancedModel{}
		if out.Advanced.MinBSSRate != nil {
			state.Advanced.MinBSSRate = types.Int64Value(int64(*out.Advanced.MinBSSRate))
		} else {
			state.Advanced.MinBSSRate = types.Int64Null()
		}
		if out.Advanced.OFDMA != nil {
			state.Advanced.OFDMA = types.BoolValue(*out.Advanced.OFDMA)
		} else {
			state.Advanced.OFDMA = types.BoolNull()
		}
	} else {
		state.Advanced = nil
	}

	resp.State.Set(ctx, &state)
}

func (r *WLANResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("provider not configured", "missing API client")
		return
	}
	var plan WLANModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	q := url.Values{}
	q.Set("serviceTicket", r.client.ServiceTicket)
	endpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlans/%s?%s",
		r.client.BaseURL, r.client.APIVersion, plan.ZoneID.ValueString(), plan.ID.ValueString(), q.Encode())
	payload := buildCreateWLANReq(&plan) // same shape for PUT in most versions
	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		resp.Diagnostics.AddError("update failed", err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json;charset=UTF-8")

	httpResp, err := r.client.HTTP.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("update failed", err.Error())
		return
	}
	defer func() {
		if cerr := httpResp.Body.Close(); cerr != nil {
			resp.Diagnostics.AddWarning("response close failed", cerr.Error())
		}
	}()

	if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("update failed", fmt.Sprintf("status %d: %s", httpResp.StatusCode, string(bodyBytes)))
		return
	}
	drainBody(httpResp.Body)
	resp.State.Set(ctx, &plan)
}

func (r *WLANResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("provider not configured", "missing API client")
		return
	}
	var state WLANModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	q := url.Values{}
	q.Set("serviceTicket", r.client.ServiceTicket)
	endpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlans/%s?%s",
		r.client.BaseURL, r.client.APIVersion, state.ZoneID.ValueString(), state.ID.ValueString(), q.Encode())

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		resp.Diagnostics.AddError("delete failed", err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json;charset=UTF-8")

	httpResp, err := r.client.HTTP.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("delete failed", err.Error())
		return
	}
	defer func() {
		if cerr := httpResp.Body.Close(); cerr != nil {
			resp.Diagnostics.AddWarning("response close failed", cerr.Error())
		}
	}()

	// 404 on delete is typically safe to treat as "already gone".
	if httpResp.StatusCode >= 400 && httpResp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("delete failed", fmt.Sprintf("status %d: %s", httpResp.StatusCode, string(bodyBytes)))
		return
	}
	drainBody(httpResp.Body)
}
