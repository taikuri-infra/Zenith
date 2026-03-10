package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &appResource{}
	_ resource.ResourceWithConfigure = &appResource{}
)

type appResource struct {
	client *ZenithClient
}

type appResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	Name             types.String `tfsdk:"name"`
	DeploySource     types.String `tfsdk:"deploy_source"`
	RepoURL          types.String `tfsdk:"repo_url"`
	Branch           types.String `tfsdk:"branch"`
	ImageURL         types.String `tfsdk:"image_url"`
	RegistryUsername types.String `tfsdk:"registry_username"`
	RegistryPassword types.String `tfsdk:"registry_password"`
	Port             types.Int64  `tfsdk:"port"`
	AppType          types.String `tfsdk:"app_type"`
	Command          types.String `tfsdk:"command"`
	CronSchedule     types.String `tfsdk:"cron_schedule"`
	// Computed
	Status    types.String `tfsdk:"status"`
	Subdomain types.String `tfsdk:"subdomain"`
	URL       types.String `tfsdk:"url"`
	Framework types.String `tfsdk:"framework"`
}

func NewAppResource() resource.Resource {
	return &appResource{}
}

func (r *appResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *appResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zenith application deployment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Application ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
			"name": schema.StringAttribute{
				Description: "Application name.",
				Required:    true,
			},
			"deploy_source": schema.StringAttribute{
				Description: "Deploy source: 'git' or 'image'.",
				Required:    true,
			},
			"repo_url": schema.StringAttribute{
				Description: "Git repository URL (required when deploy_source is 'git').",
				Optional:    true,
			},
			"branch": schema.StringAttribute{
				Description: "Git branch to deploy.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("main"),
			},
			"image_url": schema.StringAttribute{
				Description: "Container image URL (required when deploy_source is 'image').",
				Optional:    true,
			},
			"registry_username": schema.StringAttribute{
				Description: "Registry username for private images.",
				Optional:    true,
			},
			"registry_password": schema.StringAttribute{
				Description: "Registry password for private images.",
				Optional:    true,
				Sensitive:   true,
			},
			"port": schema.Int64Attribute{
				Description: "Container port.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(8080),
			},
			"app_type": schema.StringAttribute{
				Description: "Application type: 'web', 'worker', or 'cron'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("web"),
			},
			"command": schema.StringAttribute{
				Description: "Custom command for worker/cron apps.",
				Optional:    true,
			},
			"cron_schedule": schema.StringAttribute{
				Description: "Cron schedule expression (for cron apps).",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "Application status.",
				Computed:    true,
			},
			"subdomain": schema.StringAttribute{
				Description: "Auto-assigned subdomain.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "Application URL.",
				Computed:    true,
			},
			"framework": schema.StringAttribute{
				Description: "Detected framework.",
				Computed:    true,
			},
		},
	}
}

func (r *appResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *appResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"deploy_source": plan.DeploySource.ValueString(),
		"port":          plan.Port.ValueInt64(),
		"app_type":      plan.AppType.ValueString(),
	}
	if !plan.ProjectID.IsNull() && !plan.ProjectID.IsUnknown() {
		body["project_id"] = plan.ProjectID.ValueString()
	}
	if !plan.RepoURL.IsNull() {
		body["repo_url"] = plan.RepoURL.ValueString()
	}
	if !plan.Branch.IsNull() {
		body["branch"] = plan.Branch.ValueString()
	}
	if !plan.ImageURL.IsNull() {
		body["image_url"] = plan.ImageURL.ValueString()
	}
	if !plan.RegistryUsername.IsNull() {
		body["registry_username"] = plan.RegistryUsername.ValueString()
	}
	if !plan.RegistryPassword.IsNull() {
		body["registry_password"] = plan.RegistryPassword.ValueString()
	}
	if !plan.Command.IsNull() {
		body["command"] = plan.Command.ValueString()
	}
	if !plan.CronSchedule.IsNull() {
		body["cron_schedule"] = plan.CronSchedule.ValueString()
	}

	var result appAPIResponse
	if err := r.client.Post(ctx, "/api/v1/apps", body, &result); err != nil {
		resp.Diagnostics.AddError("Error creating app", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	plan.ProjectID = types.StringValue(result.ProjectID)
	plan.Status = types.StringValue(result.Status)
	plan.Subdomain = types.StringValue(result.Subdomain)
	plan.URL = types.StringValue(result.URL)
	plan.Framework = types.StringValue(result.Framework)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *appResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result appAPIResponse
	if err := r.client.Get(ctx, "/api/v1/apps/"+state.ID.ValueString(), &result); err != nil {
		resp.Diagnostics.AddError("Error reading app", err.Error())
		return
	}

	state.ProjectID = types.StringValue(result.ProjectID)
	state.Name = types.StringValue(result.Name)
	state.DeploySource = types.StringValue(result.DeploySource)
	state.Status = types.StringValue(result.Status)
	state.Subdomain = types.StringValue(result.Subdomain)
	state.URL = types.StringValue(result.URL)
	state.Framework = types.StringValue(result.Framework)
	state.Port = types.Int64Value(int64(result.Port))
	state.AppType = types.StringValue(result.AppType)

	if result.RepoURL != "" {
		state.RepoURL = types.StringValue(result.RepoURL)
	}
	if result.Branch != "" {
		state.Branch = types.StringValue(result.Branch)
	}
	if result.ImageURL != "" {
		state.ImageURL = types.StringValue(result.ImageURL)
	}
	if result.Command != "" {
		state.Command = types.StringValue(result.Command)
	}
	if result.CronSchedule != "" {
		state.CronSchedule = types.StringValue(result.CronSchedule)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *appResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan appResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete and recreate since the API doesn't have a PATCH/PUT for apps
	var state appResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, "/api/v1/apps/"+state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting app for update", err.Error())
		return
	}

	body := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"deploy_source": plan.DeploySource.ValueString(),
		"port":          plan.Port.ValueInt64(),
		"app_type":      plan.AppType.ValueString(),
	}
	if !plan.ProjectID.IsNull() && !plan.ProjectID.IsUnknown() {
		body["project_id"] = plan.ProjectID.ValueString()
	}
	if !plan.RepoURL.IsNull() {
		body["repo_url"] = plan.RepoURL.ValueString()
	}
	if !plan.Branch.IsNull() {
		body["branch"] = plan.Branch.ValueString()
	}
	if !plan.ImageURL.IsNull() {
		body["image_url"] = plan.ImageURL.ValueString()
	}
	if !plan.RegistryUsername.IsNull() {
		body["registry_username"] = plan.RegistryUsername.ValueString()
	}
	if !plan.RegistryPassword.IsNull() {
		body["registry_password"] = plan.RegistryPassword.ValueString()
	}
	if !plan.Command.IsNull() {
		body["command"] = plan.Command.ValueString()
	}
	if !plan.CronSchedule.IsNull() {
		body["cron_schedule"] = plan.CronSchedule.ValueString()
	}

	var result appAPIResponse
	if err := r.client.Post(ctx, "/api/v1/apps", body, &result); err != nil {
		resp.Diagnostics.AddError("Error recreating app", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	plan.ProjectID = types.StringValue(result.ProjectID)
	plan.Status = types.StringValue(result.Status)
	plan.Subdomain = types.StringValue(result.Subdomain)
	plan.URL = types.StringValue(result.URL)
	plan.Framework = types.StringValue(result.Framework)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *appResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state appResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/apps/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting app", err.Error())
		return
	}
}

type appAPIResponse struct {
	ID           string `json:"id"`
	ProjectID    string `json:"project_id"`
	Name         string `json:"name"`
	DeploySource string `json:"deploy_source"`
	RepoURL      string `json:"repo_url"`
	Branch       string `json:"branch"`
	ImageURL     string `json:"image_url"`
	Framework    string `json:"framework"`
	Status       string `json:"status"`
	Subdomain    string `json:"subdomain"`
	URL          string `json:"url"`
	Port         int    `json:"port"`
	AppType      string `json:"app_type"`
	Command      string `json:"command"`
	CronSchedule string `json:"cron_schedule"`
}
