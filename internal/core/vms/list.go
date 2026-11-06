package vms

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"vm-manager-backend/internal/core/projects"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
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

func IsVulnerableTarget(name, tags string) bool {
	lower := strings.ToLower(name + ";" + tags)
	keywords := []string{"vulnerable", "vuln", "metasploit", "weblab", "ctf", "dvwa", "juice", "target", "victim", "cve", "php2shell", "kioptrix", "vulnhub"}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func isAttackerMachine(name, tags string) bool {
	lower := strings.ToLower(name + ";" + tags)
	keywords := []string{"kali", "attacker", "attack", "pentest", "parrot", "blackarch"}
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

	var mu sync.Mutex
	type rawVMData struct {
		vm      types.Vm
		nicTags map[string]int
	}

	var rawList []rawVMData
	tagMap := make(map[int][]vmNetEndpoint)

	projectsMap, _ := projects.GetVMProjectsMap(conn, uuid, role == "teacher" || role == "admin")

	pool := conn.GetPool()
	sem := make(chan struct{}, 12)
	var wg sync.WaitGroup

	for _, nc := range pool {
		nodeClient := nc
		node, err := nodeClient.Client.Node(ctx, nodeClient.NodeName)
		if err != nil {
			continue
		}

		vms, err := node.VirtualMachines(ctx)
		if err == nil {
			for _, item := range vms {
				if item.Template {
					continue
				}
				i := item
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
					wg.Add(1)
					sem <- struct{}{}
					go func(res *proxmox.VirtualMachine, n *proxmox.Node, nc types.ProxmoxNodeClient, owner, vmuuid string) {
						defer wg.Done()
						defer func() { <-sem }()

						networks := make([]string, 0)
						availableNICs := make([]string, 0)
						nicTags := make(map[string]int)
						hasInternet := false
						isAttacker := isAttackerMachine(res.Name, res.Tags)

						status := "Stopped"
						if strings.EqualFold(res.Status, "running") {
							status = "Running"
						}

						ramMB := int(res.MaxMem / (1024 * 1024))
						fullVM, vmErr := n.VirtualMachine(ctx, int(res.VMID))

						var cfgIP, cfgGW, runtimeIP string
						if vmErr == nil && fullVM != nil && fullVM.VirtualMachineConfig != nil {
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
										if tag > 0 && tag != 4094 {
											nicTags[netKey] = tag
										} else if tag == 0 {
											hasInternet = true
										}
									}
								}
							}
						}

						if status == "Running" && fullVM != nil {
							agentCtx, agentCancel := context.WithTimeout(ctx, 600*time.Millisecond)
							if ifaces, aErr := fullVM.AgentGetNetworkIFaces(agentCtx); aErr == nil && ifaces != nil {
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

						isVuln := IsVulnerableTarget(res.Name, res.Tags)
						category := "Virtual Machine"
						if isVuln {
							category = "Vulnerable Target"
						}

						var cveID, cvss, exploitHint, exploitDoc string
						var port int
						if catItem := MatchCatalogItem(res.Name, res.Tags); catItem != nil {
							isVuln = true
							category = catItem.Category
							cveID = catItem.CVEID
							cvss = catItem.CVSS
							port = catItem.Port
							targetIP := runtimeIP
							if targetIP == "" {
								targetIP = "<IP>"
							}
							exploitHint = strings.ReplaceAll(catItem.ExploitHint, "{IP}", targetIP)
							exploitDoc = catItem.ExploitDoc
						}

						nodeName := nc.NodeName
						if role != "teacher" && role != "admin" {
							nodeName = ""
						}

						projID := ""
						projName := ""
						projColor := ""
						if p, exists := projectsMap[int(res.VMID)]; exists {
							projID = p.ID
							projName = p.Name
							projColor = p.Color
						}

						vm := types.Vm{
							Id:            proxmox.StringOrUint64(res.VMID),
							Name:          res.Name,
							UUID:          vmuuid,
							Type:          "qemu",
							Status:        status,
							RAM:           ramMB,
							Node:          nodeName,
							Owner:         owner,
							IPAddress:     runtimeIP,
							ConfiguredIP:  cfgIP,
							Gateway:       cfgGW,
							IsVulnerable:  isVuln,
							Category:      category,
							CVEID:         cveID,
							CVSS:          cvss,
							Port:          port,
							ExploitHint:   exploitHint,
							ExploitDoc:    exploitDoc,
							ProjectID:     projID,
							ProjectName:   projName,
							ProjectColor:  projColor,
							Networks:      networks,
							AvailableNICs: availableNICs,
							Connections:   make([]types.VMConnection, 0),
							HasInternet:   hasInternet,
							IsAttacker:    isAttacker,
						}

						mu.Lock()
						for nicKey, tag := range nicTags {
							tagMap[tag] = append(tagMap[tag], vmNetEndpoint{
								vmID: fmt.Sprint(res.VMID),
								nic:  nicKey,
							})
						}
						rawList = append(rawList, rawVMData{
							vm:      vm,
							nicTags: nicTags,
						})
						mu.Unlock()
					}(i, node, nodeClient, ownerUUID, vmUUID)
				}
			}
		}

		cts, err := node.Containers(ctx)
		if err == nil {
			for _, ct := range cts {
				item := ct
				var ownerUUID, vmUUID string

				tags := strings.Split(item.Tags, ";")
				for _, tag := range tags {
					trimmed := strings.TrimSpace(tag)
					if strings.HasPrefix(trimmed, "owner-") {
						ownerUUID = strings.TrimPrefix(trimmed, "owner-")
					} else if strings.HasPrefix(trimmed, "vm-") {
						vmUUID = strings.TrimPrefix(trimmed, "vm-")
					}
				}

				if role == "teacher" || role == "admin" || ownerUUID == uuid {
					wg.Add(1)
					sem <- struct{}{}
					go func(res *proxmox.Container, n *proxmox.Node, nc types.ProxmoxNodeClient, owner, vmuuid string) {
						defer wg.Done()
						defer func() { <-sem }()

						networks := make([]string, 0)
						availableNICs := make([]string, 0)
						nicTags := make(map[string]int)
						hasInternet := false
						isAttacker := isAttackerMachine(res.Name, res.Tags)

						status := "Stopped"
						if strings.EqualFold(res.Status, "running") {
							status = "Running"
						}

						ramMB := int(res.MaxMem / (1024 * 1024))
						fullCT, ctErr := n.Container(ctx, int(res.VMID))

						var cfgIP, cfgGW, runtimeIP string
						if ctErr == nil && fullCT != nil {
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
											if tag > 0 && tag != 4094 {
												nicTags[netKey] = tag
											} else if tag == 0 {
												hasInternet = true
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
								if ifaces, iErr := fullCT.Interfaces(ctCtx); iErr == nil && ifaces != nil {
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

						isVuln := IsVulnerableTarget(res.Name, res.Tags)
						category := "LXC Container"
						if isVuln {
							category = "Vulnerable Target"
						}

						var cveID, cvss, exploitHint, exploitDoc string
						var port int
						if catItem := MatchCatalogItem(res.Name, res.Tags); catItem != nil {
							isVuln = true
							category = catItem.Category
							cveID = catItem.CVEID
							cvss = catItem.CVSS
							port = catItem.Port
							targetIP := runtimeIP
							if targetIP == "" {
								targetIP = "<IP>"
							}
							exploitHint = strings.ReplaceAll(catItem.ExploitHint, "{IP}", targetIP)
							exploitDoc = catItem.ExploitDoc
						}

						nodeName := nc.NodeName
						if role != "teacher" && role != "admin" {
							nodeName = ""
						}

						projID := ""
						projName := ""
						projColor := ""
						if p, exists := projectsMap[int(res.VMID)]; exists {
							projID = p.ID
							projName = p.Name
							projColor = p.Color
						}

						vm := types.Vm{
							Id:            proxmox.StringOrUint64(res.VMID),
							Name:          res.Name,
							UUID:          vmuuid,
							Type:          "lxc",
							Status:        status,
							RAM:           ramMB,
							Node:          nodeName,
							Owner:         owner,
							IPAddress:     runtimeIP,
							ConfiguredIP:  cfgIP,
							Gateway:       cfgGW,
							IsVulnerable:  isVuln,
							Category:      category,
							CVEID:         cveID,
							CVSS:          cvss,
							Port:          port,
							ExploitHint:   exploitHint,
							ExploitDoc:    exploitDoc,
							ProjectID:     projID,
							ProjectName:   projName,
							ProjectColor:  projColor,
							Networks:      networks,
							AvailableNICs: availableNICs,
							Connections:   make([]types.VMConnection, 0),
							HasInternet:   hasInternet,
							IsAttacker:    isAttacker,
						}

						mu.Lock()
						for nicKey, tag := range nicTags {
							tagMap[tag] = append(tagMap[tag], vmNetEndpoint{
								vmID: fmt.Sprint(res.VMID),
								nic:  nicKey,
							})
						}
						rawList = append(rawList, rawVMData{
							vm:      vm,
							nicTags: nicTags,
						})
						mu.Unlock()
					}(item, node, nodeClient, ownerUUID, vmUUID)
				}
			}
		}
	}
	wg.Wait()

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

	sort.Slice(machines, func(i, j int) bool {
		return machines[i].Id < machines[j].Id
	})

	return machines, nil
}
