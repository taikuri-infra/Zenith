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
	_ resource.Resource              = &databaseResource{}
	_ resource.ResourceWithConfigure = &databaseResource{}
)

type databaseResource struct {
	client *ZenithClient
}

type databaseResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Engine           types.String `tfsdk:"engine"`
	ProjectID        types.String `tfsdk:"project_id"`
	// Computed
	Host             types.String `tfsdk:"host"`
	Port             types.Int64  `tfsdk:"port"`
	DBName           types.String `tfsdk:"db_name"`
	DBUser           types.String `tfsdk:"db_user"`
	Password         types.String `tfsdk:"password"`
	ConnectionString types.String `tfsdk:"connection_string"`
	SizeMB           types.Int64  `tfsdk:"size_mb"`
	MaxSizeMB        types.Int64  `tfsdk:"max_size_mb"`
	Status           types.String `tfsdk:"status"`
}

func NewDatabaseResource() resource.Resource {
	return &databaseResource{}
}

func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (r *databaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a standalone Zenith database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Database ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Database name.",
				Optional:    true,
				Computed:    true,
			},
			"engine": schema.StringAttribute{
				Description: "Database engine: 'postgresql', 'mysql', 'redis', 'mongodb'.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "Project ID. If not set, uses the default project.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host": schema.StringAttribute{
				Description: "Database host.",
				Computed:    true,
			},
			"port": schema.Int64Attribute{
				Description: "Database port.",
				Computed:    true,
			},
			"db_name": schema.StringAttribute{
				Description: "Database name on the server.",
				Computed:    true,
			},
			"db_user": schema.StringAttribute{
				Description: "Database user.",
				Computed:    true,
			},
			"password": schema.StringAttribute{
				Description: "Database password (only available after creation).",
				Computed:    true,
				Sensitive:   true,
			},
			"connection_string": schema.StringAttribute{
				Description: "Full connection string (only available after creation).",
				Computed:    true,
				Sensitive:   true,
			},
			"size_mb": schema.Int64Attribute{
				Description: "Current database size in MB.",
				Computed:    true,
			},
			"max_size_mb": schema.Int64Attribute{
				Description: "Maximum database size in MB.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Database status.",
				Computed:    true,
			},
		},
	}
}

func (r *databaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *databaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan databaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"engine": plan.Engine.ValueString(),
	}
	if !plan.Name.IsNull() {
		body["name"] = plan.Name.ValueString()
	}
	if !plan.ProjectID.IsNull() && !plan.ProjectID.IsUnknown() {
		body["project_id"] = plan.ProjectID.ValueString()
	}

	var result databaseAPIResponse
	if err := r.client.Post(ctx, "/api/v1/databases", body, &result); err != nil {
		resp.Diagnostics.AddError("Error creating database", err.Error())
		return
	}

	r.mapResponseToState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *databaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result databaseAPIResponse
	if err := r.client.Get(ctx, "/api/v1/databases/"+state.ID.ValueString(), &result); err != nil {
		resp.Diagnostics.AddError("Error reading database", err.Error())
		return
	}

	r.mapResponseToState(&state, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *databaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Databases are immutable (engine change requires replace). Only name can't be updated via API.
	resp.Diagnostics.AddError("Update not supported", "Databases cannot be updated in-place. Change the engine to trigger a replacement.")
}

func (r *databaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/databases/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting database", err.Error())
		return
	}
}

func (r *databaseResource) mapResponseToState(state *databaseResourceModel, result *databaseAPIResponse) {
	state.ID = types.StringValue(result.ID)
	state.Name = types.StringValue(result.Name)
	state.Engine = types.StringValue(result.Engine)
	state.Host = types.StringValue(result.Host)
	state.Port = types.Int64Value(int64(result.Port))
	state.DBName = types.StringValue(result.DBName)
	state.DBUser = types.StringValue(result.DBUser)
	state.SizeMB = types.Int64Value(int64(result.SizeMB))
	state.MaxSizeMB = types.Int64Value(int64(result.MaxSizeMB))
	state.Status = types.StringValue(result.Status)

	if result.Password != "" {
		state.Password = types.StringValue(result.Password)
	}
	if result.ConnectionString != "" {
		state.ConnectionString = types.StringValue(result.ConnectionString)
	}
}

type databaseAPIResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Engine           string `json:"engine"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	DBName           string `json:"db_name"`
	DBUser           string `json:"db_user"`
	Password         string `json:"db_password,omitempty"`
	ConnectionString string `json:"connection_string,omitempty"`
	SizeMB           int    `json:"size_mb"`
	MaxSizeMB        int    `json:"max_size_mb"`
	Status           string `json:"status"`
}
