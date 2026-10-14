package vms

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"vm-manager-backend/types"
)

type vmNetEndpoint struct {
	vmID string
	nic  string
}

func parseVLAN(netConfig string) int {
	parts := strings.Split(netConfig, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) == "tag" {
			val, _ := strconv.Atoi(strings.TrimSpace(kv[1]))
			return val
		}
	}
	return 0
}

func List(conn types.Conn, uuid string) ([]types.Vm, error) {
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

	type rawVMData struct {
		vm      types.Vm
		nicTags map[string]int
	}

	var rawList []rawVMData
	tagMap := make(map[int][]vmNetEndpoint)

	for _, i := range vms {
		if i.Template {
			continue
		}
		var ownerUUID, vmUUID string

		tags := strings.Split(i.Tags, ";")
		for _, tag := range tags {
			if strings.HasPrefix(tag, "owner-") {
				ownerUUID = strings.TrimPrefix(tag, "owner-")
			} else if strings.HasPrefix(tag, "vm-") {
				vmUUID = strings.TrimPrefix(tag, "vm-")
			}
		}

		if ownerUUID == uuid {
			networks := make([]string, 0)
			availableNICs := make([]string, 0)
			nicTags := make(map[string]int)

			status := "Stopped"
			if strings.EqualFold(i.Status, "running") {
				status = "Running"
			}

			ramMB := int(i.MaxMem / (1024 * 1024))

			// Fetch the full VM object to access its read-only configuration
			fullVM, err := node.VirtualMachine(ctx, int(i.VMID))

			if err != nil {
				fmt.Printf("failed to get vm %d: %v\n", i.VMID, err)
			} else if fullVM.VirtualMachineConfig != nil {
				if fullVM.VirtualMachineConfig.Memory > 0 {
					ramMB = int(fullVM.VirtualMachineConfig.Memory)
				}
				if fullVM.VirtualMachineConfig.Nets != nil {
					for j := 0; j <= 31; j++ {
						netKey := fmt.Sprintf("net%d", j)
						if netVal, exists := fullVM.VirtualMachineConfig.Nets[netKey]; exists {
							networks = append(networks, fmt.Sprintf("%s: %s", netKey, netVal))
							availableNICs = append(availableNICs, netKey)
							tag := parseVLAN(netVal)
							if tag > 0 {
								nicTags[netKey] = tag
								tagMap[tag] = append(tagMap[tag], vmNetEndpoint{
									vmID: fmt.Sprint(i.VMID),
									nic:  netKey,
								})
							}
						}
					}
				}
			}

			vm := types.Vm{
				Id:            i.VMID,
				Name:          i.Name,
				UUID:          vmUUID,
				Status:        status,
				RAM:           ramMB,
				Networks:      networks,
				AvailableNICs: availableNICs,
				Connections:   make([]types.VMConnection, 0),
			}

			rawList = append(rawList, rawVMData{
				vm:      vm,
				nicTags: nicTags,
			})
		}
	}

	var machines []types.Vm
	for _, item := range rawList {
		vm := item.vm
		for nic, tag := range item.nicTags {
			endpoints := tagMap[tag]
			for _, ep := range endpoints {
				if ep.vmID != fmt.Sprint(vm.Id) {
					vm.Connections = append(vm.Connections, types.VMConnection{
						TargetVMID: ep.vmID,
						LocalNIC:   nic,
						TargetNIC:  ep.nic,
					})
				}
			}
		}
		machines = append(machines, vm)
	}

	if machines == nil {
		machines = make([]types.Vm, 0)
	}

	return machines, nil
}
