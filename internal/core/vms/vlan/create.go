package vlan

import (
	"context"
	"fmt"
	"strconv"
	"time"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
)

func ConnectVLAN(vm1id string, nic1 string, vm2id string, nic2 string, conn types.Conn, userUUID string) error {
	id1, err := strconv.Atoi(vm1id)
	if err != nil {
		return fmt.Errorf("invalid vm1id: %w", err)
	}
	id2, err := strconv.Atoi(vm2id)
	if err != nil {
		return fmt.Errorf("invalid vm2id: %w", err)
	}

	vlanTag, err := GetNextVLAN(conn)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, vm1, err := vms.FindVMNode(ctx, conn.Proxmox, id1)
	if err != nil {
		return fmt.Errorf("failed to get vm1: %w", err)
	}

	_, vm2, err := vms.FindVMNode(ctx, conn.Proxmox, id2)
	if err != nil {
		return fmt.Errorf("failed to get vm2: %w", err)
	}

	if !verifyOwner(vm1, userUUID) || !verifyOwner(vm2, userUUID) {
		return fmt.Errorf("unauthorized: VMs do not belong to user")
	}

	vm1Net := nic1
	if vm1Net == "" {
		vm1Net, err = GetNextAvailableNet(vm1)
		if err != nil {
			return err
		}
	}

	vm2Net := nic2
	if vm2Net == "" {
		vm2Net, err = GetNextAvailableNet(vm2)
		if err != nil {
			return err
		}
	}

	networkConfig := fmt.Sprintf("model=virtio,bridge=vmbr0,tag=%d,firewall=1", vlanTag)

	err = vm1.ConfigSync(ctx, proxmox.VirtualMachineOption{
		Name:  vm1Net,
		Value: networkConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to connect vm1 to network %d: %w", vlanTag, err)
	}

	err = vm2.ConfigSync(ctx, proxmox.VirtualMachineOption{
		Name:  vm2Net,
		Value: networkConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to connect vm2 to network %d: %w", vlanTag, err)
	}
	return nil
}

func CreateVLAN(vm1id string, vm2id string, conn types.Conn) error {
	return ConnectVLAN(vm1id, "", vm2id, "", conn, "")
}
