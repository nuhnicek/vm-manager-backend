package vms

import (
	"context"
	"fmt"
	"time"
	"vm-manager-backend/types"

	"github.com/google/uuid"
	"github.com/luthermonson/go-proxmox"
)

func CreateFromTemplate(conn types.Conn, ownerUUID string, vmName string, templateID int, targetNodes ...string) (error, string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	templateVM, nodeClient, err := conn.FindTemplate(ctx, templateID, targetNodes...)
	if err != nil {
		return err, "", 0
	}
	if !templateVM.Template {
		return fmt.Errorf("VM %d is not a template", templateID), "", 0
	}

	node, err := nodeClient.Client.Node(ctx, nodeClient.NodeName)
	if err != nil {
		return err, "", 0
	}

	vmId, err := conn.NextPoolVMID(ctx)
	if err != nil {
		return err, "", 0
	}

	vmUuid := uuid.New().String()
	options := &proxmox.VirtualMachineCloneOptions{
		NewID: vmId,
		Name:  vmName,
		Full:  false,
	}

	newVmID, task, err := templateVM.Clone(ctx, options)
	if err != nil {
		return err, "", 0
	}

	if err := task.WaitFor(ctx, 3); err != nil {
		return err, "", 0
	}
	newVM, err := node.VirtualMachine(ctx, newVmID)
	if err != nil {
		return err, "", 0
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
		return err, "", 0
	}

	return nil, vmUuid, int(newVmID)
}
