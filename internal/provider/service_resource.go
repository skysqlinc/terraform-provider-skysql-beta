package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdkresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql/provisioning"
)

const defaultCreateTimeout = 60 * time.Minute
const defaultDeleteTimeout = 60 * time.Minute
const defaultUpdateTimeout = 60 * time.Minute
const visibilityPrivate = "private"
const visibilityPublic = "public"

var rxServiceName = regexp.MustCompile("(^[a-z][a-z0-9-]+$)")

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithImportState = &ServiceResource{}
var _ resource.ResourceWithConfigure = &ServiceResource{}
var _ resource.ResourceWithModifyPlan = &ServiceResource{}
var _ resource.ResourceWithUpgradeState = &ServiceResource{}

var allowListElementType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"ip":      types.StringType,
		"comment": types.StringType,
	},
}

var privateConnectMechanisms = []string{"privateconnect", "privatelink"}

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
	VolumeThroughput   types.Int64    `tfsdk:"volume_throughput"`
	SSLEnabled         types.Bool     `tfsdk:"ssl_enabled"`
	NoSQLEnabled       types.Bool     `tfsdk:"nosql_enabled"`
	VolumeType         types.String   `tfsdk:"volume_type"`
	WaitForCreation    types.Bool     `tfsdk:"wait_for_creation"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
	Mechanism          types.String   `tfsdk:"endpoint_mechanism"`
	AllowedAccounts    types.List     `tfsdk:"endpoint_allowed_accounts"`
	EndpointService    types.String   `tfsdk:"endpoint_service"`
	WaitForDeletion    types.Bool     `tfsdk:"wait_for_deletion"`
	ReplicationEnabled types.Bool     `tfsdk:"replication_enabled"`
	PrimaryHost        types.String   `tfsdk:"primary_host"`
	IsActive           types.Bool     `tfsdk:"is_active"`
	WaitForUpdate      types.Bool     `tfsdk:"wait_for_update"`
	DeletionProtection types.Bool     `tfsdk:"deletion_protection"`
	AllowList          types.List     `tfsdk:"allow_list"`
	MaxscaleNodes      types.Int64    `tfsdk:"maxscale_nodes"`
	MaxscaleSize       types.String   `tfsdk:"maxscale_size"`
	FQDN               types.String   `tfsdk:"fqdn"`
	AvailabilityZone   types.String   `tfsdk:"availability_zone"`
}

// ServiceResourceNamedPortModel is an endpoint port
type ServiceResourceNamedPortModel struct {
	Name types.String `tfsdk:"name"`
	Port types.Int64  `tfsdk:"port"`
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

var serviceResourceSchemaV0 = schema.Schema{
	Description: "Creates and manages a service in SkySQL",
	Version:     1,
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
			Optional:    true,
			Computed:    true,
			Description: "The software version",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier.RequiresReplace(),
			},
		},
		"nodes": schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: "The number of nodes",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"architecture": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The architecture of the service. Valid values are: amd64 or arm64",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier.RequiresReplace(),
			},
		},
		"size": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The size of the service. Valid values are: sky-2x4, sky-2x8 etc",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"topology": schema.StringAttribute{
			Required:    true,
			Description: "The topology of the service. Valid values are: es-single, es-replica, xpand, csdw and sa",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"storage": schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: "The storage size in GB. Valid values are: 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"volume_iops": schema.Int64Attribute{
			Optional:    true,
			Description: "The volume IOPS. This is only applicable for AWS",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"volume_throughput": schema.Int64Attribute{
			Optional:    true,
			Description: "The volume Throughput. This is only applicable for AWS",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"ssl_enabled": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether to enable SSL. Valid values are: true or false",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.RequiresReplace(),
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"nosql_enabled": schema.BoolAttribute{
			Optional:    true,
			Description: "Whether to enable NoSQL. Valid values are: true or false",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.RequiresReplace(),
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"volume_type": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The volume type. Valid values are: gp3, gp2, and io1. This is only applicable for AWS",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier.RequiresReplaceIf(
					func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
						if !req.PlanValue.IsUnknown() {
							resp.RequiresReplace = true
						}
					},
					"If the value of this attribute changes, Terraform will destroy and recreate the resource.",
					"If the value of this attribute changes, Terraform will destroy and recreate the resource.",
				),
			},
		},
		"wait_for_creation": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether to wait for the service to be created. Valid values are: true or false",
			PlanModifiers: []planmodifier.Bool{
				boolDefault(true),
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"endpoint_mechanism": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The endpoint mechanism to use. Valid values are: privateconnect or nlb",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"endpoint_allowed_accounts": schema.ListAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The list of cloud accounts (aws account ids or gcp projects) that are allowed to access the service",
			ElementType: types.StringType,
			Default:     listdefault.StaticValue(types.ListNull(types.StringType)),
		},
		"wait_for_deletion": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether to wait for the service to be deleted. Valid values are: true or false",
			PlanModifiers: []planmodifier.Bool{
				boolDefault(true),
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"replication_enabled": schema.BoolAttribute{
			Optional:    true,
			Description: "Whether to enable global replication. Valid values are: true or false. Works for xpand-direct topology only",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.RequiresReplace(),
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"primary_host": schema.StringAttribute{
			Optional:    true,
			Description: "The primary host of the service",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"is_active": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether the service is active",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"wait_for_update": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether to wait for the service to be updated. Valid values are: true or false",
			PlanModifiers: []planmodifier.Bool{
				boolDefault(true),
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"deletion_protection": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether to enable deletion protection. Valid values are: true or false. Default is true",
			PlanModifiers: []planmodifier.Bool{
				boolDefault(true),
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"allow_list": schema.ListNestedAttribute{
			Required:    false,
			Computed:    true,
			Optional:    true,
			Description: "The list of IP addresses with comments to allow access to the service",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"ip": schema.StringAttribute{
						Required:    true,
						Description: "The IP address to allow access to the service. The IP must be in a valid CIDR format",
						Validators: []validator.String{
							allowListIPValidator{},
						},
					},
					"comment": schema.StringAttribute{
						Optional:    true,
						Description: "A comment to describe the IP address",
					},
				},
			},
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
		},
		"maxscale_nodes": schema.Int64Attribute{
			Optional:    true,
			Description: "The number of MaxScale nodes",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"maxscale_size": schema.StringAttribute{
			Optional:    true,
			Description: "The size of the MaxScale nodes. Valid values are: sky-2x4, sky-2x8 etc",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"fqdn": schema.StringAttribute{
			Required: false,
			Optional: false,
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
			Description: "The fully qualified domain name of the service. The FQDN is only available when the service is in the ready state",
		},
		"endpoint_service": schema.StringAttribute{
			Required:    false,
			Optional:    false,
			Computed:    true,
			Description: "The endpoint service name of the service, when mechanism is a privateconnect.",
		},
		"availability_zone": schema.StringAttribute{
			Required:    false,
			Optional:    true,
			Computed:    true,
			Description: "The availability zone of the service",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier.RequiresReplace(),
			},
		},
	},
	Blocks: map[string]schema.Block{
		"timeouts": timeouts.Block(context.Background(), timeouts.Opts{
			Create: true,
			Delete: true,
			Update: true,
		}),
	},
}

func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = serviceResourceSchemaV0
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
	var state *ServiceResourceModel

	// Read Terraform state into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createServiceRequest := &provisioning.CreateServiceRequest{
		Name:               state.Name.ValueString(),
		ProjectID:          state.ProjectID.ValueString(),
		ServiceType:        state.ServiceType.ValueString(),
		Provider:           state.Provider.ValueString(),
		Region:             state.Region.ValueString(),
		Version:            state.Version.ValueString(),
		Nodes:              uint(state.Nodes.ValueInt64()),
		Architecture:       state.Architecture.ValueString(),
		Size:               state.Size.ValueString(),
		Topology:           state.Topology.ValueString(),
		Storage:            uint(state.Storage.ValueInt64()),
		VolumeIOPS:         uint(state.VolumeIOPS.ValueInt64()),
		VolumeThroughput:   uint(state.VolumeThroughput.ValueInt64()),
		SSLEnabled:         state.SSLEnabled.ValueBool(),
		NoSQLEnabled:       state.NoSQLEnabled.ValueBool(),
		VolumeType:         state.VolumeType.ValueString(),
		Mechanism:          state.Mechanism.ValueString(),
		ReplicationEnabled: state.ReplicationEnabled.ValueBool(),
		PrimaryHost:        state.PrimaryHost.ValueString(),
		MaxscaleNodes:      uint(state.MaxscaleNodes.ValueInt64()),
		AvailabilityZone:   state.AvailabilityZone.ValueString(),
	}

	if !Contains[string]([]string{"gcp", "aws", "azure"}, createServiceRequest.Provider) {
		resp.Diagnostics.AddAttributeError(path.Root("provider"),
			"Invalid provider value",
			fmt.Sprintf("The %q is an invalid value. Allowed values: aws, gcp, azure", createServiceRequest.Provider))
	}

	if !state.MaxscaleSize.IsUnknown() && !state.MaxscaleSize.IsNull() && len(state.MaxscaleSize.ValueString()) > 0 {
		createServiceRequest.MaxscaleSize = toPtr[string](state.MaxscaleSize.ValueString())
	} else {
		createServiceRequest.MaxscaleSize = nil
	}

	b, err := json.Marshal(createServiceRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal create service request", err.Error())
		return
	}
	tflog.Debug(ctx, string(b))

	if !state.AllowList.IsUnknown() {
		var allowList []AllowListModel
		diags := state.AllowList.ElementsAs(ctx, &allowList, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		for _, allowListItem := range allowList {
			createServiceRequest.AllowList = append(createServiceRequest.AllowList, provisioning.AllowListItem{
				IPAddress: allowListItem.IPAddress.ValueString(),
				Comment:   allowListItem.Comment.ValueString(),
			})
		}
	}

	if Contains[string](privateConnectMechanisms, state.Mechanism.ValueString()) {
		diags := state.AllowedAccounts.ElementsAs(ctx, &createServiceRequest.AllowedAccounts, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	service, err := r.client.CreateService(ctx, createServiceRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating service", err.Error())
		return
	}

	// save into the Terraform state.
	state.ID = types.StringValue(service.ID)
	state.Name = types.StringValue(service.Name)
	state.FQDN = types.StringValue(service.FQDN)

	tflog.Trace(ctx, "created a resource")

	if service.StorageVolume.IOPS > 0 {
		state.VolumeIOPS = types.Int64Value(int64(service.StorageVolume.IOPS))
	} else {
		state.VolumeIOPS = types.Int64Null()
	}
	if service.StorageVolume.Throughput > 0 {
		state.VolumeThroughput = types.Int64Value(int64(service.StorageVolume.Throughput))
	} else {
		state.VolumeThroughput = types.Int64Null()
	}
	if service.StorageVolume.VolumeType != "" {
		state.VolumeType = types.StringValue(service.StorageVolume.VolumeType)
	} else {
		state.VolumeType = types.StringNull()
	}

	state.IsActive = types.BoolValue(service.IsActive)
	state.Architecture = types.StringValue(service.Architecture)
	state.Nodes = types.Int64Value(int64(service.Nodes))
	state.Size = types.StringValue(service.Size)
	state.Version = types.StringValue(service.Version)
	state.Storage = types.Int64Value(int64(service.StorageVolume.Size))
	state.SSLEnabled = types.BoolValue(service.SSLEnabled)
	state.AvailabilityZone = types.StringValue(service.AvailabilityZone)
	if len(service.Endpoints) > 0 {
		state.Mechanism = types.StringValue(service.Endpoints[0].Mechanism)
		r.setAllowAccounts(ctx, state, service.Endpoints[0].AllowedAccounts)
		state.AllowList, _ = r.allowListToListType(ctx, service.Endpoints[0].AllowList)
		state.EndpointService = types.StringValue(service.Endpoints[0].EndpointService)
	}
	if !(state.MaxscaleSize.IsUnknown() || state.MaxscaleSize.IsNull()) && service.MaxscaleSize != nil {
		state.MaxscaleSize = types.StringValue(*service.MaxscaleSize)
	}
	if !(state.MaxscaleNodes.IsUnknown() || state.MaxscaleNodes.IsNull()) {
		state.MaxscaleNodes = types.Int64Value(int64(service.MaxscaleNodes))
	}
	// Save state into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.WaitForCreation.ValueBool() {
		createTimeout, diagsErr := state.Timeouts.Create(ctx, defaultCreateTimeout)
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
			return
		}
		var plan *ServiceResourceModel
		// Read Terraform state into the model
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

		if resp.Diagnostics.HasError() {
			return
		}
		r.readServiceState(ctx, state)
		r.updateAllowedAccountsState(plan, state)
		r.updateAllowListState(plan, state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	}
}

func (r *ServiceResource) setAllowAccounts(ctx context.Context, data *ServiceResourceModel, allowedAccounts []string) {
	data.AllowedAccounts, _ = types.ListValueFrom(ctx, types.StringType, allowedAccounts)
}

func (r *ServiceResource) allowListToListType(ctx context.Context, allowList []provisioning.AllowListItem) (types.List, diag.Diagnostics) {
	if allowList == nil {
		return types.ListNull(allowListElementType), nil
	}
	allowListModels := make([]AllowListModel, 0, len(allowList))
	for _, allowListItem := range allowList {
		allowListModels = append(allowListModels, AllowListModel{
			IPAddress: types.StringValue(allowListItem.IPAddress),
			Comment:   types.StringValue(allowListItem.Comment),
		})
	}

	list, diags := types.ListValueFrom(ctx, allowListElementType, allowListModels)
	if diags.HasError() {
		return types.ListNull(allowListElementType), diags
	}
	return list, diags
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state *ServiceResourceModel

	// Read Terraform prior state into the model
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
	var plan *ServiceResourceModel
	// Read Terraform state into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	r.updateAllowedAccountsState(plan, state)
	r.updateAllowListState(plan, state)
	// Save updated state into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
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
	data.FQDN = types.StringValue(service.FQDN)
	data.Name = types.StringValue(service.Name)
	data.SSLEnabled = types.BoolValue(service.SSLEnabled)
	data.ServiceType = types.StringValue(service.ServiceType)
	data.Provider = types.StringValue(service.Provider)
	data.Region = types.StringValue(service.Region)
	data.Version = types.StringValue(service.Version)
	data.Nodes = types.Int64Value(int64(service.Nodes))
	data.Architecture = types.StringValue(service.Architecture)
	data.Size = types.StringValue(service.Size)
	data.Topology = types.StringValue(service.Topology)
	data.Storage = types.Int64Value(int64(service.StorageVolume.Size))
	data.AvailabilityZone = types.StringValue(service.AvailabilityZone)
	if !data.VolumeIOPS.IsNull() && service.StorageVolume.IOPS > 0 {
		data.VolumeIOPS = types.Int64Value(int64(service.StorageVolume.IOPS))
	} else {
		data.VolumeIOPS = types.Int64Null()
	}
	if !data.VolumeThroughput.IsNull() && service.StorageVolume.Throughput > 0 {
		data.VolumeThroughput = types.Int64Value(int64(service.StorageVolume.Throughput))
	} else {
		data.VolumeThroughput = types.Int64Null()
	}
	data.VolumeType = types.StringValue(service.StorageVolume.VolumeType)
	if !data.ReplicationEnabled.IsNull() {
		data.ReplicationEnabled = types.BoolValue(service.ReplicationEnabled)
	}
	if !data.PrimaryHost.IsNull() {
		data.PrimaryHost = types.StringValue(service.PrimaryHost)
	}
	data.IsActive = types.BoolValue(service.IsActive)
	data.SSLEnabled = types.BoolValue(service.SSLEnabled)
	if len(service.Endpoints) > 0 {
		data.Mechanism = types.StringValue(service.Endpoints[0].Mechanism)
		r.setAllowAccounts(ctx, data, service.Endpoints[0].AllowedAccounts)
		data.AllowList, _ = r.allowListToListType(ctx, service.Endpoints[0].AllowList)
		data.EndpointService = types.StringValue(service.Endpoints[0].EndpointService)
	}
	if !(data.MaxscaleSize.IsUnknown() || data.MaxscaleSize.IsNull()) && service.MaxscaleSize != nil {
		data.MaxscaleSize = types.StringValue(*service.MaxscaleSize)
	} else {
		data.MaxscaleSize = types.StringNull()
	}
	if !(data.MaxscaleNodes.IsUnknown() || data.MaxscaleSize.IsNull()) {
		data.MaxscaleNodes = types.Int64Value(int64(service.MaxscaleNodes))
	}
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

	state.WaitForUpdate = plan.WaitForUpdate
	state.WaitForCreation = plan.WaitForCreation
	state.WaitForDeletion = plan.WaitForDeletion
	state.Timeouts = plan.Timeouts
	state.DeletionProtection = plan.DeletionProtection
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateServicePowerState(ctx, plan, state, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateServiceEndpoints(ctx, plan, state, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateServiceSize(ctx, plan, state, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateNumberOfNodeForService(ctx, plan, state, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateServiceStorage(ctx, plan, state, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateAllowList(ctx, plan, state, resp)
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

	r.updateAllowedAccountsState(plan, state)
	r.updateAllowListState(plan, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ServiceResource) updateAllowedAccountsState(plan *ServiceResourceModel, state *ServiceResourceModel) {
	if plan.AllowedAccounts.IsNull() && len(state.AllowedAccounts.Elements()) == 0 {
		state.AllowedAccounts = plan.AllowedAccounts
	}
	if len(plan.AllowedAccounts.Elements()) == 0 && state.AllowedAccounts.IsNull() {
		state.AllowedAccounts = plan.AllowedAccounts
	}
}

func (r *ServiceResource) updateAllowListState(plan *ServiceResourceModel, state *ServiceResourceModel) {
	if plan.AllowList.IsNull() && len(state.AllowList.Elements()) == 0 {
		state.AllowList = plan.AllowList
	}
	if !plan.AllowList.IsUnknown() && len(plan.AllowList.Elements()) == 0 && state.AllowList.IsNull() {
		state.AllowList = plan.AllowList
	}
}

func (r *ServiceResource) updateServiceStorage(ctx context.Context, plan *ServiceResourceModel, state *ServiceResourceModel, resp *resource.UpdateResponse) {
	if plan.Storage.ValueInt64() != state.Storage.ValueInt64() || plan.VolumeIOPS.ValueInt64() != state.VolumeIOPS.ValueInt64() || plan.VolumeThroughput.ValueInt64() != state.VolumeThroughput.ValueInt64() {
		tflog.Info(ctx, "Updating storage size for the service", map[string]interface{}{
			"id":              state.ID.ValueString(),
			"from":            state.Storage.ValueInt64(),
			"to":              plan.Storage.ValueInt64(),
			"iops_from":       state.VolumeIOPS.ValueInt64(),
			"iops_to":         plan.VolumeIOPS.ValueInt64(),
			"throughput_from": state.VolumeThroughput.ValueInt64(),
			"throughput_to":   plan.VolumeThroughput.ValueInt64(),
		})

		err := r.client.ModifyServiceStorage(ctx, state.ID.ValueString(), plan.Storage.ValueInt64(), plan.VolumeIOPS.ValueInt64(), plan.VolumeThroughput.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Error updating a storage for the service",
				fmt.Sprintf("Unable to update a storage size for the service, got error: %s", err))
			return
		}

		state.Storage = plan.Storage
		state.VolumeIOPS = plan.VolumeIOPS
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		r.waitForUpdate(ctx, state, resp)
	}
}

func (r *ServiceResource) updateNumberOfNodeForService(ctx context.Context, plan *ServiceResourceModel, state *ServiceResourceModel, resp *resource.UpdateResponse) {
	if plan.Nodes.ValueInt64() != state.Nodes.ValueInt64() {
		tflog.Info(ctx, "Updating number of nodes for the service", map[string]interface{}{
			"id":   state.ID.ValueString(),
			"from": state.Nodes.ValueInt64(),
			"to":   plan.Nodes.ValueInt64(),
		})

		err := r.client.ModifyServiceNodeNumber(ctx, state.ID.ValueString(), plan.Nodes.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Error updating a number of nodes for the service", fmt.Sprintf("Unable to update a nodes number for the service, got error: %s", err))
			return
		}

		state.Nodes = plan.Nodes
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		r.waitForUpdate(ctx, state, resp)
	}
}

func (r *ServiceResource) updateServiceSize(ctx context.Context, plan *ServiceResourceModel, state *ServiceResourceModel, resp *resource.UpdateResponse) {
	if plan.Size.ValueString() != state.Size.ValueString() {
		tflog.Info(ctx, "Updating service size", map[string]interface{}{
			"id":   state.ID.ValueString(),
			"from": state.Size.ValueString(),
			"to":   plan.Size.ValueString(),
		})

		err := r.client.ModifyServiceSize(ctx, state.ID.ValueString(), plan.Size.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error updating service size", fmt.Sprintf("Unable to update service size, got error: %s", err))
			return
		}

		state.Size = plan.Size
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		r.waitForUpdate(ctx, state, resp)
	}
}

func (r *ServiceResource) updateServiceEndpoints(ctx context.Context, plan *ServiceResourceModel, state *ServiceResourceModel, resp *resource.UpdateResponse) {
	var planAllowedAccounts []string
	d := plan.AllowedAccounts.ElementsAs(ctx, &planAllowedAccounts, false)
	if d.HasError() {
		return
	}

	var stateAllowedAccounts []string
	d = state.AllowedAccounts.ElementsAs(ctx, &stateAllowedAccounts, false)
	if d.HasError() {
		return
	}

	isMechanismChanged := plan.Mechanism.ValueString() != state.Mechanism.ValueString()

	isAllowedAccountsChanged := !reflect.DeepEqual(planAllowedAccounts, stateAllowedAccounts)

	if isMechanismChanged || isAllowedAccountsChanged {
		tflog.Info(ctx, "Updating service allowed accounts", map[string]interface{}{
			"id": state.ID.ValueString(),
		})

		visibility := visibilityPublic
		if Contains[string](privateConnectMechanisms, plan.Mechanism.ValueString()) {
			visibility = visibilityPrivate
		} else {
			planAllowedAccounts = []string{}
		}

		if planAllowedAccounts == nil {
			planAllowedAccounts = []string{}
		}

		endpoint, err := r.client.ModifyServiceEndpoints(ctx,
			state.ID.ValueString(),
			plan.Mechanism.ValueString(),
			planAllowedAccounts,
			visibility)
		if err != nil {
			resp.Diagnostics.AddError("Can not update service", err.Error())
			return
		}

		state.Mechanism = types.StringValue(endpoint.Mechanism)
		r.setAllowAccounts(ctx, state, endpoint.AllowedAccounts)
		state.EndpointService = types.StringValue(endpoint.EndpointService)

		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		r.waitForUpdate(ctx, state, resp)
		return
	}
}

func (r *ServiceResource) updateAllowList(ctx context.Context, plan *ServiceResourceModel, state *ServiceResourceModel, resp *resource.UpdateResponse) {
	if !plan.AllowList.IsUnknown() {
		var planAllowList []AllowListModel
		diags := plan.AllowList.ElementsAs(ctx, &planAllowList, false)
		if diags.HasError() {
			return
		}

		var stateAllowList []AllowListModel
		diags = state.AllowList.ElementsAs(ctx, &stateAllowList, false)
		if diags.HasError() {
			return
		}

		if !reflect.DeepEqual(planAllowList, stateAllowList) {
			tflog.Info(ctx, "Updating service allow list", map[string]interface{}{
				"id": state.ID.ValueString(),
			})

			allowListUpdateRequest := make([]provisioning.AllowListItem, 0)
			for i := range planAllowList {
				allowListUpdateRequest = append(allowListUpdateRequest, provisioning.AllowListItem{
					IPAddress: planAllowList[i].IPAddress.ValueString(),
					Comment:   planAllowList[i].Comment.ValueString(),
				})
			}

			allowListResp, err := r.client.UpdateServiceAllowListByID(ctx, plan.ID.ValueString(), allowListUpdateRequest)
			if err != nil {
				if errors.Is(err, skysql.ErrorServiceNotFound) {
					tflog.Warn(ctx, "SkySQL service not found, removing from state", map[string]interface{}{
						"id": state.ID.ValueString(),
					})
					resp.State.RemoveResource(ctx)

					return
				}
				resp.Diagnostics.AddError("Error updating service allow list", err.Error())
				return
			}

			var cdiags diag.Diagnostics
			state.AllowList, cdiags = r.allowListToListType(ctx, allowListResp)
			if cdiags.HasError() {
				resp.Diagnostics.Append(cdiags...)
				return
			}
			r.updateAllowListState(plan, state)
			// Save updated data into Terraform state
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			if resp.Diagnostics.HasError() {
				return
			}
			r.waitForUpdate(ctx, state, resp)
			return
		}
	}
}

func (r *ServiceResource) updateServicePowerState(ctx context.Context, plan *ServiceResourceModel, state *ServiceResourceModel, resp *resource.UpdateResponse) {
	if !plan.IsActive.IsUnknown() && plan.IsActive.ValueBool() != state.IsActive.ValueBool() {
		tflog.Info(ctx, "Updating service active state", map[string]interface{}{
			"id":        state.ID.ValueString(),
			"is_active": plan.IsActive.ValueBool(),
		})
		err := r.client.SetServicePowerState(ctx, state.ID.ValueString(), plan.IsActive.ValueBool())
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
		r.waitForUpdate(ctx, state, resp)
	}
}

var serviceUpdateWaitStates = []string{"ready", "failed", "stopped"}

func (r *ServiceResource) waitForUpdate(ctx context.Context, state *ServiceResourceModel, resp *resource.UpdateResponse) {
	if state.WaitForUpdate.ValueBool() {
		err := sdkresource.RetryContext(ctx, defaultUpdateTimeout, func() *sdkresource.RetryError {
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

	if state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError("Can not delete service", "Deletion protection is enabled")
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

	// Reminder!!!
	// The plan new state is represented by
	// ModifyPlanResponse.Plan. It must meet the following
	// constraints:
	// 1. Any non-Computed attribute set in plan must preserve the exact
	// plan value or return the corresponding attribute value from the
	// prior state (ModifyPlanRequest.State).
	// 2. Any attribute with a known value must not have its value changed
	// in subsequent calls to ModifyPlan or Create/Read/Update.
	// 3. Any attribute with an unknown value may either remain unknown
	// or take on any value of the expected type.
	//
	// Any errors will prevent further resource-level plan modifications.

	var plan *ServiceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var state *ServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var config *ServiceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !Contains[string]([]string{"gcp", "aws", "azure"}, plan.Provider.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("provider"),
			"Invalid provider value",
			fmt.Sprintf("The %q is an invalid value. Allowed values: aws, gcp, or azure", plan.Provider.ValueString()))
	}

	if plan.Provider.ValueString() == "aws" {
		if !plan.VolumeIOPS.IsNull() && plan.VolumeType.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("volume_type"),
				"volume_type is required",
				"volume_typed is required when volume_iops is set. "+
					"Use: io1|gp2|gp3 for volume_type if volume_iops is set")
			return
		}
		if !plan.VolumeThroughput.IsNull() && plan.VolumeType.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("volume_type"),
				"volume_type is required",
				"volume_typed is required when volume_throughput is set. "+
					"Use: io1|gp2|gp3 for volume_type if volume_throughput is set")
			return
		}
		if !plan.VolumeIOPS.IsNull() && (plan.VolumeType.ValueString() != "io1" && plan.VolumeType.ValueString() != "gp2" && plan.VolumeType.ValueString() != "gp3") {
			resp.Diagnostics.AddAttributeError(path.Root("volume_type"),
				"volume_type must be io1|gp2|gp3 when you want to set IOPS",
				"Use: io1|gp2|gp3 for volume_type if volume_iops is set")
			return
		}
	} else if plan.Provider.ValueString() == "gcp" {
		if !(plan.VolumeType.ValueString() == "" || plan.VolumeType.IsNull() || plan.VolumeType.ValueString() == "pd-ssd") {
			resp.Diagnostics.AddAttributeError(
				path.Root("volume_type"),
				fmt.Sprintf("volume_type = %q is not supported for %q provider",
					plan.VolumeType.ValueString(),
					plan.Provider.ValueString(),
				),
				fmt.Sprintf("Volume type is not supported for %q provider", plan.Provider.ValueString()))
			return
		}

		if plan.VolumeType.ValueString() == "" || plan.VolumeType.IsNull() {
			resp.Plan.SetAttribute(ctx, path.Root("volume_type"), types.StringValue("pd-ssd"))
		}

		if !plan.VolumeIOPS.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("volume_iops"),
				fmt.Sprintf("volume_iops is not supported for %q provider", plan.Provider.ValueString()),
				fmt.Sprintf("Volume IOPS are not supported for %q provider", plan.Provider.ValueString()))
			return
		}
	} else if plan.Provider.ValueString() == "azure" {
		if plan.VolumeType.ValueString() != "" && plan.VolumeType.ValueString() != "StandardSSD_LRS" {
			resp.Diagnostics.AddAttributeError(path.Root("volume_type"),
				"volume_type is not supported for azure provider",
				"Volume type is not supported for azure provider")
			return
		}
		if !plan.VolumeIOPS.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("volume_iops"),
				"volume_iops is not supported for azure provider",
				"Volume IOPS are not supported for azure provider")
			return
		}
	}

	if !Contains[string]([]string{"lakehouse", "sa"}, plan.Topology.ValueString()) {
		if plan.SSLEnabled.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("ssl_enabled"),
				"Missing required argument",
				fmt.Sprintf("The argument %q is required, but no definition was found.", "ssl_enabled"))
			return
		}
		if plan.Storage.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("storage"),
				"Missing required argument",
				fmt.Sprintf("The argument %q is required, but no definition was found.", "storage"))
			return
		}
		if plan.Size.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("size"),
				"Missing required argument",
				fmt.Sprintf("The argument %q is required, but no definition was found.", "size"))
			return
		}
		if plan.Nodes.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("size"),
				"Missing required argument",
				fmt.Sprintf("The argument %q is required, but no definition was found.", "size"))
			return
		}
	} else {
		if state == nil && !plan.Architecture.IsUnknown() {
			// We apply validation only if the resource is not created yet
			resp.Diagnostics.AddAttributeError(path.Root("architecture"),
				"Attempt to set read-only attribute",
				fmt.Sprintf("The argument %q is read only for the %q topology", "architecture", plan.Topology.ValueString()))
		}

		if state == nil && !plan.Nodes.IsUnknown() {
			resp.Diagnostics.AddAttributeError(path.Root("nodes"),
				"Attempt to set read-only attribute",
				fmt.Sprintf("The argument %q is read only for the %q topology", "nodes", plan.Topology.ValueString()))
		}

		if state == nil && !plan.Size.IsUnknown() {
			resp.Diagnostics.AddAttributeError(path.Root("size"),
				"Attempt to set read-only attribute",
				fmt.Sprintf("The argument %q is read only for the %q topology", "size", plan.Topology.ValueString()))
		}

		if state == nil && !plan.SSLEnabled.IsUnknown() {
			resp.Diagnostics.AddAttributeError(path.Root("ssl_enabled"),
				"Attempt to set read-only attribute",
				fmt.Sprintf("The argument %q is read only for the %q topology", "ssl_enabled", plan.Topology.ValueString()))
		}

		if state == nil && !plan.Version.IsUnknown() {
			resp.Diagnostics.AddAttributeError(path.Root("version"),
				"Attempt to set read-only attribute",
				fmt.Sprintf("The argument %q is read only for the %q topology", "version", plan.Topology.ValueString()))
		}

		if state != nil && plan.Nodes.ValueInt64() != state.Nodes.ValueInt64() {
			resp.Diagnostics.AddAttributeError(path.Root("nodes"),
				"Attempt to modify read-only attribute",
				fmt.Sprintf("The argument %q is read only for the %q topology", "version", plan.Topology.ValueString()))
		}

		if state != nil && plan.Size.ValueString() != state.Size.ValueString() {
			resp.Diagnostics.AddAttributeError(path.Root("size"),
				"Attempt to modify read-only attribute",
				fmt.Sprintf("The argument %q is read only for the %q topology", "version", plan.Topology.ValueString()))
		}
	}

	if state != nil && plan.Architecture.ValueString() != state.Architecture.ValueString() {
		resp.Diagnostics.AddError("Cannot change service architecture",
			"To prevent accidental deletion of data, changing architecture isn't allowed. "+
				"Please explicitly destroy this service before changing its architecture.")
	}

	if state != nil && plan.SSLEnabled.ValueBool() != state.SSLEnabled.ValueBool() {
		resp.Diagnostics.AddError("Cannot change service ssl_enabled",
			"To prevent accidental deletion of data, changing ssl_enabled isn't allowed. "+
				"Please explicitly destroy this service before changing its ssl_enabled.")
	}

	if state != nil && plan.Version.ValueString() != state.Version.ValueString() {
		resp.Diagnostics.AddError("Cannot change service version",
			"To prevent accidental deletion of data, changing version isn't allowed. "+
				"Please explicitly destroy this service before changing its version.")
	}

	if state == nil &&
		Contains[string](privateConnectMechanisms, plan.Mechanism.ValueString()) &&
		!plan.AllowList.IsUnknown() &&
		!plan.AllowList.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("allow_list"),
			fmt.Sprintf("You can not set allow_list when mechanism has %q value", plan.Mechanism.ValueString()),
			fmt.Sprintf("When you set mechanism=%q, don't use allow_list, use endpoint_allowed_accounts instead", plan.Mechanism.ValueString()))
	}

	if plan.Mechanism.ValueString() == "nlb" {
		// Force mechanism update
		resp.Plan.SetAttribute(ctx, path.Root("endpoint_allowed_accounts"), types.ListNull(types.StringType))
	}

	if state != nil && !state.AllowList.IsUnknown() && plan.AllowList.IsNull() {
		resp.Plan.SetAttribute(ctx, path.Root("allow_list"), state.AllowList)
	}
}

func (r *ServiceResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &serviceResourceSchemaV0,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var state ServiceResourceModel
				diags := req.State.Get(ctx, &state)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				if state.Provider.ValueString() == "gcp" {
					state.VolumeType = types.StringValue("pd-ssd")
					diags = resp.State.Set(ctx, state)
					resp.Diagnostics.Append(diags...)
				}
			},
		},
	}
}
