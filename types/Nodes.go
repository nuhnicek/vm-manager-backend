package types

import (
	"context"
	"time"

	"github.com/luthermonson/go-proxmox"
)

// StandaloneNode represents an independent, non-clustered Proxmox VE server.
type StandaloneNode struct {
	ID        string    `json:"id" bson:"_id"`
	Name      string    `json:"name" bson:"name"`           // Proxmox node name (e.g. "pve1")
	HostURI   string    `json:"host_uri" bson:"host_uri"`   // Full API URI, e.g. "https://10.0.1.11:8006/api2/json"
	Username  string    `json:"username" bson:"username"`   // e.g. "root@pam"
	Password  string    `json:"password" bson:"password"`   // PAM password or API token
	Weight    int       `json:"weight" bson:"weight"`       // Balancing weight (default: 100)
	IsActive  bool      `json:"is_active" bson:"is_active"` // Whether scheduler sends VMs here
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// NodeTelemetry holds real-time health and resource utilization metrics for a node.
type NodeTelemetry struct {
	NodeID        string    `json:"node_id"`
	NodeName      string    `json:"node_name"`
	HostURI       string    `json:"host_uri"`
	IsOnline      bool      `json:"is_online"`
	TotalRAMMB    int       `json:"total_ram_mb"`
	UsedRAMMB     int       `json:"used_ram_mb"`
	FreeRAMMB     int       `json:"free_ram_mb"`
	RAMUsagePct   float64   `json:"ram_usage_pct"`
	CPUUsagePct   float64   `json:"cpu_usage_pct"`
	ActiveVMCount int       `json:"active_vm_count"`
	TotalVMCount  int       `json:"total_vm_count"`
	Templates     []int     `json:"templates"` // VMIDs of templates present on this node
	Weight        int       `json:"weight"`
	IsActive      bool      `json:"is_active"`
	LastUpdated   time.Time `json:"last_updated"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// NodePlacementRequest specifies the resource and template requirements for placing a VM or lab.
type NodePlacementRequest struct {
	RequiredRAMMB      int  `json:"required_ram_mb"`
	RequiredTemplateID int  `json:"required_template_id"`
	PreferNodeID       string `json:"prefer_node_id,omitempty"`
}

// CreateTemplateRequest defines parameters for converting or cloning a VM into a template.
type CreateTemplateRequest struct {
	SourceVMID   int    `json:"source_vmid"`
	TemplateName string `json:"template_name"`
	CloneFirst   bool   `json:"clone_first"` // If true, clones to new VM first instead of converting source in-place
	TargetNodeID string `json:"target_node_id,omitempty"`
	Description  string `json:"description,omitempty"`
	RAMMB        int    `json:"ram_mb,omitempty"`
}

// NodePoolInterface abstracts multi-node discovery, telemetry, and load-balanced placement.
type NodePoolInterface interface {
	GetTelemetry(ctx context.Context) []NodeTelemetry
	SelectNode(ctx context.Context, req NodePlacementRequest) (*proxmox.Client, *proxmox.Node, string, error)
	GetClientForNode(nodeID string) (*proxmox.Client, *proxmox.Node, error)
	FindVM(ctx context.Context, vmid int) (*proxmox.Client, *proxmox.Node, *proxmox.VirtualMachine, error)
	AllocateVMID(ctx context.Context) (int, error)
	ListAllTemplates(ctx context.Context) ([]Vm, error)
	GetNodes(ctx context.Context) []*proxmox.Node
	RegisterNode(ctx context.Context, node StandaloneNode) error
	RemoveNode(ctx context.Context, nodeID string) error
	RefreshTelemetry(ctx context.Context)
}

