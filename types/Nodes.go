package types

import (
	"context"
	"time"

	"github.com/luthermonson/go-proxmox"
)

type StandaloneNode struct {
	ID        string    `json:"id" bson:"_id"`
	Name      string    `json:"name" bson:"name"`
	HostURI   string    `json:"host_uri" bson:"host_uri"`
	Username  string    `json:"username" bson:"username"`
	Password  string    `json:"password" bson:"password"`
	Weight    int       `json:"weight" bson:"weight"`
	IsActive  bool      `json:"is_active" bson:"is_active"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

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
	Templates     []int     `json:"templates"`
	Weight        int       `json:"weight"`
	IsActive      bool      `json:"is_active"`
	LastUpdated   time.Time `json:"last_updated"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

type NodePlacementRequest struct {
	RequiredRAMMB      int  `json:"required_ram_mb"`
	RequiredTemplateID int  `json:"required_template_id"`
	PreferNodeID       string `json:"prefer_node_id,omitempty"`
}

type CreateTemplateRequest struct {
	SourceVMID   int    `json:"source_vmid"`
	TemplateName string `json:"template_name"`
	CloneFirst   bool   `json:"clone_first"`
	TargetNodeID string `json:"target_node_id,omitempty"`
	Description  string `json:"description,omitempty"`
	RAMMB        int    `json:"ram_mb,omitempty"`
}

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

