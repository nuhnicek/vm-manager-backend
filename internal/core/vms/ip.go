package vms

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
)

func parseNetIdx(nic string) int {
	clean := strings.TrimPrefix(strings.ToLower(nic), "net")
	idx, err := strconv.Atoi(clean)
	if err != nil {
		return 0
	}
	return idx
}

func SetMachineIP(conn types.Conn, userUUID, role string, req types.SetIPRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var ipStr, gwStr string
	if !req.IsDHCP {
		ipStr = strings.TrimSpace(req.IP)
		if ipStr == "" {
			return fmt.Errorf("IP adresa nesmí být prázdná")
		}
		if !strings.Contains(ipStr, "/") {
			ipStr += "/24"
		}
		_, _, err := net.ParseCIDR(ipStr)
		if err != nil {
			return fmt.Errorf("neplatný formát IP adresy: %w", err)
		}
		gwStr = strings.TrimSpace(req.Gateway)
		if gwStr != "" && net.ParseIP(gwStr) == nil {
			return fmt.Errorf("neplatná IP adresa výchozí brány")
		}
	}

	targetNIC := strings.ToLower(strings.TrimSpace(req.NIC))
	if targetNIC == "" {
		targetNIC = "net0"
	}
	idx := parseNetIdx(targetNIC)

	vm, _, err := conn.FindVM(ctx, req.VMID, req.TargetNode)
	if err == nil && vm != nil {
		if !VerifyVMOwner(vm, userUUID, role) {
			return fmt.Errorf("unauthorized: VM %d does not belong to user", req.VMID)
		}

		var ipconfigVal string
		if req.IsDHCP {
			ipconfigVal = "ip=dhcp"
		} else {
			ipconfigVal = fmt.Sprintf("ip=%s", ipStr)
			if gwStr != "" {
				ipconfigVal += fmt.Sprintf(",gw=%s", gwStr)
			}
		}

		options := []proxmox.VirtualMachineOption{
			{Name: fmt.Sprintf("ipconfig%d", idx), Value: ipconfigVal},
		}

		if vm.VirtualMachineConfig != nil && vm.VirtualMachineConfig.Nets != nil {
			if _, exists := vm.VirtualMachineConfig.Nets[targetNIC]; !exists {
				options = append(options, proxmox.VirtualMachineOption{
					Name:  targetNIC,
					Value: "model=virtio,bridge=vmbr0,firewall=1",
				})
			}
		} else if idx == 0 {
			options = append(options, proxmox.VirtualMachineOption{
				Name:  "net0",
				Value: "model=virtio,bridge=vmbr0,firewall=1",
			})
		}

		_, err := vm.Config(ctx, options...)
		if err != nil {
			err = vm.ConfigSync(ctx, options...)
			if err != nil {
				return fmt.Errorf("failed to configure VM IP: %w", err)
			}
		}
		return nil
	}

	ct, _, ctErr := conn.FindContainer(ctx, req.VMID, req.TargetNode)
	if ctErr == nil && ct != nil {
		if !VerifyContainerOwner(ct, userUUID, role) {
			return fmt.Errorf("unauthorized: container %d does not belong to user", req.VMID)
		}

		name := fmt.Sprintf("eth%d", idx)
		bridge := "vmbr0"
		hwaddr := ""
		tag := ""

		if ct.ContainerConfig != nil && ct.ContainerConfig.Nets != nil {
			if existingNet, exists := ct.ContainerConfig.Nets[targetNIC]; exists {
				parts := strings.Split(existingNet, ",")
				for _, part := range parts {
					kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
					if len(kv) == 2 {
						k := strings.ToLower(kv[0])
						v := kv[1]
						if k == "name" {
							name = v
						} else if k == "bridge" {
							bridge = v
						} else if k == "hwaddr" {
							hwaddr = v
						} else if k == "tag" {
							tag = v
						}
					}
				}
			}
		}

		var netVal string
		if req.IsDHCP {
			netVal = fmt.Sprintf("name=%s,bridge=%s,ip=dhcp", name, bridge)
		} else {
			netVal = fmt.Sprintf("name=%s,bridge=%s,ip=%s", name, bridge, ipStr)
			if gwStr != "" {
				netVal += fmt.Sprintf(",gw=%s", gwStr)
			}
		}
		if hwaddr != "" {
			netVal += fmt.Sprintf(",hwaddr=%s", hwaddr)
		}
		if tag != "" {
			netVal += fmt.Sprintf(",tag=%s", tag)
		}

		_, err := ct.Config(ctx, proxmox.ContainerOption{
			Name:  targetNIC,
			Value: netVal,
		})
		if err != nil {
			return fmt.Errorf("failed to configure container IP: %w", err)
		}
		return nil
	}

	return fmt.Errorf("machine %d not found in pool", req.VMID)
}
