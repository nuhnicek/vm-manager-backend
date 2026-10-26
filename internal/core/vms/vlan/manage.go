package vlan

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
)

type MachineEndpoint struct {
	ID     int
	IsCT   bool
	VM     *proxmox.VirtualMachine
	CT     *proxmox.Container
	Client *types.ProxmoxNodeClient
	Nets   map[string]string
	Tags   string
}

func parseNetIndex(nic string) int {
	clean := strings.TrimPrefix(nic, "net")
	idx, err := strconv.Atoi(clean)
	if err != nil {
		return 0
	}
	return idx
}

func removeTagFromNetConfig(config string) string {
	parts := strings.Split(config, ",")
	var kept []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if strings.HasPrefix(trimmed, "tag=") || trimmed == "tag" {
			continue
		}
		if trimmed != "" {
			kept = append(kept, trimmed)
		}
	}
	return strings.Join(kept, ",")
}

func verifyOwner(tags string, userUUID string, roles ...string) bool {
	for _, role := range roles {
		if role == "teacher" || role == "admin" {
			return true
		}
	}
	if userUUID == "" {
		return true
	}
	parts := strings.Split(tags, ";")
	for _, tag := range parts {
		if strings.TrimSpace(tag) == "owner-"+userUUID {
			return true
		}
	}
	return false
}

func FindEndpoint(ctx context.Context, conn types.Conn, id int) (*MachineEndpoint, error) {
	vm, nodeClient, err := conn.FindVM(ctx, id)
	if err == nil && vm != nil {
		nets := make(map[string]string)
		if vm.VirtualMachineConfig != nil && vm.VirtualMachineConfig.Nets != nil {
			for k, v := range vm.VirtualMachineConfig.Nets {
				nets[k] = v
			}
		}
		return &MachineEndpoint{
			ID:     id,
			IsCT:   false,
			VM:     vm,
			Client: nodeClient,
			Nets:   nets,
			Tags:   vm.Tags,
		}, nil
	}
	ct, nodeClient, ctErr := conn.FindContainer(ctx, id)
	if ctErr == nil && ct != nil {
		nets := make(map[string]string)
		if ct.ContainerConfig != nil && ct.ContainerConfig.Nets != nil {
			for k, v := range ct.ContainerConfig.Nets {
				nets[k] = v
			}
		}
		return &MachineEndpoint{
			ID:     id,
			IsCT:   true,
			CT:     ct,
			Client: nodeClient,
			Nets:   nets,
			Tags:   ct.Tags,
		}, nil
	}
	return nil, fmt.Errorf("machine %d not found in pool", id)
}

func (m *MachineEndpoint) NextAvailableNet() (string, error) {
	for i := 1; i <= 31; i++ {
		key := fmt.Sprintf("net%d", i)
		if _, exists := m.Nets[key]; !exists {
			return key, nil
		}
	}
	return "", fmt.Errorf("maximum network interfaces reached for machine %d", m.ID)
}

func (m *MachineEndpoint) SetNIC(ctx context.Context, nic string, value string) error {
	if m.IsCT {
		task, err := m.CT.Config(ctx, proxmox.ContainerOption{
			Name:  nic,
			Value: value,
		})
		if err != nil {
			return err
		}
		if task != nil {
			_ = task.WaitFor(ctx, 3)
		}
		m.Nets[nic] = value
		return nil
	}
	err := m.VM.ConfigSync(ctx, proxmox.VirtualMachineOption{
		Name:  nic,
		Value: value,
	})
	if err != nil {
		return err
	}
	m.Nets[nic] = value
	return nil
}

func (m *MachineEndpoint) DeleteNIC(ctx context.Context, nic string) error {
	if m.IsCT {
		task, err := m.CT.Config(ctx, proxmox.ContainerOption{
			Name:  "delete",
			Value: nic,
		})
		if err != nil {
			return err
		}
		if task != nil {
			_ = task.WaitFor(ctx, 3)
		}
		delete(m.Nets, nic)
		return nil
	}
	err := m.VM.ConfigSync(ctx, proxmox.VirtualMachineOption{
		Name:  "delete",
		Value: nic,
	})
	if err != nil {
		return err
	}
	delete(m.Nets, nic)
	return nil
}

func (m *MachineEndpoint) UntagNIC(ctx context.Context, nic string) error {
	val, exists := m.Nets[nic]
	if !exists || val == "" {
		return nil
	}
	newVal := removeTagFromNetConfig(val)
	if newVal == "" {
		if nic == "net0" {
			if m.IsCT {
				newVal = "name=eth0,bridge=vmbr0,ip=dhcp"
			} else {
				newVal = "model=virtio,bridge=vmbr0,firewall=1"
			}
		} else {
			return m.DeleteNIC(ctx, nic)
		}
	}
	return m.SetNIC(ctx, nic, newVal)
}

func (m *MachineEndpoint) ApplyVLAN(ctx context.Context, nic string, vlanTag int) error {
	existing, exists := m.Nets[nic]
	if exists && existing != "" {
		cleaned := removeTagFromNetConfig(existing)
		newVal := fmt.Sprintf("%s,tag=%d", cleaned, vlanTag)
		return m.SetNIC(ctx, nic, newVal)
	}
	var newVal string
	if m.IsCT {
		idx := parseNetIndex(nic)
		newVal = fmt.Sprintf("name=eth%d,bridge=vmbr0,tag=%d", idx, vlanTag)
	} else {
		newVal = fmt.Sprintf("model=virtio,bridge=vmbr0,tag=%d,firewall=1", vlanTag)
	}
	return m.SetNIC(ctx, nic, newVal)
}

func ParseVLAN(netConfig string) int {
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

func AddNIC(conn types.Conn, userUUID string, vmID string, roles ...string) (string, error) {
	id, err := strconv.Atoi(vmID)
	if err != nil {
		return "", fmt.Errorf("invalid vmid: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m, err := FindEndpoint(ctx, conn, id)
	if err != nil {
		return "", err
	}

	if !verifyOwner(m.Tags, userUUID, roles...) {
		return "", fmt.Errorf("unauthorized: machine %d does not belong to user", id)
	}

	netKey, err := m.NextAvailableNet()
	if err != nil {
		return "", err
	}

	var val string
	if m.IsCT {
		idx := parseNetIndex(netKey)
		val = fmt.Sprintf("name=eth%d,bridge=vmbr0", idx)
	} else {
		val = "model=virtio,bridge=vmbr0,firewall=1"
	}

	err = m.SetNIC(ctx, netKey, val)
	if err != nil {
		return "", fmt.Errorf("failed to add NIC %s: %w", netKey, err)
	}

	return netKey, nil
}

func RemoveNIC(conn types.Conn, userUUID string, vmID string, nic string, roles ...string) error {
	id, err := strconv.Atoi(vmID)
	if err != nil {
		return fmt.Errorf("invalid vmid: %w", err)
	}
	if nic == "" {
		return fmt.Errorf("nic name cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m, err := FindEndpoint(ctx, conn, id)
	if err != nil {
		return err
	}

	if !verifyOwner(m.Tags, userUUID, roles...) {
		return fmt.Errorf("unauthorized: machine %d does not belong to user", id)
	}

	return m.DeleteNIC(ctx, nic)
}

func DisconnectVLAN(conn types.Conn, userUUID string, vm1id string, vm2id string, nic1 string, nic2 string, roles ...string) error {
	id1, err := strconv.Atoi(vm1id)
	if err != nil {
		return fmt.Errorf("invalid vm1id: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m1, err := FindEndpoint(ctx, conn, id1)
	if err != nil {
		return err
	}

	if !verifyOwner(m1.Tags, userUUID, roles...) {
		return fmt.Errorf("unauthorized: machine %d does not belong to user", id1)
	}

	id2, _ := strconv.Atoi(vm2id)
	if id2 != 0 {
		m2, err := FindEndpoint(ctx, conn, id2)
		if err != nil {
			return err
		}
		if !verifyOwner(m2.Tags, userUUID, roles...) {
			return fmt.Errorf("unauthorized: machine %d does not belong to user", id2)
		}

		matched := false
		for k1, v1 := range m1.Nets {
			tag1 := ParseVLAN(v1)
			if tag1 == 0 {
				continue
			}
			if nic1 != "" && k1 != nic1 {
				continue
			}
			for k2, v2 := range m2.Nets {
				tag2 := ParseVLAN(v2)
				if tag2 == 0 {
					continue
				}
				if nic2 != "" && k2 != nic2 {
					continue
				}
				if tag1 == tag2 {
					matched = true
					_ = m1.UntagNIC(ctx, k1)
					_ = m2.UntagNIC(ctx, k2)
				}
			}
		}

		if !matched {
			if nic1 != "" {
				_ = m1.UntagNIC(ctx, nic1)
			}
			if nic2 != "" {
				_ = m2.UntagNIC(ctx, nic2)
			}
		}
		return nil
	}

	if nic1 != "" {
		val := m1.Nets[nic1]
		tag := ParseVLAN(val)
		_ = m1.UntagNIC(ctx, nic1)
		if tag > 0 {
			pool := conn.GetPool()
			for _, nc := range pool {
				node, err := nc.Client.Node(ctx, nc.NodeName)
				if err != nil {
					continue
				}
				vms, err := node.VirtualMachines(ctx)
				if err == nil {
					for _, vm := range vms {
						if int(vm.VMID) == id1 || vm.Template {
							continue
						}
						fvm, err := node.VirtualMachine(ctx, int(vm.VMID))
						if err == nil && fvm != nil && fvm.VirtualMachineConfig != nil && fvm.VirtualMachineConfig.Nets != nil {
							for nk, nv := range fvm.VirtualMachineConfig.Nets {
								if ParseVLAN(nv) == tag {
									cleanVal := removeTagFromNetConfig(nv)
									if cleanVal == "" {
										cleanVal = "model=virtio,bridge=vmbr0,firewall=1"
									}
									_ = fvm.ConfigSync(ctx, proxmox.VirtualMachineOption{
										Name:  nk,
										Value: cleanVal,
									})
								}
							}
						}
					}
				}
				cts, err := node.Containers(ctx)
				if err == nil {
					for _, ct := range cts {
						if int(ct.VMID) == id1 {
							continue
						}
						fct, err := node.Container(ctx, int(ct.VMID))
						if err == nil && fct != nil && fct.ContainerConfig != nil && fct.ContainerConfig.Nets != nil {
							for nk, nv := range fct.ContainerConfig.Nets {
								if ParseVLAN(nv) == tag {
									cleanVal := removeTagFromNetConfig(nv)
									if cleanVal == "" {
										cleanVal = "name=eth0,bridge=vmbr0,ip=dhcp"
									}
									t, _ := fct.Config(ctx, proxmox.ContainerOption{
										Name:  nk,
										Value: cleanVal,
									})
									if t != nil {
										_ = t.WaitFor(ctx, 3)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}
