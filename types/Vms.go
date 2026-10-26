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
	Type          string                 `json:"type"`
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
	IPAddress     string                 `json:"ip_address,omitempty"`
	ConfiguredIP  string                 `json:"configured_ip,omitempty"`
	Gateway       string                 `json:"gateway,omitempty"`
	IsVulnerable  bool                   `json:"is_vulnerable,omitempty"`
	Category      string                 `json:"category,omitempty"`
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

type SetIPRequest struct {
	VMID       int    `json:"vmid"`
	IP         string `json:"ip"`
	Gateway    string `json:"gateway,omitempty"`
	IsDHCP     bool   `json:"is_dhcp"`
	TargetNode string `json:"node,omitempty"`
}

type CatalogItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Category     string `json:"category"`
	Description  string `json:"description"`
	TemplateID   int    `json:"template_id,omitempty"`
	OSTemplate   string `json:"ostemplate,omitempty"`
	DefaultRAM   int    `json:"default_ram"`
	DefaultDisk  int    `json:"default_disk_gb"`
	DefaultCPUs  int    `json:"default_cpus"`
	IsVulnerable bool   `json:"is_vulnerable"`
	Ready        bool   `json:"ready"`
}

type DeployCatalogRequest struct {
	CatalogID  string `json:"catalog_id"`
	Name       string `json:"name"`
	RAM        int    `json:"ram,omitempty"`
	Cores      int    `json:"cores,omitempty"`
	Disk       int    `json:"disk,omitempty"`
	ProjectID  string `json:"project_id,omitempty"`
	IP         string `json:"ip,omitempty"`
	Gateway    string `json:"gateway,omitempty"`
	IsDHCP     bool   `json:"is_dhcp"`
	TargetNode string `json:"node,omitempty"`
	Password   string `json:"password,omitempty"`
}
