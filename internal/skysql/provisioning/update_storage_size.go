package provisioning

type UpdateStorageSizeRequest struct {
	// Size in GBs
	Size int64 `json:"size"`
}
