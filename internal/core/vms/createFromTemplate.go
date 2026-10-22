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

	templateVM, nodeClient, err := conn.FindTemplate(ctx, templateID)
	if err != nil {
		return err, ""
	}
	if !templateVM.Template {
		return fmt.Errorf("VM %d is not a template", templateID), ""
	}

	node, err := nodeClient.Client.Node(ctx, nodeClient.NodeName)
	if err != nil {
		return err, ""
	}

	vmId, err := conn.NextPoolVMID(ctx)
	if err != nil {
		return err, ""
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
	)

	if err != nil {
		return err, ""
	}

	return nil, vmUuid
}
