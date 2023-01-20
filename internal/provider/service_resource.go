package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdkresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/mariadb-corporation/terraform-provider-skysql-beta/internal/skysql"
	"github.com/mariadb-corporation/terraform-provider-skysql-beta/internal/skysql/provisioning"
	"reflect"
	"regexp"
	"time"
)

const defaultCreateTimeout = 60 * time.Minute
const defaultDeleteTimeout = 60 * time.Minute
const defaultUpdateTimeout = 60 * time.Minute

var rxServiceName = regexp.MustCompile("(^[a-z][a-z0-9-]+$)")

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithImportState = &ServiceResource{}
var _ resource.ResourceWithConfigure = &ServiceResource{}
var _ resource.ResourceWithModifyPlan = &ServiceResource{}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

// ServiceResource defines the resource implementation.
type ServiceResource struct {
	client *skysql.Client
}

// ServiceResourceModel describes the resource data model.
type ServiceResourceModel struct {
	ID                 types.String   `tfsdk:"id"`
	Name               types.String   `tfsdk:"name"`
	ProjectID          types.String   `tfsdk:"project_id"`
	ServiceType        types.String   `tfsdk:"service_type"`
	Provider           types.String   `tfsdk:"cloud_provider"`
	Region             types.String   `tfsdk:"region"`
	Version            types.String   `tfsdk:"version"`
	Nodes              types.Int64    `tfsdk:"nodes"`
	Architecture       types.String   `tfsdk:"architecture"`
	Size               types.String   `tfsdk:"size"`
	Topology           types.String   `tfsdk:"topology"`
	Storage            types.Int64    `tfsdk:"storage"`
	VolumeIOPS         types.Int64    `tfsdk:"volume_iops"`
	SSLEnabled         types.Bool     `tfsdk:"ssl_enabled"`
	NoSQLEnabled       types.Bool     `tfsdk:"nosql_enabled"`
	VolumeType         types.String   `tfsdk:"volume_type"`
	WaitForCreation    types.Bool     `tfsdk:"wait_for_creation"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
	Mechanism          types.String   `tfsdk:"endpoint_mechanism"`
	AllowedAccounts    types.List     `tfsdk:"endpoint_allowed_accounts"`
	WaitForDeletion    types.Bool     `tfsdk:"wait_for_deletion"`
	ReplicationEnabled types.Bool     `tfsdk:"replication_enabled"`
	PrimaryHost        types.String   `tfsdk:"primary_host"`
	IsActive           types.Bool     `tfsdk:"is_active"`
	WaitForUpdate      types.Bool     `tfsdk:"wait_for_update"`
}

// ServiceResourceNamedPortModel is an endpoint port
type ServiceResourceNamedPortModel struct {
	Name types.String `tfsdk:"name"`
	Port types.Int64  `tfsdk:"port"`
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a service in SkySQL",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required: false,
				Optional: false,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The ID of the service",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the service",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 24),
					stringvalidator.RegexMatches(
						rxServiceName,
						"must start from a lowercase letter and contain only lowercase letters, numbers and hyphens",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    false,
				Optional:    true,
				Description: "The ID of the project to create the service in",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of service to create. Valid values are: analytical or transactional",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cloud_provider": schema.StringAttribute{
				Required:    true,
				Description: "The cloud provider to create the service in. Valid values are: aws or gcp",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "The region to create the service in. Value should be valid for a specific cloud provider",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Required:    true,
				Description: "The server version",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"nodes": schema.Int64Attribute{
				Required:    true,
				Description: "The number of nodes",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"architecture": schema.StringAttribute{
				Optional:    true,
				Description: "The architecture of the service. Valid values are: amd64 or arm64",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"size": schema.StringAttribute{
				Required:    true,
				Description: "The size of the service. Valid values are: sky-2x4, sky-2x8 etc",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"topology": schema.StringAttribute{
				Required:    true,
				Description: "The topology of the service. Valid values are: masterslave, standalone, xpand-direct, columnstore, lakehouse",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"storage": schema.Int64Attribute{
				Required:    true,
				Description: "The storage size in GB. Valid values are: 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"volume_iops": schema.Int64Attribute{
				Optional:    true,
				Description: "The volume IOPS. This is only applicable for AWS",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"ssl_enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Whether to enable SSL. Valid values are: true or false",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"nosql_enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to enable NoSQL. Valid values are: true or false",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"volume_type": schema.StringAttribute{
				Optional:    true,
				Description: "The volume type. Valid values are: gp2 and io1. This is only applicable for AWS",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"wait_for_creation": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to wait for the service to be created. Valid values are: true or false",
			},
			"endpoint_mechanism": schema.StringAttribute{
				Optional:    true,
				Description: "The endpoint mechanism to use. Valid values are: privatelink or nlb",
			},
			"endpoint_allowed_accounts": schema.ListAttribute{
				Optional:    true,
				Description: "The list of cloud accounts (aws account ids or gcp projects) that are allowed to access the service",
				ElementType: types.StringType,
			},
			"wait_for_deletion": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to wait for the service to be deleted. Valid values are: true or false",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"replication_enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to enable global replication. Valid values are: true or false. Works for xpand-direct topology only",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"primary_host": schema.StringAttribute{
				Optional:    true,
				Description: "The primary host of the service",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the service is active",
			},
			"wait_for_update": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to wait for the service to be updated. Valid values are: true or false",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Delete: true,
				Update: true,
			}),
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
		Name:               data.Name.ValueString(),
		ProjectID:          data.ProjectID.ValueString(),
		ServiceType:        data.ServiceType.ValueString(),
		Provider:           data.Provider.ValueString(),
		Region:             data.Region.ValueString(),
		Version:            data.Version.ValueString(),
		Nodes:              uint(data.Nodes.ValueInt64()),
		Architecture:       data.Architecture.ValueString(),
		Size:               data.Size.ValueString(),
		Topology:           data.Topology.ValueString(),
		Storage:            uint(data.Storage.ValueInt64()),
		VolumeIOPS:         uint(data.VolumeIOPS.ValueInt64()),
		SSLEnabled:         data.SSLEnabled.ValueBool(),
		NoSQLEnabled:       data.NoSQLEnabled.ValueBool(),
		VolumeType:         data.VolumeType.ValueString(),
		Mechanism:          data.Mechanism.ValueString(),
		ReplicationEnabled: data.ReplicationEnabled.ValueBool(),
		PrimaryHost:        data.PrimaryHost.ValueString(),
	}

	diag := data.AllowedAccounts.ElementsAs(ctx, &createServiceRequest.AllowedAccounts, false)
	if diag.HasError() {
		return
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
	if service.StorageVolume.VolumeType != "" {
		data.VolumeType = types.StringValue(service.StorageVolume.VolumeType)
	} else {
		data.VolumeType = types.StringNull()
	}

	data.IsActive = types.BoolValue(service.IsActive)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.WaitForCreation.ValueBool() {

		createTimeout, diagsErr := data.Timeouts.Create(ctx, defaultCreateTimeout)
		if diagsErr != nil {
			diagsErr.AddError("Error creating service", fmt.Sprintf("Unable to create service, got error: %s", err))
			resp.Diagnostics.Append(diagsErr...)
		}

		err = sdkresource.RetryContext(ctx, createTimeout, func() *sdkresource.RetryError {

			service, err := r.client.GetServiceByID(ctx, service.ID)
			if err != nil {
				return sdkresource.NonRetryableError(fmt.Errorf("error retrieving service details: %v", err))
			}

			if service.Status != "ready" && service.Status != "failed" {
				return sdkresource.RetryableError(fmt.Errorf("expected instance to be ready or failed state but was in state %s", service.Status))
			}

			if service.Status == "failed" {
				return sdkresource.NonRetryableError(errors.New("service creation failed"))
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

	err := r.readServiceState(ctx, data)
	if err != nil {
		if errors.Is(err, skysql.ErrorServiceNotFound) {
			tflog.Warn(ctx, "SkySQL service not found, removing from state", map[string]interface{}{
				"id": data.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)

			return
		}
		resp.Diagnostics.AddError("Can not read service", err.Error())
		return
	}
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ServiceResource) readServiceState(ctx context.Context, data *ServiceResourceModel) error {
	service, err := r.client.GetServiceByID(ctx, data.ID.ValueString())
	if err != nil {
		return err
	}
	data.ID = types.StringValue(service.ID)
	data.ServiceType = types.StringValue(service.ServiceType)
	data.Provider = types.StringValue(service.Provider)
	data.Region = types.StringValue(service.Region)
	data.Version = types.StringValue(service.Version)
	data.Nodes = types.Int64Value(int64(service.Nodes))
	data.Architecture = types.StringValue(service.Architecture)
	data.Size = types.StringValue(service.Size)
	data.Topology = types.StringValue(service.Topology)
	data.Storage = types.Int64Value(int64(service.StorageVolume.Size))
	if !data.VolumeIOPS.IsNull() && service.StorageVolume.IOPS > 0 {
		data.VolumeIOPS = types.Int64Value(int64(service.StorageVolume.IOPS))
	} else {
		data.VolumeIOPS = types.Int64Null()
	}
	if !data.VolumeType.IsNull() {
		data.VolumeType = types.StringValue(service.StorageVolume.VolumeType)
	}
	if !data.ReplicationEnabled.IsNull() {
		data.ReplicationEnabled = types.BoolValue(service.ReplicationEnabled)
	}
	if !data.PrimaryHost.IsNull() {
		data.PrimaryHost = types.StringValue(service.PrimaryHost)
	}
	data.IsActive = types.BoolValue(service.IsActive)

	return nil
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

	err := r.readServiceState(ctx, state)
	if err != nil {
		if errors.Is(err, skysql.ErrorServiceNotFound) {
			tflog.Warn(ctx, "SkySQL service not found, removing from state", map[string]interface{}{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)

			return
		}
		resp.Diagnostics.AddError("Can not read service", err.Error())
		return
	}

	state.WaitForUpdate = plan.WaitForUpdate
	state.WaitForCreation = plan.WaitForCreation
	state.WaitForDeletion = plan.WaitForDeletion
	state.Timeouts = plan.Timeouts
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.IsActive.IsUnknown() && plan.IsActive.ValueBool() != state.IsActive.ValueBool() {
		tflog.Info(ctx, "Updating service active state", map[string]interface{}{
			"id":        state.ID.ValueString(),
			"is_active": plan.IsActive.ValueBool(),
		})
		err = r.client.SetServicePowerState(ctx, state.ID.ValueString(), plan.IsActive.ValueBool())
		if err != nil {
			resp.Diagnostics.AddError("Can not update service", err.Error())
			return
		}
		state.IsActive = plan.IsActive
		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		r.waitForUpdate(ctx, state, err, resp)
	}

	var planAllowedAccounts []string
	diag := plan.AllowedAccounts.ElementsAs(ctx, &planAllowedAccounts, false)
	if diag.HasError() {
		return
	}

	var stateAllowedAccounts []string
	diag = state.AllowedAccounts.ElementsAs(ctx, &stateAllowedAccounts, false)
	if diag.HasError() {
		return
	}

	if !(reflect.DeepEqual(plan.Mechanism.ValueString(), state.Mechanism.ValueString()) ||
		reflect.DeepEqual(planAllowedAccounts, stateAllowedAccounts)) {
		tflog.Info(ctx, "Updating service allowed accounts", map[string]interface{}{
			"id": state.ID.ValueString(),
		})

		visibility := "public"
		if plan.Mechanism.ValueString() == "privatelink" {
			visibility = "private"
		}

		_, err := r.client.ModifyServiceEndpoints(ctx,
			state.ID.ValueString(),
			plan.Mechanism.ValueString(),
			planAllowedAccounts,
			visibility)
		if err != nil {
			resp.Diagnostics.AddError("Can not update service", err.Error())
			return
		}

		state.AllowedAccounts = plan.AllowedAccounts
		state.Mechanism = plan.Mechanism

		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		r.waitForUpdate(ctx, state, err, resp)
		return
	}
}

var serviceUpdateWaitStates = []string{"ready", "failed", "stopped"}

func (r *ServiceResource) waitForUpdate(ctx context.Context, state *ServiceResourceModel, err error, resp *resource.UpdateResponse) {
	if state.WaitForUpdate.ValueBool() {
		err = sdkresource.RetryContext(ctx, defaultUpdateTimeout, func() *sdkresource.RetryError {
			service, err := r.client.GetServiceByID(ctx, state.ID.ValueString())
			if err != nil {
				return sdkresource.NonRetryableError(fmt.Errorf("error retrieving service details: %v", err))
			}

			if Contains[string](serviceUpdateWaitStates, service.Status) {
				return nil
			}

			if service.Status == "failed" {
				return sdkresource.NonRetryableError(errors.New("service creation failed"))
			}

			return sdkresource.RetryableError(fmt.Errorf("expected instance to be ready or failed or stopped state but was in state %s", service.Status))
		})

		if err != nil {
			resp.Diagnostics.AddError("Error updating service", fmt.Sprintf("Unable to update service, got error: %s", err))
		}
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
		if errors.Is(err, skysql.ErrorServiceNotFound) {
			tflog.Warn(ctx, "SkySQL service not found, removing from state", map[string]interface{}{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)

			return
		}
		return
	}
	if state.WaitForDeletion.ValueBool() {

		deleteTimeout, diagsErr := state.Timeouts.Delete(ctx, defaultDeleteTimeout)
		if diagsErr != nil {
			diagsErr.AddError("Error deleting service", fmt.Sprintf("Unable to delete service, got error: %s", err))
			resp.Diagnostics.Append(diagsErr...)
		}

		err = sdkresource.RetryContext(ctx, deleteTimeout, func() *sdkresource.RetryError {

			service, err := r.client.GetServiceByID(ctx, state.ID.ValueString())
			if err != nil {
				if errors.Is(err, skysql.ErrorServiceNotFound) {
					return nil
				}
				return sdkresource.NonRetryableError(fmt.Errorf("error retrieving service details: %v", err))
			}

			return sdkresource.RetryableError(fmt.Errorf("expected that the instance was deleted, but it was in state %s", service.Status))
		})

		if err != nil {
			resp.Diagnostics.AddError("Error delete service", fmt.Sprintf("Unable to delete service, got error: %s", err))
		}
	}
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ServiceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Plan does not need to be modified when the resource is being destroyed.
	if req.Plan.Raw.IsNull() {
		return
	}

	var config *ServiceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if config.Provider.ValueString() == "aws" {
		if !config.VolumeIOPS.IsNull() && config.VolumeType.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("volume_type"),
				"volume_type is require",
				"volume_type is required when volume_iops is set. "+
					"Use: io1 for volume_type if volume_iops is set")
			return
		}
		if !config.VolumeIOPS.IsNull() &&
			config.VolumeType.ValueString() != "io1" {
			resp.Diagnostics.AddAttributeError(path.Root("volume_type"),
				"volume_type must be io1 when you want to set IOPS",
				"Use: io1 for volume_type if volume_iops is set")
			return
		}
	} else {
		if !config.VolumeType.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("volume_type"),
				fmt.Sprintf("volume_type is not supported for %q provider", config.Provider.ValueString()),
				fmt.Sprintf("Volume type is not supported for %q provider", config.Provider.ValueString()))
			return
		}
		if !config.VolumeIOPS.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("volume_iops"),
				fmt.Sprintf("volume_iops is not supported for %q provider", config.Provider.ValueString()),
				fmt.Sprintf("Volume IOPS are not supported for %q provider", config.Provider.ValueString()))
			return
		}
	}
	resp.Diagnostics.Append(req.Plan.Set(ctx, &config)...)
}
