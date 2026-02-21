package provisioning

// Config represents a DPS configuration object.
type Config struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	CreatedDate int64    `json:"created_date"`
	UpdatedDate int64    `json:"updated_date"`
	CreatedBy   string   `json:"created_by,omitempty"`
	UpdatedBy   string   `json:"updated_by,omitempty"`
	Public      bool     `json:"public"`
	TopologyID  string   `json:"topology_id"`
	VersionID   string   `json:"version_id"`
	Services    []string `json:"services,omitempty"`
}

// CreateConfigRequest is the request body for POST /configs.
// Uses name-based topology and version resolution (Phase 1 DPS enhancement).
type CreateConfigRequest struct {
	Name     string `json:"name"`
	Topology string `json:"topology"`
	Version  string `json:"version"`
}

// UpdateConfigRequest is the request body for PATCH /configs/{id}.
type UpdateConfigRequest struct {
	Name string `json:"name"`
}

// ConfigValueRequest is the request body for POST /configs/{id}/values/{variable_name}.
type ConfigValueRequest struct {
	Value string `json:"value"`
}
