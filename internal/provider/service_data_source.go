package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mariadb-corporation/terraform-provider-skysql-v2/internal/skysql"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &ServiceDataSource{}

func NewServiceDataSource() datasource.DataSource {
	return &ServiceDataSource{}
}

// ServiceDataSource defines the data source implementation.
type ServiceDataSource struct {
	client *skysql.Client
}

type ServiceDataSourceModel struct {
	ID                 types.String                     `tfsdk:"service_id"`
	Name               types.String                     `tfsdk:"name"`
	Region             types.String                     `tfsdk:"region"`
	Provider           types.String                     `tfsdk:"cloud_provider"`
	Tier               types.String                     `tfsdk:"tier"`
	Topology           types.String                     `tfsdk:"topology"`
	Version            types.String                     `tfsdk:"version"`
	Architecture       types.String                     `tfsdk:"architecture"`
	Size               types.String                     `tfsdk:"size"`
	Nodes              types.Int64                      `tfsdk:"nodes"`
	SslEnabled         types.Bool                       `tfsdk:"ssl_enabled"`
	NosqlEnabled       types.Bool                       `tfsdk:"nosql_enabled"`
	FQDN               types.String                     `tfsdk:"fqdn"`
	Status             types.String                     `tfsdk:"status"`
	CreatedOn          types.Int64                      `tfsdk:"created_on"`
	UpdatedOn          types.Int64                      `tfsdk:"updated_on"`
	CreatedBy          types.String                     `tfsdk:"created_by"`
	UpdatedBy          types.String                     `tfsdk:"updated_by"`
	Endpoints          []ServiceEndpointDataSourceModel `tfsdk:"endpoints"`
	StorageVolume      *StorageVolumeDataSourceModel    `tfsdk:"storage_volume"`
	OutboundIps        []types.String                   `tfsdk:"outbound_ips"`
	IsActive           types.Bool                       `tfsdk:"is_active"`
	ServiceType        types.String                     `tfsdk:"service_type"`
	ReplicationEnabled types.Bool                       `tfsdk:"replication_enabled"`
	PrimaryHost        types.String                     `tfsdk:"primary_host"`
}

type ServiceEndpointDataSourceModel struct {
	Name  types.String                  `tfsdk:"name"`
	Ports []EndpointPortDataSourceModel `tfsdk:"ports"`
}

type EndpointPortDataSourceModel struct {
	Name    types.String `tfsdk:"name"`
	Port    types.Int64  `tfsdk:"port"`
	Purpose types.String `tfsdk:"purpose"`
}

type StorageVolumeDataSourceModel struct {
	Size       types.Int64  `tfsdk:"size"`
	VolumeType types.String `tfsdk:"volume_type"`
	IOPS       types.Int64  `tfsdk:"iops"`
}

func (d *ServiceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *ServiceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"region": schema.StringAttribute{
				Computed: true,
			},
			"cloud_provider": schema.StringAttribute{
				Computed: true,
			},
			"tier": schema.StringAttribute{
				Computed: true,
			},
			"topology": schema.StringAttribute{
				Computed: true,
			},
			"version": schema.StringAttribute{
				Computed: true,
			},
			"architecture": schema.StringAttribute{
				Computed: true,
			},
			"size": schema.StringAttribute{
				Computed: true,
			},
			"nodes": schema.Int64Attribute{
				Computed: true,
			},
			"ssl_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"nosql_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"fqdn": schema.StringAttribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"created_on": schema.Int64Attribute{
				Computed: true,
			},
			"updated_on": schema.Int64Attribute{
				Computed: true,
			},
			"created_by": schema.StringAttribute{
				Computed: true,
			},
			"updated_by": schema.StringAttribute{
				Computed: true,
			},
			"endpoints": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed: true,
						},
						"ports": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Computed: true,
									},
									"port": schema.Int64Attribute{
										Computed: true,
									},
									"purpose": schema.StringAttribute{
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
			"storage_volume": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"size": schema.Int64Attribute{
						Computed: true,
					},
					"volume_type": schema.StringAttribute{
						Computed: true,
					},
					"iops": schema.Int64Attribute{
						Computed: true,
						Optional: true},
				},
			},
			"outbound_ips": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"is_active": schema.BoolAttribute{
				Computed: true,
			},
			"service_type": schema.StringAttribute{
				Computed: true,
			},
			"replication_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"primary_host": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *ServiceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}

func (d *ServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServiceDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing Service ID",
			"service_id is required",
		)
		return
	}

	service, err := d.client.GetServiceByID(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read SkySQL service", err.Error())
		return
	}

	data.ID = types.StringValue(service.ID)
	data.Name = types.StringValue(service.Name)
	data.Region = types.StringValue(service.Region)
	data.Provider = types.StringValue(service.Provider)
	data.Tier = types.StringValue(service.Tier)
	data.Topology = types.StringValue(service.Topology)
	data.Version = types.StringValue(service.Version)
	data.Architecture = types.StringValue(service.Architecture)
	data.Size = types.StringValue(service.Size)
	data.Nodes = types.Int64Value(int64(service.Nodes))
	data.SslEnabled = types.BoolValue(service.SslEnabled)
	data.NosqlEnabled = types.BoolValue(service.NosqlEnabled)
	data.FQDN = types.StringValue(service.FQDN)
	data.Status = types.StringValue(service.Status)
	data.CreatedOn = types.Int64Value(int64(service.CreatedOn))
	data.UpdatedOn = types.Int64Value(int64(service.UpdatedOn))
	data.CreatedBy = types.StringValue(service.CreatedBy)
	data.UpdatedBy = types.StringValue(service.UpdatedBy)
	data.Endpoints = make([]ServiceEndpointDataSourceModel, len(service.Endpoints))
	for i := range service.Endpoints {
		data.Endpoints[i].Name = types.StringValue(service.Endpoints[i].Name)
		data.Endpoints[i].Ports = make([]EndpointPortDataSourceModel, len(service.Endpoints[i].Ports))
		for j := range service.Endpoints[i].Ports {
			data.Endpoints[i].Ports[j].Name = types.StringValue(service.Endpoints[i].Ports[j].Name)
			data.Endpoints[i].Ports[j].Port = types.Int64Value(int64(service.Endpoints[i].Ports[j].Port))
			data.Endpoints[i].Ports[j].Purpose = types.StringValue(service.Endpoints[i].Ports[j].Purpose)
		}
	}

	data.StorageVolume = &StorageVolumeDataSourceModel{
		Size:       types.Int64Value(int64(service.StorageVolume.Size)),
		VolumeType: types.StringValue(service.StorageVolume.VolumeType),
		IOPS:       types.Int64Value(int64(service.StorageVolume.IOPS)),
	}

	data.OutboundIps = make([]types.String, len(service.OutboundIps))
	for i := range service.OutboundIps {
		data.OutboundIps[i] = types.StringValue(service.OutboundIps[i])
	}

	data.IsActive = types.BoolValue(service.IsActive)
	data.ServiceType = types.StringValue(service.ServiceType)
	data.ReplicationEnabled = types.BoolValue(service.ReplicationEnabled)
	data.PrimaryHost = types.StringValue(service.PrimaryHost)
	// Set state
	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
