package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql/provisioning"
	"github.com/stretchr/testify/require"
)

const (
	testConfigID   = "cfg-abc-123"
	testTopologyID = "topo-uuid-001"
	testVersionID  = "ver-uuid-002"
	testConfigName = "my-test-config"
	testTopology   = "es-single"
	testVersion    = "10.6.7-3-1"
)

func versionsResponse(t *testing.T) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/versions", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]provisioning.Version{})
	}
}

func createConfigResponse(t *testing.T) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodPost, req.Method)
		r.Equal("/provisioning/v1/configs", req.URL.Path)

		var payload provisioning.CreateConfigRequest
		err := json.NewDecoder(req.Body).Decode(&payload)
		r.NoError(err)
		r.Equal(testConfigName, payload.Name)
		r.Equal(testTopology, payload.Topology)
		r.Equal(testVersion, payload.Version)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(provisioning.Config{
			ID:         testConfigID,
			Name:       testConfigName,
			TopologyID: testTopologyID,
			VersionID:  testVersionID,
		})
	}
}

func getConfigResponse(t *testing.T) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/configs/"+testConfigID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(provisioning.Config{
			ID:         testConfigID,
			Name:       testConfigName,
			TopologyID: testTopologyID,
			VersionID:  testVersionID,
		})
	}
}

func deleteConfigResponse(t *testing.T) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodDelete, req.Method)
		r.Equal("/provisioning/v1/configs/"+testConfigID, req.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}
}

func setConfigValueResponse(t *testing.T, variableName string, expectedValue string) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodPost, req.Method)
		r.Equal("/provisioning/v1/configs/"+testConfigID+"/values/"+variableName, req.URL.Path)

		var payload provisioning.ConfigValueRequest
		err := json.NewDecoder(req.Body).Decode(&payload)
		r.NoError(err)
		r.Equal(expectedValue, payload.Value)

		w.WriteHeader(http.StatusNoContent)
	}
}

func unsetConfigValueResponse(t *testing.T, variableName string) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodDelete, req.Method)
		r.Equal("/provisioning/v1/configs/"+testConfigID+"/values/"+variableName, req.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}
}

func TestConfigResource_CreateWithValues(t *testing.T) {
	configureOnce.Reset()

	testUrl, expectRequest, close := mockSkySQLAPI(t)
	defer close()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testUrl)

	// Provider configure
	expectRequest(versionsResponse(t))
	// Create: POST /configs
	expectRequest(createConfigResponse(t))
	// Create: set values (alphabetical order)
	expectRequest(setConfigValueResponse(t, "innodb_buffer_pool_size", "2G"))
	expectRequest(setConfigValueResponse(t, "max_connections", "500"))
	// Read after create
	expectRequest(getConfigResponse(t))
	// Destroy: delete
	expectRequest(deleteConfigResponse(t))

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"skysql": providerserver.NewProtocol6WithError(New("")()),
		},
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "skysql_config" "test" {
					name     = "%s"
					topology = "%s"
					version  = "%s"
					values = {
						"max_connections"         = "500"
						"innodb_buffer_pool_size" = "2G"
					}
				}`, testConfigName, testTopology, testVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("skysql_config.test", "id", testConfigID),
					resource.TestCheckResourceAttr("skysql_config.test", "name", testConfigName),
					resource.TestCheckResourceAttr("skysql_config.test", "topology", testTopology),
					resource.TestCheckResourceAttr("skysql_config.test", "version", testVersion),
					resource.TestCheckResourceAttr("skysql_config.test", "topology_id", testTopologyID),
					resource.TestCheckResourceAttr("skysql_config.test", "version_id", testVersionID),
					resource.TestCheckResourceAttr("skysql_config.test", "values.max_connections", "500"),
					resource.TestCheckResourceAttr("skysql_config.test", "values.innodb_buffer_pool_size", "2G"),
				),
			},
		},
	})
}

func TestConfigResource_CreateWithoutValues(t *testing.T) {
	configureOnce.Reset()

	testUrl, expectRequest, close := mockSkySQLAPI(t)
	defer close()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testUrl)

	// Provider configure
	expectRequest(versionsResponse(t))
	// Create: POST /configs (no values to set)
	expectRequest(createConfigResponse(t))
	// Read after create
	expectRequest(getConfigResponse(t))
	// Destroy: delete
	expectRequest(deleteConfigResponse(t))

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"skysql": providerserver.NewProtocol6WithError(New("")()),
		},
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "skysql_config" "test" {
					name     = "%s"
					topology = "%s"
					version  = "%s"
				}`, testConfigName, testTopology, testVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("skysql_config.test", "id", testConfigID),
					resource.TestCheckResourceAttr("skysql_config.test", "name", testConfigName),
					resource.TestCheckResourceAttr("skysql_config.test", "topology_id", testTopologyID),
					resource.TestCheckResourceAttr("skysql_config.test", "version_id", testVersionID),
				),
			},
		},
	})
}

func TestConfigResource_UpdateNameAndValues(t *testing.T) {
	configureOnce.Reset()

	const updatedName = "renamed-config"

	testUrl, expectRequest, close := mockSkySQLAPI(t)
	defer close()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testUrl)

	// Provider configure
	expectRequest(versionsResponse(t))

	// --- Step 1: Create with one value ---
	// Create: POST /configs
	expectRequest(createConfigResponse(t))
	// Create: set value
	expectRequest(setConfigValueResponse(t, "max_connections", "500"))
	// Read after create
	expectRequest(getConfigResponse(t))

	// --- Step 2: Update name + change value + add value ---
	// Read before update (plan)
	expectRequest(getConfigResponse(t))
	// Update: PATCH /configs/{id} (name changed)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodPatch, req.Method)
		r.Equal("/provisioning/v1/configs/"+testConfigID, req.URL.Path)

		var payload provisioning.UpdateConfigRequest
		err := json.NewDecoder(req.Body).Decode(&payload)
		r.NoError(err)
		r.Equal(updatedName, payload.Name)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(provisioning.Config{
			ID:         testConfigID,
			Name:       updatedName,
			TopologyID: testTopologyID,
			VersionID:  testVersionID,
		})
	})
	// Update: set changed value (alphabetical: innodb_buffer_pool_size before max_connections)
	expectRequest(setConfigValueResponse(t, "innodb_buffer_pool_size", "4G"))
	expectRequest(setConfigValueResponse(t, "max_connections", "1000"))
	// Read after update
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/configs/"+testConfigID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(provisioning.Config{
			ID:         testConfigID,
			Name:       updatedName,
			TopologyID: testTopologyID,
			VersionID:  testVersionID,
		})
	})

	// --- Destroy ---
	expectRequest(deleteConfigResponse(t))

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"skysql": providerserver.NewProtocol6WithError(New("")()),
		},
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "skysql_config" "test" {
					name     = "%s"
					topology = "%s"
					version  = "%s"
					values = {
						"max_connections" = "500"
					}
				}`, testConfigName, testTopology, testVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("skysql_config.test", "name", testConfigName),
					resource.TestCheckResourceAttr("skysql_config.test", "values.max_connections", "500"),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "skysql_config" "test" {
					name     = "%s"
					topology = "%s"
					version  = "%s"
					values = {
						"max_connections"         = "1000"
						"innodb_buffer_pool_size" = "4G"
					}
				}`, updatedName, testTopology, testVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("skysql_config.test", "name", updatedName),
					resource.TestCheckResourceAttr("skysql_config.test", "values.max_connections", "1000"),
					resource.TestCheckResourceAttr("skysql_config.test", "values.innodb_buffer_pool_size", "4G"),
				),
			},
		},
	})
}
