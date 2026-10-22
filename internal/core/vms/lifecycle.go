package vms

import (
	"context"
	"fmt"
	"strings"
	"time"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
)

func VerifyVMOwner(vm *proxmox.VirtualMachine, userUUID string, roles ...string) bool {
	for _, role := range roles {
		if role == "teacher" || role == "admin" {
			return true
		}
	}
	tags := strings.Split(vm.Tags, ";")
	for _, tag := range tags {
		if strings.TrimSpace(tag) == "owner-"+userUUID {
			return true
		}
	}
	return false
}

func StartVM(conn types.Conn, userUUID string, role string, vmid int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vm, _, err := conn.FindVM(ctx, vmid)
	if err != nil {
		return fmt.Errorf("failed to get VM %d: %w", vmid, err)
	}

	if vm.Template {
		return fmt.Errorf("cannot start global template %d", vmid)
	}

	if !VerifyVMOwner(vm, userUUID, role) {
		return fmt.Errorf("unauthorized: VM %d does not belong to user", vmid)
	}

	task, err := vm.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start VM %d: %w", vmid, err)
	}

	if task != nil {
		_ = task.WaitFor(ctx, 3)
	}
	return nil
}

func StopVM(conn types.Conn, userUUID string, role string, vmid int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vm, _, err := conn.FindVM(ctx, vmid)
	if err != nil {
		return fmt.Errorf("failed to get VM %d: %w", vmid, err)
	}

	if vm.Template {
		return fmt.Errorf("cannot stop global template %d", vmid)
	}

	if !VerifyVMOwner(vm, userUUID, role) {
		return fmt.Errorf("unauthorized: VM %d does not belong to user", vmid)
	}

	task, err := vm.Stop(ctx)
	if err != nil {
		return fmt.Errorf("failed to stop VM %d: %w", vmid, err)
	}

	if task != nil {
		_ = task.WaitFor(ctx, 3)
	}
	return nil
}

func DeleteVM(conn types.Conn, userUUID string, role string, vmid int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	vm, _, err := conn.FindVM(ctx, vmid)
	if err != nil {
		return fmt.Errorf("failed to get VM %d: %w", vmid, err)
	}

	if vm.Template {
		return fmt.Errorf("cannot delete global template %d", vmid)
	}

	if !VerifyVMOwner(vm, userUUID, role) {
		return fmt.Errorf("unauthorized: VM %d does not belong to user", vmid)
	}

	if strings.EqualFold(vm.Status, "running") {
		task, err := vm.Stop(ctx)
		if err == nil && task != nil {
			_ = task.WaitFor(ctx, 5)
		}
	}

	task, err := vm.Delete(ctx, &proxmox.VirtualMachineDeleteOptions{
		Purge:                    proxmox.IntOrBool(true),
		DestroyUnreferencedDisks: proxmox.IntOrBool(true),
	})
	if err != nil {
		return fmt.Errorf("failed to delete VM %d: %w", vmid, err)
	}

	if task != nil {
		_ = task.WaitFor(ctx, 5)
	}
	return nil
}
