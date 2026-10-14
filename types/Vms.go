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
	Name          string                 `json:"name"`
	Status        string                 `json:"status"`
	RAM           int                    `json:"ram_mb"`
	Networks      []string               `json:"networks"`
	AvailableNICs []string               `json:"available_nics"`
	Connections   []VMConnection         `json:"connections"`
}
