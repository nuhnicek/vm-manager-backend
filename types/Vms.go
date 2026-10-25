package types

import "github.com/luthermonson/go-proxmox"

type VMConnection struct {
	TargetVMID string `json:"target_vm_id"`
	LocalNIC   string `json:"local_nic"`
	TargetNIC  string `json:"target_nic"`
}

type Vm struct {
	Id            proxmox.StringOrUint64 `json:"id"`
	UUID          string                 `json:"uuid"`
	OwnerUUID     string                 `json:"owner_uuid,omitempty"`
	OwnerEmail    string                 `json:"owner_email,omitempty"`
	LabID         string                 `json:"lab_id,omitempty"`
	LabName       string                 `json:"lab_name,omitempty"`
	ProjectID     string                 `json:"project_id,omitempty"`
	ProjectName   string                 `json:"project_name,omitempty"`
	ProjectColor  string                 `json:"project_color,omitempty"`
	Node          string                 `json:"node,omitempty"`
	Name          string                 `json:"name"`
	Status        string                 `json:"status"`
	RAM           int                    `json:"ram_mb"`
	Owner         string                 `json:"owner,omitempty"`
	Networks      []string               `json:"networks"`
	AvailableNICs []string               `json:"available_nics"`
	Connections   []VMConnection         `json:"connections"`
}

type ProxmoxNodeInfo struct {
	Name       string `json:"name"`
	HostURI    string `json:"host_uri"`
	Status     string `json:"status"`
	Version    string `json:"version,omitempty"`
	TotalVMs   int    `json:"total_vms"`
	RunningVMs int    `json:"running_vms"`
	Templates  int    `json:"templates"`
}
