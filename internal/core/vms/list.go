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

func List(conn types.Conn, uuid string, roles ...string) ([]types.Vm, error) {
	role := ""
	if len(roles) > 0 {
		role = roles[0]
	}
	if uuid == "" && role == "" {
		role = "teacher"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	type rawVMData struct {
		vm      types.Vm
		nicTags map[string]int
	}

	var rawList []rawVMData
	tagMap := make(map[int][]vmNetEndpoint)

	pool := conn.GetPool()
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
				continue
			}
			var ownerUUID, vmUUID string

			tags := strings.Split(i.Tags, ";")
			for _, tag := range tags {
				trimmed := strings.TrimSpace(tag)
				if strings.HasPrefix(trimmed, "owner-") {
					ownerUUID = strings.TrimPrefix(trimmed, "owner-")
				} else if strings.HasPrefix(trimmed, "vm-") {
					vmUUID = strings.TrimPrefix(trimmed, "vm-")
				}
			}

			if role == "teacher" || ownerUUID == uuid {
				networks := make([]string, 0)
				availableNICs := make([]string, 0)
				nicTags := make(map[string]int)

				status := "Stopped"
				if strings.EqualFold(i.Status, "running") {
					status = "Running"
				}

				ramMB := int(i.MaxMem / (1024 * 1024))
				fullVM, err := node.VirtualMachine(ctx, int(i.VMID))

				if err == nil && fullVM.VirtualMachineConfig != nil {
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

				nodeName := nc.NodeName
				if role != "teacher" && role != "admin" {
					nodeName = ""
				}
				vm := types.Vm{
					Id:            i.VMID,
					Name:          i.Name,
					UUID:          vmUUID,
					Status:        status,
					RAM:           ramMB,
					Node:          nodeName,
					Owner:         ownerUUID,
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
