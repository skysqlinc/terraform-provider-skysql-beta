package provisioning

type UpdateStorageIOPSRequest struct {
	// Type of the EBS volume. Can be gp2, gp3, io1 or io2
	VolumeType string `json:"volume_type,omitempty"`
	// Input/output operations per second
	// Minimum / Maximum IOPs depend on the size of the disk and volume type.
	IOPS int64 `json:"iops,omitempty"`
}
