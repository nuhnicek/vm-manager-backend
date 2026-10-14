package vms

import (
	"context"
	"fmt"
	"strings"
	"time"
	"vm-manager-backend/types"
)

// ListTemplates returns all VM templates available in the Proxmox cluster.
// Templates are global resources for all users and have no owner.
func ListTemplates(conn types.Conn) ([]types.Vm, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, err := conn.Proxmox.Node(ctx, "pve")
	if err != nil {
		return nil, err
	}

	vms, err := node.VirtualMachines(ctx)
	if err != nil {
		return nil, err
	}

	machines := make([]types.Vm, 0)
	for _, i := range vms {
		if i.Template {
			var vmUUID string

			tags := strings.Split(i.Tags, ";")
			for _, tag := range tags {
				trimmed := strings.TrimSpace(tag)
				if strings.HasPrefix(trimmed, "vm-") {
					vmUUID = strings.TrimPrefix(trimmed, "vm-")
					break
				}
			}

			if vmUUID == "" {
				for _, tag := range tags {
					trimmed := strings.TrimSpace(tag)
					if len(trimmed) == 36 && strings.Count(trimmed, "-") == 4 {
						vmUUID = trimmed
						break
					}
				}
			}

			if vmUUID == "" {
				vmUUID = fmt.Sprintf("tpl-%d", i.VMID)
			}

			vm := types.Vm{
				Id:   i.VMID,
				Name: i.Name,
				UUID: vmUUID,
			}
			machines = append(machines, vm)
		}
	}

	return machines, nil
}
