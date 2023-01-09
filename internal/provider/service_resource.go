package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdkresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/mariadb-corporation/terraform-provider-skysql-beta/internal/skysql"
	"github.com/mariadb-corporation/terraform-provider-skysql-beta/internal/skysql/provisioning"
	"regexp"
	"time"
)

const defaultCreateTimeout = 60 * time.Minute

var rxServiceName = regexp.MustCompile("(^[a-z][a-z0-9-]+$)")

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
	ID              types.String   `tfsdk:"id"`
	Name            types.String   `tfsdk:"name"`
	ProjectID       types.String   `tfsdk:"project_id"`
	ServiceType     types.String   `tfsdk:"service_type"`
	Provider        types.String   `tfsdk:"cloud_provider"`
	Region          types.String   `tfsdk:"region"`
	Version         types.String   `tfsdk:"version"`
	Nodes           types.Int64    `tfsdk:"nodes"`
	Architecture    types.String   `tfsdk:"architecture"`
	Size            types.String   `tfsdk:"size"`
	Topology        types.String   `tfsdk:"topology"`
	Storage         types.Int64    `tfsdk:"storage"`
	VolumeIOPS      types.Int64    `tfsdk:"volume_iops"`
	SSLEnabled      types.Bool     `tfsdk:"ssl_enabled"`
	NoSQLEnabled    types.Bool     `tfsdk:"nosql_enabled"`
	VolumeType      types.String   `tfsdk:"volume_type"`
	WaitForCreation types.Bool     `tfsdk:"wait_for_creation"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
	Mechanism       types.String   `tfsdk:"endpoint_mechanism"`
	AllowedAccounts types.List     `tfsdk:"endpoint_allowed_accounts"`
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
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The ID of the service",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the service",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 24),
					stringvalidator.RegexMatches(
						rxServiceName,
						"must start from a lowercase letter and contain only lowercase letters, numbers and hyphens",
					),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    true,
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
				Description: "The volume type. Valid values are: gp2,gp3,io1,io2. This is only applicable for AWS",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"endpoint_allowed_accounts": schema.ListAttribute{
				Optional:    true,
				Description: "The list of cloud accounts (aws account ids or gcp projects) that are allowed to access the service",
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
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
		Mechanism:    data.Mechanism.ValueString(),
	}

	for _, element := range data.AllowedAccounts.Elements() {
		createServiceRequest.AllowedAccounts = append(createServiceRequest.AllowedAccounts, element.String())
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
	if resp.Diagnostics.HasError() {
		return
	}
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
