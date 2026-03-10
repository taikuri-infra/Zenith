package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &domainResource{}
	_ resource.ResourceWithConfigure = &domainResource{}
)

type domainResource struct {
	client *ZenithClient
}

type domainResourceModel struct {
	ID       types.String `tfsdk:"id"`
	AppID    types.String `tfsdk:"app_id"`
	Domain   types.String `tfsdk:"domain"`
	Status   types.String `tfsdk:"status"`
	TLSReady types.Bool   `tfsdk:"tls_ready"`
}

func NewDomainResource() resource.Resource {
	return &domainResource{}
}

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a custom domain attached to a Zenith application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Domain ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_id": schema.StringAttribute{
				Description: "Application ID to attach the domain to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain": schema.StringAttribute{
				Description: "Domain name (e.g., 'app.example.com').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Domain verification status.",
				Computed:    true,
			},
			"tls_ready": schema.BoolAttribute{
				Description: "Whether TLS certificate is provisioned.",
				Computed:    true,
			},
		},
	}
}

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*ZenithClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *ZenithClient")
		return
	}
	r.client = client
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]string{
		"domain": plan.Domain.ValueString(),
	}

	var result domainAPIResponse
	if err := r.client.Post(ctx, fmt.Sprintf("/api/v1/apps/%s/domains", plan.AppID.ValueString()), body, &result); err != nil {
		resp.Diagnostics.AddError("Error creating domain", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	plan.Status = types.StringValue(result.Status)
	plan.TLSReady = types.BoolValue(result.TLSReady)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// List domains for the app and find this one
	var result struct {
		Domains []domainAPIResponse `json:"domains"`
	}
	if err := r.client.Get(ctx, fmt.Sprintf("/api/v1/apps/%s/domains", state.AppID.ValueString()), &result); err != nil {
		resp.Diagnostics.AddError("Error reading domains", err.Error())
		return
	}

	for _, d := range result.Domains {
		if d.ID == state.ID.ValueString() {
			state.Domain = types.StringValue(d.Domain)
			state.Status = types.StringValue(d.Status)
			state.TLSReady = types.BoolValue(d.TLSReady)
			resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
			return
		}
	}

	// Domain not found — remove from state
	resp.State.RemoveResource(ctx)
}

func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Both app_id and domain require replacement, so update should never be called
	resp.Diagnostics.AddError("Update not supported", "Domain changes require replacement.")
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/apps/%s/domains/%s", state.AppID.ValueString(), state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting domain", err.Error())
		return
	}
}

type domainAPIResponse struct {
	ID       string `json:"id"`
	AppID    string `json:"app_id"`
	Domain   string `json:"domain"`
	Status   string `json:"status"`
	TLSReady bool   `json:"tls_ready"`
}
