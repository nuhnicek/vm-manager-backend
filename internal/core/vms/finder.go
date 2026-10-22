package vms

import (
	"context"
	"fmt"
	"strings"
	"time"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
)

// ExtractUUIDFromTags parses the tag string and returns a UUID if present,
// otherwise generates a stable fallback identifier like "tpl-<vmid>".
func ExtractUUIDFromTags(tags string, vmid uint64) string {
	tagList := strings.Split(tags, ";")
	for _, tag := range tagList {
		trimmed := strings.TrimSpace(tag)
		if strings.HasPrefix(trimmed, "vm-") {
			return strings.TrimPrefix(trimmed, "vm-")
		}
	}
	for _, tag := range tagList {
		trimmed := strings.TrimSpace(tag)
		if len(trimmed) == 36 && strings.Count(trimmed, "-") == 4 {
			return trimmed
		}
	}
	return fmt.Sprintf("tpl-%d", vmid)
}

// GetClusterNodes returns all online proxmox nodes in the cluster.
func GetClusterNodes(ctx context.Context, client *proxmox.Client) ([]*proxmox.Node, error) {
	if client == nil {
		return nil, fmt.Errorf("proxmox client is nil")
	}

	nodeStatuses, err := client.Nodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var activeNodes []*proxmox.Node
	for _, ns := range nodeStatuses {
		if strings.EqualFold(ns.Status, "online") || ns.Status == "" {
			node, err := client.Node(ctx, ns.Node)
			if err == nil && node != nil {
				activeNodes = append(activeNodes, node)
			}
		}
	}

	if len(activeNodes) == 0 {
		return nil, fmt.Errorf("no online nodes found in Proxmox cluster")
	}
	return activeNodes, nil
}

// FindVMNode dynamically locates the node and VirtualMachine instance for a given VMID
// across any node in the Proxmox cluster without requiring hardcoded node names.
func FindVMNode(ctx context.Context, client *proxmox.Client, vmid int) (*proxmox.Node, *proxmox.VirtualMachine, error) {
	if client == nil {
		return nil, nil, fmt.Errorf("proxmox client is nil")
	}

	// 1. First, check Proxmox cluster resources (fastest, covers all nodes in one API call)
	cCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	cluster, cErr := client.Cluster(cCtx)
	if cErr == nil && cluster != nil {
		if resources, err := cluster.Resources(cCtx, "vm"); err == nil {
			for _, r := range resources {
				if r.VMID == uint64(vmid) && r.Node != "" {
					cancel()
					node, err := client.Node(ctx, r.Node)
					if err == nil && node != nil {
						vm, err := node.VirtualMachine(ctx, vmid)
						if err == nil && vm != nil {
							return node, vm, nil
						}
					}
				}
			}
		}
	}
	cancel()

	// 2. Fallback: query each active node individually
	nodes, err := GetClusterNodes(ctx, client)
	if err == nil {
		for _, node := range nodes {
			vm, err := node.VirtualMachine(ctx, vmid)
			if err == nil && vm != nil {
				return node, vm, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("VM %d not found on any active node in Proxmox cluster", vmid)
}

// FindVMInConn searches for a VM across all nodes in the multi-node pool (if active)
// or across nodes in the single Proxmox cluster.
func FindVMInConn(ctx context.Context, conn types.Conn, vmid int) (*proxmox.Node, *proxmox.VirtualMachine, error) {
	if conn.NodePool != nil {
		_, node, vm, err := conn.NodePool.FindVM(ctx, vmid)
		if err == nil && vm != nil {
			return node, vm, nil
		}
	}
	return FindVMNode(ctx, conn.Proxmox, vmid)
}

