package vms

import (
	"context"
	"fmt"
	"time"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
)

// ListTemplates dynamically discovers all VM templates across all nodes in the Proxmox cluster.
// Templates are identified by Proxmox's native `template` flag, irrespective of whether tags exist.
func ListTemplates(conn types.Conn) ([]types.Vm, error) {
	if conn.Proxmox == nil {
		return nil, fmt.Errorf("proxmox client is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if conn.NodePool != nil {
		return conn.NodePool.ListAllTemplates(ctx)
	}

	machines := make([]types.Vm, 0)
	seen := make(map[uint64]bool)

	// 1. Try cluster resources first (covers all nodes in a single API call)
	if cluster, err := conn.Proxmox.Cluster(ctx); err == nil && cluster != nil {
		if resources, err := cluster.Resources(ctx, "vm"); err == nil {
			for _, r := range resources {
				if r.Template > 0 && !seen[r.VMID] {
					seen[r.VMID] = true
					vmUUID := ExtractUUIDFromTags(r.Tags, r.VMID)
					machines = append(machines, types.Vm{
						Id:   proxmox.StringOrUint64(r.VMID),
						Name: r.Name,
						UUID: vmUUID,
						Node: r.Node,
					})
				}
			}
			if len(machines) > 0 {
				return machines, nil
			}
		}
	}

	// 2. Fallback: iterate each active node in the cluster
	nodes, err := GetClusterNodes(ctx, conn.Proxmox)
	if err == nil {
		for _, node := range nodes {
			vms, err := node.VirtualMachines(ctx)
			if err != nil {
				continue
			}
			for _, i := range vms {
				if bool(i.Template) && !seen[uint64(i.VMID)] {
					seen[uint64(i.VMID)] = true
					vmUUID := ExtractUUIDFromTags(i.Tags, uint64(i.VMID))
					machines = append(machines, types.Vm{
						Id:   i.VMID,
						Name: i.Name,
						UUID: vmUUID,
						Node: node.Name,
					})
				}
			}
		}
	}

	return machines, nil
}
