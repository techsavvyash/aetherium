package firecracker

// Firecracker API request/response types
// Reference: https://github.com/firecracker-microvm/firecracker/blob/main/src/api_server/swagger/firecracker.yaml

// BootSource configures the kernel and boot arguments
type BootSource struct {
	KernelImagePath string  `json:"kernel_image_path"`
	BootArgs        *string `json:"boot_args,omitempty"`
}

// Drive configures a block device
type Drive struct {
	DriveID      string `json:"drive_id"`
	PathOnHost   string `json:"path_on_host"`
	IsRootDevice bool   `json:"is_root_device"`
	IsReadOnly   bool   `json:"is_read_only"`
}

// MachineConfiguration defines VM resources
type MachineConfiguration struct {
	VcpuCount  int `json:"vcpu_count"`
	MemSizeMib int `json:"mem_size_mib"`
}

// InstanceActionInfo triggers VM actions
type InstanceActionInfo struct {
	ActionType string `json:"action_type"`
}

// Action types
const (
	ActionInstanceStart  = "InstanceStart"
	ActionSendCtrlAltDel = "SendCtrlAltDel"
)

// InstanceInfo contains VM state information
type InstanceInfo struct {
	State        string `json:"state"`
	ID           string `json:"id"`
	VMMVersion   string `json:"vmm_version"`
	AppName      string `json:"app_name"`
	StartTimeUS  int64  `json:"start_time_us,omitempty"`
	StartTimeCPU int64  `json:"start_time_cpu_us,omitempty"`
}
