package vms

import (
	"context"
	"fmt"
	"strings"
	"time"
	"vm-manager-backend/types"
)

func ListTemplates(conn types.Conn) ([]types.Vm, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool := conn.GetPool()
	machines := make([]types.Vm, 0)
	seen := make(map[uint64]bool)

	for _, nc := range pool {
		node, err := nc.Client.Node(ctx, nc.NodeName)
		if err != nil {
			continue
		}

		vms, err := node.VirtualMachines(ctx)
		if err != nil {
			continue
		}

		for _, i := range vms {
			if i.Template {
				if seen[uint64(i.VMID)] {
					continue
				}
				seen[uint64(i.VMID)] = true

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
	}

	return machines, nil
}
