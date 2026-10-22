package vms

import (
	"context"
	"fmt"
	"time"
	"vm-manager-backend/types"

	"github.com/google/uuid"

	"github.com/luthermonson/go-proxmox"
)

func CreateFromTemplate(conn types.Conn, ownerUUID string, vmName string, templateID int) (error, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var node *proxmox.Node
	var templateVM *proxmox.VirtualMachine
	var err error

	if conn.NodePool != nil {
		_, node, _, err = conn.NodePool.SelectNode(ctx, types.NodePlacementRequest{
			RequiredRAMMB:      2048,
			RequiredTemplateID: templateID,
		})
		if err == nil && node != nil {
			templateVM, _ = node.VirtualMachine(ctx, templateID)
		}
	}

	if templateVM == nil {
		node, templateVM, err = FindVMNode(ctx, conn.Proxmox, templateID)
		if err != nil {
			return fmt.Errorf("failed to locate template VM %d: %w", templateID, err), ""
		}
	}
	if !templateVM.Template {
		return fmt.Errorf("VM %d is not a template", templateID), ""
	}

	var vmId int
	if conn.NodePool != nil {
		vmId, err = conn.NodePool.AllocateVMID(ctx)
	} else {
		cluster, cErr := conn.Proxmox.Cluster(ctx)
		if cErr == nil && cluster != nil {
			vmId, err = cluster.NextID(ctx)
		}
	}
	if err != nil || vmId <= 0 {
		return fmt.Errorf("failed to allocate global VMID: %w", err), ""
	}
	vmUuid := uuid.New().String()
	options := &proxmox.VirtualMachineCloneOptions{
		NewID: vmId,
		Name:  vmName,
		Full:  false,
	}

	newVmID, task, err := templateVM.Clone(ctx, options)
	if err != nil {
		return err, ""
	}

	if err := task.WaitFor(ctx, 3); err != nil {
		return err, ""
	}
	newVM, err := node.VirtualMachine(ctx, newVmID)
	if err != nil {
		return err, ""
	}
	err = newVM.ConfigSync(ctx,
		proxmox.VirtualMachineOption{
			Name:  "tags",
			Value: "vm-" + vmUuid + ";" + "owner-" + ownerUUID,
		},
		proxmox.VirtualMachineOption{
			Name:  "serial0",
			Value: "socket",
		},
		proxmox.VirtualMachineOption{
			Name:  "ciuser",
			Value: "ubuntu",
		},
		proxmox.VirtualMachineOption{
			Name:  "cipassword",
			Value: "ubuntu",
		},
	)

	if err != nil {
		return err, ""
	}

	return nil, vmUuid
}
