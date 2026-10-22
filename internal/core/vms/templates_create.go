package vms

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
)

func ConvertVMToTemplate(conn types.Conn, userUUID string, vmid int) (*types.Vm, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var nodeObj *proxmox.Node
	var vm *proxmox.VirtualMachine
	var err error

	if conn.NodePool != nil {
		_, nodeObj, vm, err = conn.NodePool.FindVM(ctx, vmid)
	} else {
		nodeObj, vm, err = FindVMNode(ctx, conn.Proxmox, vmid)
	}

	if err != nil || vm == nil {
		return nil, fmt.Errorf("VM %d not found on any Proxmox node: %w", vmid, err)
	}

	if vm.Template {
		return nil, fmt.Errorf("VM %d is already a template", vmid)
	}

	if !VerifyVMOwner(vm, userUUID) {
		return nil, fmt.Errorf("unauthorized: VM %d does not belong to user", vmid)
	}

	if strings.EqualFold(vm.Status, "running") {
		stopTask, err := vm.Stop(ctx)
		if err == nil && stopTask != nil {
			_ = stopTask.WaitFor(ctx, 10)
		}
	}

	convTask, err := vm.ConvertToTemplate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to convert VM %d to template: %w", vmid, err)
	}
	if convTask != nil {
		_ = convTask.WaitFor(ctx, 15)
	}

	newTag := "template;tpl-" + strconv.Itoa(vmid)
	if vm.Tags != "" {
		newTag = vm.Tags + ";" + newTag
	}
	_ = vm.ConfigSync(ctx, proxmox.VirtualMachineOption{
		Name:  "tags",
		Value: newTag,
	})

	if conn.NodePool != nil {
		conn.NodePool.RefreshTelemetry(ctx)
	}

	nodeName := "pve"
	if nodeObj != nil {
		nodeName = nodeObj.Name
	}

	return &types.Vm{
		Id:     proxmox.StringOrUint64(vmid),
		Name:   vm.Name,
		UUID:   fmt.Sprintf("tpl-%d", vmid),
		Node:   nodeName,
		Status: "stopped",
	}, nil
}

func CloneVMAsTemplate(conn types.Conn, userUUID string, req types.CreateTemplateRequest) (*types.Vm, error) {
	if req.SourceVMID <= 0 {
		return nil, fmt.Errorf("source VMID is required")
	}
	if req.TemplateName == "" {
		req.TemplateName = fmt.Sprintf("template-%d", req.SourceVMID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var nodeObj *proxmox.Node
	var sourceVM *proxmox.VirtualMachine
	var err error

	if conn.NodePool != nil {
		_, nodeObj, sourceVM, err = conn.NodePool.FindVM(ctx, req.SourceVMID)
	} else {
		nodeObj, sourceVM, err = FindVMNode(ctx, conn.Proxmox, req.SourceVMID)
	}

	if err != nil || sourceVM == nil {
		return nil, fmt.Errorf("source VM %d not found: %w", req.SourceVMID, err)
	}

	if !VerifyVMOwner(sourceVM, userUUID) {
		return nil, fmt.Errorf("unauthorized: source VM %d does not belong to user", req.SourceVMID)
	}

	var newVMID int
	if conn.NodePool != nil {
		newVMID, err = conn.NodePool.AllocateVMID(ctx)
	} else {
		cluster, cErr := conn.Proxmox.Cluster(ctx)
		if cErr == nil && cluster != nil {
			newVMID, err = cluster.NextID(ctx)
		} else {
			newVMID = req.SourceVMID + 100
		}
	}
	if err != nil || newVMID <= 0 {
		return nil, fmt.Errorf("failed to allocate new VMID for template: %w", err)
	}

	cloneOpts := &proxmox.VirtualMachineCloneOptions{
		NewID: newVMID,
		Name:  req.TemplateName,
		Full:  true,
	}

	_, cloneTask, err := sourceVM.Clone(ctx, cloneOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to clone source VM %d for template: %w", req.SourceVMID, err)
	}
	if cloneTask != nil {
		if err := cloneTask.WaitFor(ctx, 60); err != nil {
			return nil, fmt.Errorf("clone task failed: %w", err)
		}
	}

	clonedVM, err := nodeObj.VirtualMachine(ctx, newVMID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cloned VM %d: %w", newVMID, err)
	}

	var syncOpts []proxmox.VirtualMachineOption
	syncOpts = append(syncOpts, proxmox.VirtualMachineOption{
		Name:  "tags",
		Value: fmt.Sprintf("template;tpl-%d", newVMID),
	})
	if req.RAMMB > 0 {
		syncOpts = append(syncOpts, proxmox.VirtualMachineOption{
			Name:  "memory",
			Value: strconv.Itoa(req.RAMMB),
		})
	}
	if req.Description != "" {
		syncOpts = append(syncOpts, proxmox.VirtualMachineOption{
			Name:  "description",
			Value: req.Description,
		})
	}
	_ = clonedVM.ConfigSync(ctx, syncOpts...)

	convTask, err := clonedVM.ConvertToTemplate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to convert cloned VM %d to template: %w", newVMID, err)
	}
	if convTask != nil {
		_ = convTask.WaitFor(ctx, 15)
	}

	if conn.NodePool != nil {
		conn.NodePool.RefreshTelemetry(ctx)
	}

	return &types.Vm{
		Id:     proxmox.StringOrUint64(newVMID),
		Name:   req.TemplateName,
		UUID:   fmt.Sprintf("tpl-%d", newVMID),
		Node:   nodeObj.Name,
		Status: "stopped",
	}, nil
}
