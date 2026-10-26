package vms

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
)

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
			{Name: "ipconfig0", Value: ipconfigVal},
		}

		if vm.VirtualMachineConfig != nil && len(vm.VirtualMachineConfig.Nets) == 0 {
			options = append(options, proxmox.VirtualMachineOption{
				Name:  "net0",
				Value: "virtio,bridge=vmbr0,firewall=1",
			})
		}

		task, err := vm.Config(ctx, options...)
		if err != nil {
			return fmt.Errorf("failed to configure VM IP: %w", err)
		}
		if task != nil {
			_ = task.WaitFor(ctx, 3)
		}
		return nil
	}

	ct, _, ctErr := conn.FindContainer(ctx, req.VMID, req.TargetNode)
	if ctErr == nil && ct != nil {
		if !VerifyContainerOwner(ct, userUUID, role) {
			return fmt.Errorf("unauthorized: container %d does not belong to user", req.VMID)
		}

		name := "eth0"
		bridge := "vmbr0"
		hwaddr := ""
		tag := ""

		if ct.ContainerConfig != nil && ct.ContainerConfig.Nets != nil {
			if existingNet0, exists := ct.ContainerConfig.Nets["net0"]; exists {
				parts := strings.Split(existingNet0, ",")
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

		var net0Val string
		if req.IsDHCP {
			net0Val = fmt.Sprintf("name=%s,bridge=%s,ip=dhcp", name, bridge)
		} else {
			net0Val = fmt.Sprintf("name=%s,bridge=%s,ip=%s", name, bridge, ipStr)
			if gwStr != "" {
				net0Val += fmt.Sprintf(",gw=%s", gwStr)
			}
		}
		if hwaddr != "" {
			net0Val += fmt.Sprintf(",hwaddr=%s", hwaddr)
		}
		if tag != "" {
			net0Val += fmt.Sprintf(",tag=%s", tag)
		}

		task, err := ct.Config(ctx, proxmox.ContainerOption{
			Name:  "net0",
			Value: net0Val,
		})
		if err != nil {
			return fmt.Errorf("failed to configure container IP: %w", err)
		}
		if task != nil {
			_ = task.WaitFor(ctx, 3)
		}
		return nil
	}

	return fmt.Errorf("machine %d not found in pool", req.VMID)
}
