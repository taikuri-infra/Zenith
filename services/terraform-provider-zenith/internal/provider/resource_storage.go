package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &storageResource{}
	_ resource.ResourceWithConfigure = &storageResource{}
)

type storageResource struct {
	client *ZenithClient
}

type storageResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Access    types.String `tfsdk:"access"`
	ProjectID types.String `tfsdk:"project_id"`
	// Computed
	Region    types.String `tfsdk:"region"`
	SizeMB    types.Int64  `tfsdk:"size_mb"`
	MaxSizeMB types.Int64  `tfsdk:"max_size_mb"`
	Objects   types.Int64  `tfsdk:"objects"`
	Status    types.String `tfsdk:"status"`
	Endpoint  types.String `tfsdk:"endpoint"`
}

func NewStorageResource() resource.Resource {
	return &storageResource{}
}

func (r *storageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage"
}

func (r *storageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zenith S3-compatible storage bucket.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Bucket ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Bucket name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access": schema.StringAttribute{
				Description: "Access level: 'private' or 'public'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("private"),
			},
			"project_id": schema.StringAttribute{
				Description: "Project ID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				Description: "Storage region.",
				Computed:    true,
			},
			"size_mb": schema.Int64Attribute{
				Description: "Current size in MB.",
				Computed:    true,
			},
			"max_size_mb": schema.Int64Attribute{
				Description: "Maximum size in MB.",
				Computed:    true,
			},
			"objects": schema.Int64Attribute{
				Description: "Number of objects.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Bucket status.",
				Computed:    true,
			},
			"endpoint": schema.StringAttribute{
				Description: "S3 endpoint URL.",
				Computed:    true,
			},
		},
	}
}

func (r *storageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *storageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan storageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":   plan.Name.ValueString(),
		"access": plan.Access.ValueString(),
	}
	if !plan.ProjectID.IsNull() && !plan.ProjectID.IsUnknown() {
		body["project_id"] = plan.ProjectID.ValueString()
	}

	var result storageAPIResponse
	if err := r.client.Post(ctx, "/api/v1/storage-buckets", body, &result); err != nil {
		resp.Diagnostics.AddError("Error creating storage bucket", err.Error())
		return
	}

	r.mapResponseToState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *storageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state storageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result storageAPIResponse
	if err := r.client.Get(ctx, "/api/v1/storage-buckets/"+state.ID.ValueString(), &result); err != nil {
		resp.Diagnostics.AddError("Error reading storage bucket", err.Error())
		return
	}

	r.mapResponseToState(&state, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *storageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan storageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"access": plan.Access.ValueString(),
	}

	var result storageAPIResponse
	if err := r.client.Put(ctx, "/api/v1/storage-buckets/"+plan.ID.ValueString(), body, &result); err != nil {
		resp.Diagnostics.AddError("Error updating storage bucket", err.Error())
		return
	}

	r.mapResponseToState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *storageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state storageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/storage-buckets/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting storage bucket", err.Error())
		return
	}
}

func (r *storageResource) mapResponseToState(state *storageResourceModel, result *storageAPIResponse) {
	state.ID = types.StringValue(result.ID)
	state.Name = types.StringValue(result.Name)
	state.Access = types.StringValue(result.Access)
	state.Region = types.StringValue(result.Region)
	state.SizeMB = types.Int64Value(int64(result.SizeMB))
	state.MaxSizeMB = types.Int64Value(int64(result.MaxSizeMB))
	state.Objects = types.Int64Value(int64(result.Objects))
	state.Status = types.StringValue(result.Status)
	state.Endpoint = types.StringValue(result.Endpoint)
}

type storageAPIResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Access    string `json:"access"`
	Region    string `json:"region"`
	SizeMB    int    `json:"size_mb"`
	MaxSizeMB int    `json:"max_size_mb"`
	Objects   int    `json:"objects"`
	Status    string `json:"status"`
	Endpoint  string `json:"endpoint"`
}
