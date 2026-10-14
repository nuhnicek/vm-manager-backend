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

func verifyOwner(vm *proxmox.VirtualMachine, userUUID string) bool {
	if userUUID == "" {
		return true
	}
	tags := strings.Split(vm.Tags, ";")
	for _, tag := range tags {
		if strings.TrimSpace(tag) == "owner-"+userUUID {
			return true
		}
	}
	return false
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

func AddNIC(conn types.Conn, userUUID string, vmID string) (string, error) {
	id, err := strconv.Atoi(vmID)
	if err != nil {
		return "", fmt.Errorf("invalid vmid: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	node, err := conn.Proxmox.Node(ctx, "pve")
	if err != nil {
		return "", fmt.Errorf("failed to get node: %w", err)
	}

	vm, err := node.VirtualMachine(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to get VM %d: %w", id, err)
	}

	if !verifyOwner(vm, userUUID) {
		return "", fmt.Errorf("unauthorized: VM %d does not belong to user", id)
	}

	netKey, err := GetNextAvailableNet(vm)
	if err != nil {
		return "", err
	}

	err = vm.ConfigSync(ctx, proxmox.VirtualMachineOption{
		Name:  netKey,
		Value: "model=virtio,bridge=vmbr0,firewall=1",
	})
	if err != nil {
		return "", fmt.Errorf("failed to add NIC %s to VM %d: %w", netKey, id, err)
	}

	return netKey, nil
}

func RemoveNIC(conn types.Conn, userUUID string, vmID string, nic string) error {
	id, err := strconv.Atoi(vmID)
	if err != nil {
		return fmt.Errorf("invalid vmid: %w", err)
	}
	if nic == "" {
		return fmt.Errorf("nic name cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	node, err := conn.Proxmox.Node(ctx, "pve")
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	vm, err := node.VirtualMachine(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get VM %d: %w", id, err)
	}

	if !verifyOwner(vm, userUUID) {
		return fmt.Errorf("unauthorized: VM %d does not belong to user", id)
	}

	err = vm.ConfigSync(ctx, proxmox.VirtualMachineOption{
		Name:  "delete",
		Value: nic,
	})
	if err != nil {
		return fmt.Errorf("failed to remove NIC %s from VM %d: %w", nic, id, err)
	}

	return nil
}

func DisconnectVLAN(conn types.Conn, userUUID string, vm1id string, vm2id string) error {
	id1, err := strconv.Atoi(vm1id)
	if err != nil {
		return fmt.Errorf("invalid vm1id: %w", err)
	}
	id2, err := strconv.Atoi(vm2id)
	if err != nil {
		return fmt.Errorf("invalid vm2id: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	node, err := conn.Proxmox.Node(ctx, "pve")
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	vm1, err := node.VirtualMachine(ctx, id1)
	if err != nil {
		return fmt.Errorf("failed to get VM %d: %w", id1, err)
	}
	vm2, err := node.VirtualMachine(ctx, id2)
	if err != nil {
		return fmt.Errorf("failed to get VM %d: %w", id2, err)
	}

	if !verifyOwner(vm1, userUUID) || !verifyOwner(vm2, userUUID) {
		return fmt.Errorf("unauthorized: VMs do not belong to user")
	}

	if vm1.VirtualMachineConfig == nil || vm1.VirtualMachineConfig.Nets == nil ||
		vm2.VirtualMachineConfig == nil || vm2.VirtualMachineConfig.Nets == nil {
		return nil
	}

	for net1Key, net1Val := range vm1.VirtualMachineConfig.Nets {
		tag1 := ParseVLAN(net1Val)
		if tag1 == 0 {
			continue
		}
		for net2Key, net2Val := range vm2.VirtualMachineConfig.Nets {
			tag2 := ParseVLAN(net2Val)
			if tag1 == tag2 {
				_ = vm1.ConfigSync(ctx, proxmox.VirtualMachineOption{
					Name:  "delete",
					Value: net1Key,
				})
				_ = vm2.ConfigSync(ctx, proxmox.VirtualMachineOption{
					Name:  "delete",
					Value: net2Key,
				})
			}
		}
	}

	return nil
}
