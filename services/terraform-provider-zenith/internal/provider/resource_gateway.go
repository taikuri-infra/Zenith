package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &gatewayResource{}
	_ resource.ResourceWithConfigure = &gatewayResource{}
)

type gatewayResource struct {
	client *ZenithClient
}

type gatewayResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	ProjectID  types.String `tfsdk:"project_id"`
	Status     types.String `tfsdk:"status"`
	Endpoint   types.String `tfsdk:"endpoint"`
	Slug       types.String `tfsdk:"slug"`
	RouteCount types.Int64  `tfsdk:"route_count"`
	Routes     types.List   `tfsdk:"routes"`
}

func NewGatewayResource() resource.Resource {
	return &gatewayResource{}
}

func (r *gatewayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gateway"
}

func (r *gatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zenith API Gateway with routes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Gateway ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Gateway name.",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "Project ID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Gateway status.",
				Computed:    true,
			},
			"endpoint": schema.StringAttribute{
				Description: "Gateway endpoint URL.",
				Computed:    true,
			},
			"slug": schema.StringAttribute{
				Description: "URL-friendly slug.",
				Computed:    true,
			},
			"route_count": schema.Int64Attribute{
				Description: "Number of routes.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"routes": schema.ListNestedBlock{
				Description: "Gateway routes.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Route ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Route name.",
							Required:    true,
						},
						"path": schema.StringAttribute{
							Description: "URL path pattern (e.g., '/api/v1/*').",
							Required:    true,
						},
						"methods": schema.ListAttribute{
							Description: "HTTP methods (GET, POST, PUT, DELETE, etc.).",
							Required:    true,
							ElementType: types.StringType,
						},
						"app_id": schema.StringAttribute{
							Description: "Target application ID.",
							Required:    true,
						},
						"strip_prefix": schema.BoolAttribute{
							Description: "Strip path prefix from upstream request.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"auth": schema.StringAttribute{
							Description: "Authentication type: 'none', 'jwt', 'key-auth', 'oidc'.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("none"),
						},
						"auth_pool_id": schema.StringAttribute{
							Description: "Auth pool ID for OIDC authentication.",
							Optional:    true,
						},
						"priority": schema.Int64Attribute{
							Description: "Route priority (higher = matched first).",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(0),
						},
						"status": schema.StringAttribute{
							Description: "Route status.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *gatewayResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *gatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.ProjectID.IsNull() && !plan.ProjectID.IsUnknown() {
		body["project_id"] = plan.ProjectID.ValueString()
	}

	var gwResult gatewayAPIResponse
	if err := r.client.Post(ctx, "/api/v1/gateways", body, &gwResult); err != nil {
		resp.Diagnostics.AddError("Error creating gateway", err.Error())
		return
	}

	plan.ID = types.StringValue(gwResult.ID)
	plan.ProjectID = types.StringValue(gwResult.ProjectID)
	plan.Status = types.StringValue(gwResult.Status)
	plan.Endpoint = types.StringValue(gwResult.Endpoint)
	plan.Slug = types.StringValue(gwResult.Slug)

	// Create routes
	routeResults := r.createRoutes(ctx, gwResult.ID, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.RouteCount = types.Int64Value(int64(len(routeResults)))
	plan.Routes = buildRouteListValue(routeResults)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *gatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var gwResult struct {
		Gateway gatewayAPIResponse   `json:"gateway"`
		Routes  []gatewayRouteAPIRes `json:"routes"`
	}
	if err := r.client.Get(ctx, "/api/v1/gateways/"+state.ID.ValueString(), &gwResult); err != nil {
		resp.Diagnostics.AddError("Error reading gateway", err.Error())
		return
	}

	state.Name = types.StringValue(gwResult.Gateway.Name)
	state.ProjectID = types.StringValue(gwResult.Gateway.ProjectID)
	state.Status = types.StringValue(gwResult.Gateway.Status)
	state.Endpoint = types.StringValue(gwResult.Gateway.Endpoint)
	state.Slug = types.StringValue(gwResult.Gateway.Slug)
	state.RouteCount = types.Int64Value(int64(gwResult.Gateway.RouteCount))

	state.Routes = buildRouteListValue(gwResult.Routes)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *gatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan gatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state gatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gwID := state.ID.ValueString()

	// Update gateway name
	updateBody := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	var gwResult gatewayAPIResponse
	if err := r.client.Put(ctx, "/api/v1/gateways/"+gwID, updateBody, &gwResult); err != nil {
		resp.Diagnostics.AddError("Error updating gateway", err.Error())
		return
	}

	plan.ID = types.StringValue(gwResult.ID)
	plan.ProjectID = types.StringValue(gwResult.ProjectID)
	plan.Status = types.StringValue(gwResult.Status)
	plan.Endpoint = types.StringValue(gwResult.Endpoint)
	plan.Slug = types.StringValue(gwResult.Slug)

	// Delete existing routes
	var existingRoutes []gatewayRouteAPIRes
	if err := r.client.Get(ctx, "/api/v1/gateways/"+gwID+"/routes", &existingRoutes); err == nil {
		for _, route := range existingRoutes {
			_ = r.client.Delete(ctx, fmt.Sprintf("/api/v1/gateways/%s/routes/%s", gwID, route.ID))
		}
	}

	// Recreate routes from plan
	routeResults := r.createRoutes(ctx, gwID, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.RouteCount = types.Int64Value(int64(len(routeResults)))
	plan.Routes = buildRouteListValue(routeResults)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *gatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/gateways/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting gateway", err.Error())
		return
	}
}

func (r *gatewayResource) createRoutes(ctx context.Context, gwID string, plan *gatewayResourceModel, diags *diag.Diagnostics) []gatewayRouteAPIRes {
	if plan.Routes.IsNull() || plan.Routes.IsUnknown() {
		return nil
	}

	var planRoutes []gatewayRouteModel
	diags.Append(plan.Routes.ElementsAs(ctx, &planRoutes, false)...)
	if diags.HasError() {
		return nil
	}

	var results []gatewayRouteAPIRes
	for _, route := range planRoutes {
		var methods []string
		diags.Append(route.Methods.ElementsAs(ctx, &methods, false)...)
		if diags.HasError() {
			return nil
		}

		body := map[string]interface{}{
			"name":         route.Name.ValueString(),
			"path":         route.Path.ValueString(),
			"methods":      methods,
			"app_id":       route.AppID.ValueString(),
			"strip_prefix": route.StripPrefix.ValueBool(),
			"auth":         route.Auth.ValueString(),
			"priority":     route.Priority.ValueInt64(),
		}
		if !route.AuthPoolID.IsNull() {
			body["auth_pool_id"] = route.AuthPoolID.ValueString()
		}

		var routeResult gatewayRouteAPIRes
		if err := r.client.Post(ctx, "/api/v1/gateways/"+gwID+"/routes", body, &routeResult); err != nil {
			diags.AddError("Error creating gateway route", err.Error())
			return nil
		}
		results = append(results, routeResult)
	}
	return results
}

func buildRouteListValue(routes []gatewayRouteAPIRes) types.List {
	if len(routes) == 0 {
		return types.ListNull(routeObjectType())
	}

	routeObjects := make([]attr.Value, len(routes))
	for i, route := range routes {
		methodValues := make([]attr.Value, len(route.Methods))
		for j, m := range route.Methods {
			methodValues[j] = types.StringValue(m)
		}
		methodsList, _ := types.ListValue(types.StringType, methodValues)

		authPoolID := types.StringNull()
		if route.AuthPoolID != "" {
			authPoolID = types.StringValue(route.AuthPoolID)
		}

		routeObj, _ := types.ObjectValue(
			routeAttrTypes(),
			map[string]attr.Value{
				"id":           types.StringValue(route.ID),
				"name":         types.StringValue(route.Name),
				"path":         types.StringValue(route.Path),
				"methods":      methodsList,
				"app_id":       types.StringValue(route.AppID),
				"strip_prefix": types.BoolValue(route.StripPrefix),
				"auth":         types.StringValue(route.Auth),
				"auth_pool_id": authPoolID,
				"priority":     types.Int64Value(int64(route.Priority)),
				"status":       types.StringValue(route.Status),
			},
		)
		routeObjects[i] = routeObj
	}

	routesList, _ := types.ListValue(routeObjectType(), routeObjects)
	return routesList
}

func routeAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":           types.StringType,
		"name":         types.StringType,
		"path":         types.StringType,
		"methods":      types.ListType{ElemType: types.StringType},
		"app_id":       types.StringType,
		"strip_prefix": types.BoolType,
		"auth":         types.StringType,
		"auth_pool_id": types.StringType,
		"priority":     types.Int64Type,
		"status":       types.StringType,
	}
}

func routeObjectType() attr.Type {
	return types.ObjectType{AttrTypes: routeAttrTypes()}
}

type gatewayRouteModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Path        types.String `tfsdk:"path"`
	Methods     types.List   `tfsdk:"methods"`
	AppID       types.String `tfsdk:"app_id"`
	StripPrefix types.Bool   `tfsdk:"strip_prefix"`
	Auth        types.String `tfsdk:"auth"`
	AuthPoolID  types.String `tfsdk:"auth_pool_id"`
	Priority    types.Int64  `tfsdk:"priority"`
	Status      types.String `tfsdk:"status"`
}

type gatewayAPIResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ProjectID  string `json:"project_id"`
	Slug       string `json:"slug"`
	Status     string `json:"status"`
	Endpoint   string `json:"endpoint"`
	RouteCount int    `json:"route_count"`
}

type gatewayRouteAPIRes struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Path        string          `json:"path"`
	Methods     []string        `json:"methods"`
	AppID       string          `json:"app_id"`
	StripPrefix bool            `json:"strip_prefix"`
	Auth        string          `json:"auth"`
	AuthPoolID  string          `json:"auth_pool_id"`
	Priority    int             `json:"priority"`
	Status      string          `json:"status"`
	Plugins     json.RawMessage `json:"plugins,omitempty"`
}
