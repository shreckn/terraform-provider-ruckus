package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WLANGroupResource struct{ client *APIClient }

type WLANGroupModel struct {
	ID          types.String `tfsdk:"id"`
	ZoneID      types.String `tfsdk:"zone_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	WLANIds     types.List   `tfsdk:"wlan_ids"` // list of WLAN IDs to add/manage
	Members     types.List   `tfsdk:"members"`  // list of WLAN IDs (computed from API)
}

func NewWLANGroupResource() resource.Resource {
	return &WLANGroupResource{}
}

func (r *WLANGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wlan_group"
}

func (r *WLANGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Ruckus WLAN Group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the WLAN Group.",
				Computed:    true,
			},
			"zone_id": schema.StringAttribute{
				Description: "ID of the zone to which the WLAN Group belongs.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the WLAN Group.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the WLAN Group.",
				Optional:    true,
			},
			"wlan_ids": schema.ListAttribute{
				Description: "List of WLAN IDs to add to this group. Can be existing WLANs or newly created ones.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"members": schema.ListAttribute{
				Description: "List of WLAN IDs that are members of this group (computed from API).",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

func (r *WLANGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *APIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func buildCreateWLANGroupReq(plan *WLANGroupModel) (createWLANGroupReq, error) {
	req := createWLANGroupReq{
		Name: plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		req.Description = plan.Description.ValueString()
	}

	// Note: members are not sent during create/update; they are managed through the WLAN resource

	return req, nil
}

func (r *WLANGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("provider not configured", "missing API client")
		return
	}
	var plan WLANGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	q := url.Values{}
	q.Set("serviceTicket", r.client.ServiceTicket)

	endpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlangroups?%s",
		r.client.BaseURL, r.client.APIVersion, plan.ZoneID.ValueString(), q.Encode())

	payload, err := buildCreateWLANGroupReq(&plan)
	if err != nil {
		resp.Diagnostics.AddError("create failed", err.Error())
		return
	}
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
			resp.Diagnostics.AddWarning("failed to close response body", cerr.Error())
		}
	}()

	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("create failed", fmt.Sprintf("failed to read response body: %v", err))
		return
	}

	if httpResp.StatusCode != http.StatusCreated {
		resp.Diagnostics.AddError("create failed", fmt.Sprintf("HTTP status %d: %s", httpResp.StatusCode, string(bodyBytes)))
		return
	}

	var createResp createWLANGroupResp
	if err := json.Unmarshal(bodyBytes, &createResp); err != nil {
		resp.Diagnostics.AddError("create failed", err.Error())
		return
	}

	plan.ID = types.StringValue(createResp.ID)

	// Add WLANs to the group if specified
	var wlanIds []string
	if !plan.WLANIds.IsNull() && !plan.WLANIds.IsUnknown() {
		plan.WLANIds.ElementsAs(ctx, &wlanIds, false)
		for _, wlanID := range wlanIds {
			q3 := url.Values{}
			q3.Set("serviceTicket", r.client.ServiceTicket)
			addMemberEndpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlangroups/%s/members?%s",
				r.client.BaseURL, r.client.APIVersion, plan.ZoneID.ValueString(), plan.ID.ValueString(), q3.Encode())

			addPayload := addWLANToGroupReq{ID: wlanID}
			addBody, _ := json.Marshal(addPayload)

			addReq, err := http.NewRequestWithContext(ctx, http.MethodPost, addMemberEndpoint, bytes.NewReader(addBody))
			if err != nil {
				resp.Diagnostics.AddError("add wlan to group failed", err.Error())
				return
			}
			addReq.Header.Set("Content-Type", "application/json;charset=UTF-8")

			addResp, err := r.client.HTTP.Do(addReq)
			if err != nil {
				resp.Diagnostics.AddError("add wlan to group failed", err.Error())
				return
			}
			defer func() {
				if cerr := addResp.Body.Close(); cerr != nil {
					resp.Diagnostics.AddWarning("add member response close failed", cerr.Error())
				}
			}()

			if addResp.StatusCode < 200 || addResp.StatusCode > 299 {
				bodyBytes, _ := io.ReadAll(addResp.Body)
				resp.Diagnostics.AddError("add wlan to group failed", fmt.Sprintf("HTTP status %d: %s", addResp.StatusCode, string(bodyBytes)))
				return
			}
			drainBody(addResp.Body)
		}
	}

	// Read the created resource to populate computed fields (members)
	q2 := url.Values{}
	q2.Set("serviceTicket", r.client.ServiceTicket)
	readEndpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlangroups/%s?%s",
		r.client.BaseURL, r.client.APIVersion, plan.ZoneID.ValueString(), plan.ID.ValueString(), q2.Encode())

	readReq, err := http.NewRequestWithContext(ctx, http.MethodGet, readEndpoint, nil)
	if err != nil {
		resp.Diagnostics.AddError("create failed", fmt.Sprintf("failed to read created resource: %v", err))
		return
	}

	readResp, err := r.client.HTTP.Do(readReq)
	if err != nil {
		resp.Diagnostics.AddError("create failed", fmt.Sprintf("failed to read created resource: %v", err))
		return
	}
	defer func() {
		if cerr := readResp.Body.Close(); cerr != nil {
			resp.Diagnostics.AddWarning("failed to close read response body", cerr.Error())
		}
	}()

	if readResp.StatusCode == http.StatusOK {
		var readRespData readWLANGroupResp
		if err := json.NewDecoder(readResp.Body).Decode(&readRespData); err != nil {
			resp.Diagnostics.AddError("create failed", fmt.Sprintf("failed to decode read response: %v", err))
			return
		}

		// Update plan with data from read
		plan.Name = types.StringValue(readRespData.Name)
		if readRespData.Description != "" {
			plan.Description = types.StringValue(readRespData.Description)
		} else {
			plan.Description = types.StringNull()
		}

		members := make([]string, 0, len(readRespData.Members))
		for _, m := range readRespData.Members {
			members = append(members, m.ID)
		}
		plan.Members, _ = types.ListValueFrom(ctx, types.StringType, members)
	} else {
		// Fallback: set empty members list if read fails
		plan.Members, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WLANGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("provider not configured", "missing API client")
		return
	}
	var state WLANGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	q := url.Values{}
	q.Set("serviceTicket", r.client.ServiceTicket)

	endpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlangroups/%s?%s",
		r.client.BaseURL, r.client.APIVersion, state.ZoneID.ValueString(), state.ID.ValueString(), q.Encode())

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		resp.Diagnostics.AddError("read failed", err.Error())
		return
	}

	httpResp, err := r.client.HTTP.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("read failed", err.Error())
		return
	}
	defer func() {
		if cerr := httpResp.Body.Close(); cerr != nil {
			resp.Diagnostics.AddWarning("failed to close response body", cerr.Error())
		}
	}()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("read failed", fmt.Sprintf("HTTP status %d: %s", httpResp.StatusCode, string(bodyBytes)))
		return
	}

	var readResp readWLANGroupResp
	if err := json.NewDecoder(httpResp.Body).Decode(&readResp); err != nil {
		resp.Diagnostics.AddError("read failed", err.Error())
		return
	}

	state.Name = types.StringValue(readResp.Name)
	if readResp.Description != "" {
		state.Description = types.StringValue(readResp.Description)
	} else {
		state.Description = types.StringNull()
	}

	members := make([]string, 0, len(readResp.Members))
	for _, m := range readResp.Members {
		members = append(members, m.ID)
	}
	state.Members, _ = types.ListValueFrom(ctx, types.StringType, members)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WLANGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("provider not configured", "missing API client")
		return
	}
	var plan WLANGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WLANGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Compute differences
	var currentWLANIds []string
	if !state.Members.IsNull() {
		state.Members.ElementsAs(ctx, &currentWLANIds, false)
	}

	var planWLANIds []string
	if !plan.WLANIds.IsNull() && !plan.WLANIds.IsUnknown() {
		plan.WLANIds.ElementsAs(ctx, &planWLANIds, false)
	}

	// Create sets
	currentSet := make(map[string]bool)
	for _, id := range currentWLANIds {
		currentSet[id] = true
	}

	planSet := make(map[string]bool)
	for _, id := range planWLANIds {
		planSet[id] = true
	}

	// To remove: in current but not in plan
	var toRemove []string
	for id := range currentSet {
		if !planSet[id] {
			toRemove = append(toRemove, id)
		}
	}

	// To add: in plan but not in current
	var toAdd []string
	for id := range planSet {
		if !currentSet[id] {
			toAdd = append(toAdd, id)
		}
	}

	q := url.Values{}
	q.Set("serviceTicket", r.client.ServiceTicket)

	// Remove old members
	for _, wlanID := range toRemove {
		removeEndpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlangroups/%s/members/%s?%s",
			r.client.BaseURL, r.client.APIVersion, plan.ZoneID.ValueString(), plan.ID.ValueString(), wlanID, q.Encode())

		removeReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, removeEndpoint, nil)
		if err != nil {
			resp.Diagnostics.AddError("remove wlan from group failed", err.Error())
			return
		}

		removeResp, err := r.client.HTTP.Do(removeReq)
		if err != nil {
			resp.Diagnostics.AddError("remove wlan from group failed", err.Error())
			return
		}
		defer func() {
			if cerr := removeResp.Body.Close(); cerr != nil {
				resp.Diagnostics.AddWarning("remove member response close failed", cerr.Error())
			}
		}()

		if removeResp.StatusCode < 200 || removeResp.StatusCode > 299 {
			bodyBytes, _ := io.ReadAll(removeResp.Body)
			resp.Diagnostics.AddError("remove wlan from group failed", fmt.Sprintf("HTTP status %d: %s", removeResp.StatusCode, string(bodyBytes)))
			return
		}
		drainBody(removeResp.Body)
	}

	// Add new members
	for _, wlanID := range toAdd {
		addMemberEndpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlangroups/%s/members?%s",
			r.client.BaseURL, r.client.APIVersion, plan.ZoneID.ValueString(), plan.ID.ValueString(), q.Encode())

		addPayload := addWLANToGroupReq{ID: wlanID}
		addBody, _ := json.Marshal(addPayload)

		addReq, err := http.NewRequestWithContext(ctx, http.MethodPost, addMemberEndpoint, bytes.NewReader(addBody))
		if err != nil {
			resp.Diagnostics.AddError("add wlan to group failed", err.Error())
			return
		}
		addReq.Header.Set("Content-Type", "application/json;charset=UTF-8")

		addResp, err := r.client.HTTP.Do(addReq)
		if err != nil {
			resp.Diagnostics.AddError("add wlan to group failed", err.Error())
			return
		}
		defer func() {
			if cerr := addResp.Body.Close(); cerr != nil {
				resp.Diagnostics.AddWarning("add member response close failed", cerr.Error())
			}
		}()

		if addResp.StatusCode < 200 || addResp.StatusCode > 299 {
			bodyBytes, _ := io.ReadAll(addResp.Body)
			resp.Diagnostics.AddError("add wlan to group failed", fmt.Sprintf("HTTP status %d: %s", addResp.StatusCode, string(bodyBytes)))
			return
		}
		drainBody(addResp.Body)
	}

	endpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlangroups/%s?%s",
		r.client.BaseURL, r.client.APIVersion, plan.ZoneID.ValueString(), plan.ID.ValueString(), q.Encode())

	payload, err := buildCreateWLANGroupReq(&plan)
	if err != nil {
		resp.Diagnostics.AddError("update failed", err.Error())
		return
	}
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
			resp.Diagnostics.AddWarning("failed to close response body", cerr.Error())
		}
	}()

	if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("update failed", fmt.Sprintf("HTTP status %d: %s", httpResp.StatusCode, string(bodyBytes)))
		return
	}

	// Read the updated resource to populate computed fields (members)
	q2 := url.Values{}
	q2.Set("serviceTicket", r.client.ServiceTicket)
	readEndpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlangroups/%s?%s",
		r.client.BaseURL, r.client.APIVersion, plan.ZoneID.ValueString(), plan.ID.ValueString(), q2.Encode())

	readReq, err := http.NewRequestWithContext(ctx, http.MethodGet, readEndpoint, nil)
	if err != nil {
		resp.Diagnostics.AddError("update failed", fmt.Sprintf("failed to read updated resource: %v", err))
		return
	}

	readResp, err := r.client.HTTP.Do(readReq)
	if err != nil {
		resp.Diagnostics.AddError("update failed", fmt.Sprintf("failed to read updated resource: %v", err))
		return
	}
	defer func() {
		if cerr := readResp.Body.Close(); cerr != nil {
			resp.Diagnostics.AddWarning("failed to close read response body", cerr.Error())
		}
	}()

	if readResp.StatusCode == http.StatusOK {
		var readRespData readWLANGroupResp
		if err := json.NewDecoder(readResp.Body).Decode(&readRespData); err != nil {
			resp.Diagnostics.AddError("update failed", fmt.Sprintf("failed to decode read response: %v", err))
			return
		}

		// Update plan with data from read
		if readRespData.Description != "" {
			plan.Description = types.StringValue(readRespData.Description)
		} else {
			plan.Description = types.StringNull()
		}

		members := make([]string, 0, len(readRespData.Members))
		for _, m := range readRespData.Members {
			members = append(members, m.ID)
		}
		plan.Members, _ = types.ListValueFrom(ctx, types.StringType, members)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WLANGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("provider not configured", "missing API client")
		return
	}
	var state WLANGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	q := url.Values{}
	q.Set("serviceTicket", r.client.ServiceTicket)

	endpoint := fmt.Sprintf("%s/wsg/api/public/%s/rkszones/%s/wlangroups/%s?%s",
		r.client.BaseURL, r.client.APIVersion, state.ZoneID.ValueString(), state.ID.ValueString(), q.Encode())

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		resp.Diagnostics.AddError("delete failed", err.Error())
		return
	}

	httpResp, err := r.client.HTTP.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("delete failed", err.Error())
		return
	}
	defer func() {
		if cerr := httpResp.Body.Close(); cerr != nil {
			resp.Diagnostics.AddWarning("failed to close response body", cerr.Error())
		}
	}()

	if httpResp.StatusCode >= 400 && httpResp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("delete failed", fmt.Sprintf("HTTP status %d: %s", httpResp.StatusCode, string(bodyBytes)))
		return
	}
}

type createWLANGroupReq struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Members     *[]string `json:"members,omitempty"`
}

type wlanGroupMember struct {
	ID string `json:"id"`
}

type createWLANGroupResp struct {
	ID string `json:"id"`
}

type readWLANGroupResp struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Members     []wlanGroupMember `json:"members"`
}

type addWLANToGroupReq struct {
	ID string `json:"id"`
}
