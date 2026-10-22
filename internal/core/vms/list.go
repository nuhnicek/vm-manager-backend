package vms

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
	"go.mongodb.org/mongo-driver/v2/bson"
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

	var nodes []*proxmox.Node
	var err error
	if conn.NodePool != nil {
		nodes = conn.NodePool.GetNodes(ctx)
	}
	if len(nodes) == 0 {
		nodes, err = GetClusterNodes(ctx, conn.Proxmox)
		if err != nil {
			return nil, err
		}
	}

	userEmailMap := make(map[string]string)
	if coll := conn.Mongo.Database("lab411").Collection("users"); coll != nil {
		cursor, err := coll.Find(ctx, bson.M{})
		if err == nil {
			var users []types.UserDocument
			if err := cursor.All(ctx, &users); err == nil {
				for _, u := range users {
					userEmailMap[u.UUID] = u.Email
				}
			}
		}
	}

	type rawVMData struct {
		vm      types.Vm
		nicTags map[string]int
	}

	var rawList []rawVMData
	tagMap := make(map[int][]vmNetEndpoint)

	for _, node := range nodes {
		vms, err := node.VirtualMachines(ctx)
		if err != nil {
			continue
		}

		for _, i := range vms {
			if bool(i.Template) {
				continue
			}
		var ownerUUID, vmUUID, labID string

		tags := strings.Split(i.Tags, ";")
		for _, tag := range tags {
			trimmed := strings.TrimSpace(tag)
			if strings.HasPrefix(trimmed, "owner-") {
				ownerUUID = strings.TrimPrefix(trimmed, "owner-")
			} else if strings.HasPrefix(trimmed, "vm-") {
				vmUUID = strings.TrimPrefix(trimmed, "vm-")
			} else if strings.HasPrefix(trimmed, "lab-") {
				labID = strings.TrimPrefix(trimmed, "lab-")
			}
		}

		if uuid == "" || ownerUUID == uuid {
			networks := make([]string, 0)
			availableNICs := make([]string, 0)
			nicTags := make(map[string]int)

			status := "Stopped"
			if strings.EqualFold(i.Status, "running") {
				status = "Running"
			}

			ramMB := int(i.MaxMem / (1024 * 1024))
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

			ownerEmail := userEmailMap[ownerUUID]

			vm := types.Vm{
				Id:            i.VMID,
				Name:          i.Name,
				UUID:          vmUUID,
				OwnerUUID:     ownerUUID,
				OwnerEmail:    ownerEmail,
				LabID:         labID,
				Node:          node.Name,
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
