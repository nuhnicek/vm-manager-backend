package vms

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"vm-manager-backend/internal/core/projects"
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

func parseIPAndGateway(configStr string) (string, string) {
	var ip, gw string
	parts := strings.Split(configStr, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			k := strings.ToLower(strings.TrimSpace(kv[0]))
			v := strings.TrimSpace(kv[1])
			if k == "ip" {
				ip = v
			} else if k == "gw" || k == "gw4" || k == "gateway" {
				gw = v
			}
		}
	}
	return ip, gw
}

func isVulnerableTarget(name, tags string) bool {
	lower := strings.ToLower(name + ";" + tags)
	keywords := []string{"vulnerable", "vuln", "metasploit", "weblab", "ctf", "dvwa", "juice"}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func List(conn types.Conn, uuid string, roles ...string) ([]types.Vm, error) {
	role := ""
	if len(roles) > 0 {
		role = roles[0]
	}
	if uuid == "" && role == "" {
		role = "teacher"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	type rawVMData struct {
		vm      types.Vm
		nicTags map[string]int
	}

	var rawList []rawVMData
	tagMap := make(map[int][]vmNetEndpoint)

	projectsMap, _ := projects.GetVMProjectsMap(conn, uuid, role == "teacher" || role == "admin")

	pool := conn.GetPool()
	for _, nc := range pool {
		node, err := nc.Client.Node(ctx, nc.NodeName)
		if err != nil {
			continue
		}

		vms, err := node.VirtualMachines(ctx)
		if err == nil {
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

				if role == "teacher" || role == "admin" || ownerUUID == uuid {
					networks := make([]string, 0)
					availableNICs := make([]string, 0)
					nicTags := make(map[string]int)

					status := "Stopped"
					if strings.EqualFold(i.Status, "running") {
						status = "Running"
					}

					ramMB := int(i.MaxMem / (1024 * 1024))
					fullVM, err := node.VirtualMachine(ctx, int(i.VMID))

					var cfgIP, cfgGW, runtimeIP string
					if err == nil && fullVM != nil && fullVM.VirtualMachineConfig != nil {
						if fullVM.VirtualMachineConfig.Memory > 0 {
							ramMB = int(fullVM.VirtualMachineConfig.Memory)
						}
						if fullVM.VirtualMachineConfig.IPConfigs != nil {
							if ipVal, exists := fullVM.VirtualMachineConfig.IPConfigs["ipconfig0"]; exists {
								cfgIP, cfgGW = parseIPAndGateway(ipVal)
							}
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

					if status == "Running" && fullVM != nil {
						agentCtx, agentCancel := context.WithTimeout(ctx, 600*time.Millisecond)
						if ifaces, err := fullVM.AgentGetNetworkIFaces(agentCtx); err == nil && ifaces != nil {
							for _, iface := range ifaces {
								if iface.Name != "lo" {
									for _, ipInfo := range iface.IPAddresses {
										if ipInfo.IPAddressType == "ipv4" && ipInfo.IPAddress != "127.0.0.1" {
											runtimeIP = ipInfo.IPAddress
											break
										}
									}
								}
								if runtimeIP != "" {
									break
								}
							}
						}
						agentCancel()
					}

					if runtimeIP == "" && cfgIP != "" && !strings.EqualFold(cfgIP, "dhcp") {
						runtimeIP = strings.Split(cfgIP, "/")[0]
					}

					isVuln := isVulnerableTarget(i.Name, i.Tags)
					category := "Virtual Machine"
					if isVuln {
						category = "Vulnerable Target"
					}

					nodeName := nc.NodeName
					if role != "teacher" && role != "admin" {
						nodeName = ""
					}

					projID := ""
					projName := ""
					projColor := ""
					if p, exists := projectsMap[int(i.VMID)]; exists {
						projID = p.ID
						projName = p.Name
						projColor = p.Color
					}

					vm := types.Vm{
						Id:            i.VMID,
						Name:          i.Name,
						UUID:          vmUUID,
						Type:          "qemu",
						Status:        status,
						RAM:           ramMB,
						Node:          nodeName,
						Owner:         ownerUUID,
						IPAddress:     runtimeIP,
						ConfiguredIP:  cfgIP,
						Gateway:       cfgGW,
						IsVulnerable:  isVuln,
						Category:      category,
						ProjectID:     projID,
						ProjectName:   projName,
						ProjectColor:  projColor,
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

		cts, err := node.Containers(ctx)
		if err == nil {
			for _, ct := range cts {
				var ownerUUID, vmUUID string

				tags := strings.Split(ct.Tags, ";")
				for _, tag := range tags {
					trimmed := strings.TrimSpace(tag)
					if strings.HasPrefix(trimmed, "owner-") {
						ownerUUID = strings.TrimPrefix(trimmed, "owner-")
					} else if strings.HasPrefix(trimmed, "vm-") {
						vmUUID = strings.TrimPrefix(trimmed, "vm-")
					}
				}

				if role == "teacher" || role == "admin" || ownerUUID == uuid {
					networks := make([]string, 0)
					availableNICs := make([]string, 0)
					nicTags := make(map[string]int)

					status := "Stopped"
					if strings.EqualFold(ct.Status, "running") {
						status = "Running"
					}

					ramMB := int(ct.MaxMem / (1024 * 1024))
					fullCT, err := node.Container(ctx, int(ct.VMID))

					var cfgIP, cfgGW, runtimeIP string
					if err == nil && fullCT != nil {
						if fullCT.ContainerConfig != nil {
							if fullCT.ContainerConfig.Memory != nil && *fullCT.ContainerConfig.Memory > 0 {
								ramMB = *fullCT.ContainerConfig.Memory
							}
							if fullCT.ContainerConfig.Nets != nil {
								for j := 0; j <= 31; j++ {
									netKey := fmt.Sprintf("net%d", j)
									if netVal, exists := fullCT.ContainerConfig.Nets[netKey]; exists {
										networks = append(networks, fmt.Sprintf("%s: %s", netKey, netVal))
										availableNICs = append(availableNICs, netKey)
										tag := parseVLAN(netVal)
										if tag > 0 {
											nicTags[netKey] = tag
											tagMap[tag] = append(tagMap[tag], vmNetEndpoint{
												vmID: fmt.Sprint(ct.VMID),
												nic:  netKey,
											})
										}
										if netKey == "net0" {
											cfgIP, cfgGW = parseIPAndGateway(netVal)
										}
									}
								}
							}
						}

						if status == "Running" {
							ctCtx, ctCancel := context.WithTimeout(ctx, 1*time.Second)
							if ifaces, err := fullCT.Interfaces(ctCtx); err == nil && ifaces != nil {
								for _, iface := range ifaces {
									if iface.Name != "lo" && iface.Inet != "" {
										runtimeIP = strings.Split(iface.Inet, "/")[0]
										break
									}
								}
							}
							ctCancel()
						}
					}

					if runtimeIP == "" && cfgIP != "" && !strings.EqualFold(cfgIP, "dhcp") {
						runtimeIP = strings.Split(cfgIP, "/")[0]
					}

					isVuln := isVulnerableTarget(ct.Name, ct.Tags)
					category := "LXC Container"
					if isVuln {
						category = "Vulnerable Target"
					}

					nodeName := nc.NodeName
					if role != "teacher" && role != "admin" {
						nodeName = ""
					}

					projID := ""
					projName := ""
					projColor := ""
					if p, exists := projectsMap[int(ct.VMID)]; exists {
						projID = p.ID
						projName = p.Name
						projColor = p.Color
					}

					vm := types.Vm{
						Id:            ct.VMID,
						Name:          ct.Name,
						UUID:          vmUUID,
						Type:          "lxc",
						Status:        status,
						RAM:           ramMB,
						Node:          nodeName,
						Owner:         ownerUUID,
						IPAddress:     runtimeIP,
						ConfiguredIP:  cfgIP,
						Gateway:       cfgGW,
						IsVulnerable:  isVuln,
						Category:      category,
						ProjectID:     projID,
						ProjectName:   projName,
						ProjectColor:  projColor,
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
