package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdkresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/mariadb-corporation/terraform-provider-skysql-v2/internal/skysql"
	"github.com/mariadb-corporation/terraform-provider-skysql-v2/internal/skysql/provisioning"
	"time"
)

const defaultCreateTimeout = 60 * time.Minute

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithImportState = &ServiceResource{}
var _ resource.ResourceWithConfigure = &ServiceResource{}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

// ServiceResource defines the resource implementation.
type ServiceResource struct {
	client *skysql.Client
}

// ServiceResourceModel describes the resource data model.
type ServiceResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	ProjectID       types.String `tfsdk:"project_id"`
	ServiceType     types.String `tfsdk:"service_type"`
	Provider        types.String `tfsdk:"cloud_provider"`
	Region          types.String `tfsdk:"region"`
	Version         types.String `tfsdk:"version"`
	Nodes           types.Int64  `tfsdk:"nodes"`
	Architecture    types.String `tfsdk:"architecture"`
	Size            types.String `tfsdk:"size"`
	Topology        types.String `tfsdk:"topology"`
	Storage         types.Int64  `tfsdk:"storage"`
	VolumeIOPS      types.Int64  `tfsdk:"volume_iops"`
	SSLEnabled      types.Bool   `tfsdk:"ssl_enabled"`
	NoSQLEnabled    types.Bool   `tfsdk:"nosql_enabled"`
	VolumeType      types.String `tfsdk:"volume_type"`
	WaitForCreation types.Bool   `tfsdk:"wait_for_creation"`
}

// ServiceResourceNamedPortModel is an endpoint port
type ServiceResourceNamedPortModel struct {
	Name string `tfsdk:"name"`
	Port int    `tfsdk:"port"`
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"project_id": schema.StringAttribute{
				Required: true,
			},
			"service_type": schema.StringAttribute{
				Required: true,
			},
			"cloud_provider": schema.StringAttribute{
				Required: true,
			},
			"region": schema.StringAttribute{
				Required: true,
			},
			"version": schema.StringAttribute{
				Required: true,
			},
			"nodes": schema.Int64Attribute{
				Required: true,
			},
			"architecture": schema.StringAttribute{
				Optional: true,
			},
			"size": schema.StringAttribute{
				Required: true,
			},
			"topology": schema.StringAttribute{
				Required: true,
			},
			"storage": schema.Int64Attribute{
				Required: true,
			},
			"volume_iops": schema.Int64Attribute{
				Optional: true,
			},
			"ssl_enabled": schema.BoolAttribute{
				Required: true,
			},
			"nosql_enabled": schema.BoolAttribute{
				Optional: true,
			},
			"volume_type": schema.StringAttribute{
				Required: true,
			},
			"wait_for_creation": schema.BoolAttribute{
				Optional: true,
			},
		},
	}
}

func (r *ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*skysql.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ServiceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createServiceRequest := &provisioning.CreateServiceRequest{
		Name:         data.Name.ValueString(),
		ProjectID:    data.ProjectID.ValueString(),
		ServiceType:  data.ServiceType.ValueString(),
		Provider:     data.Provider.ValueString(),
		Region:       data.Region.ValueString(),
		Version:      data.Version.ValueString(),
		Nodes:        uint(data.Nodes.ValueInt64()),
		Architecture: data.Architecture.ValueString(),
		Size:         data.Size.ValueString(),
		Topology:     data.Topology.ValueString(),
		Storage:      uint(data.Storage.ValueInt64()),
		VolumeIOPS:   uint(data.VolumeIOPS.ValueInt64()),
		SSLEnabled:   data.SSLEnabled.ValueBool(),
		NoSQLEnabled: data.NoSQLEnabled.ValueBool(),
		VolumeType:   data.VolumeType.ValueString(),
	}

	service, err := r.client.CreateService(ctx, createServiceRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating service", err.Error())
		return
	}

	// save into the Terraform state.
	data.ID = types.StringValue(service.ID)

	tflog.Trace(ctx, "created a resource")

	if service.StorageVolume.IOPS > 0 {
		data.VolumeIOPS = types.Int64Value(int64(service.StorageVolume.IOPS))
	} else {
		data.VolumeIOPS = types.Int64Null()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if data.WaitForCreation.ValueBool() {

		err = sdkresource.RetryContext(ctx, defaultCreateTimeout, func() *sdkresource.RetryError {

			service, err := r.client.GetServiceByID(ctx, service.ID)
			if err != nil {
				return sdkresource.NonRetryableError(fmt.Errorf("error retrieving service details: %v", err))
			}

			// block until install is complete
			if service.Status != "ready" && service.Status != "failed" {
				return sdkresource.RetryableError(fmt.Errorf("expected instance to be ready or failed state but was in state %s", service.Status))
			}

			return nil
		})

		if err != nil {
			resp.Diagnostics.AddError("Error creating service", fmt.Sprintf("Unable to create service, got error: %s", err))
		}
	}
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ServiceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.client.GetServiceByID(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Can not find service", err.Error())
		return
	}

	data.Name = types.StringValue(service.Name)
	data.ServiceType = types.StringValue(service.ServiceType)
	data.Provider = types.StringValue(service.Provider)
	data.Region = types.StringValue(service.Region)
	data.Version = types.StringValue(service.Version)
	data.Nodes = types.Int64Value(int64(service.Nodes))
	data.Architecture = types.StringValue(service.Architecture)
	data.Size = types.StringValue(service.Size)
	data.Topology = types.StringValue(service.Topology)
	data.Storage = types.Int64Value(int64(service.StorageVolume.Size))
	if service.StorageVolume.IOPS > 0 {
		data.VolumeIOPS = types.Int64Value(int64(service.StorageVolume.IOPS))
	} else {
		data.VolumeIOPS = types.Int64Null()
	}
	data.VolumeType = types.StringValue(service.StorageVolume.VolumeType)
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan *ServiceResourceModel
	var state *ServiceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Nodes.ValueInt64() != state.Nodes.ValueInt64() {

	}

	if plan.Size.ValueString() != state.Size.ValueString() {

	}

	if plan.Storage.ValueInt64() != state.Storage.ValueInt64() {

	}

	if plan.VolumeIOPS.ValueInt64() != state.VolumeIOPS.ValueInt64() {

	}

	service, err := r.client.GetServiceByID(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Can not find service", err.Error())
		return
	}

	state.Name = types.StringValue(service.Name)
	state.ServiceType = types.StringValue(service.ServiceType)
	state.Provider = types.StringValue(service.Provider)
	state.Region = types.StringValue(service.Region)
	state.Version = types.StringValue(service.Version)
	state.Nodes = types.Int64Value(int64(service.Nodes))
	state.Architecture = types.StringValue(service.Architecture)
	state.Size = types.StringValue(service.Size)
	state.Topology = types.StringValue(service.Topology)
	state.Storage = types.Int64Value(int64(service.StorageVolume.Size))
	state.VolumeIOPS = types.Int64Value(int64(service.StorageVolume.IOPS))
	state.VolumeType = types.StringValue(service.StorageVolume.VolumeType)
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state *ServiceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteServiceByID(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Can not find service", err.Error())
		return
	}
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
