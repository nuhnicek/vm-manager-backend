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

	targetNode := ""
	if len(targetNodes) > 0 && targetNodes[0] != "" {
		targetNode = targetNodes[0]
	}

	if targetNode == "" && conn.NodePool != nil {
		_, _, selectedNodeName, selectErr := conn.NodePool.SelectNode(ctx, types.NodePlacementRequest{
			RequiredTemplateID: templateID,
		})
		if selectErr == nil && selectedNodeName != "" {
			targetNode = selectedNodeName
		}
	}

	var templateVM *proxmox.VirtualMachine
	var nodeClient *types.ProxmoxNodeClient
	var err error

	if targetNode != "" {
		templateVM, nodeClient, err = conn.FindTemplate(ctx, templateID, targetNode)
		if err != nil {
			return fmt.Errorf("šablona %d nebyla nalezena na uzlu '%s': %w", templateID, targetNode, err), "", 0
		}
	} else {
		templateVM, nodeClient, err = conn.FindTemplate(ctx, templateID)
		if err != nil {
			return fmt.Errorf("šablona %d nebyla nalezena v poolu: %w", templateID, err), "", 0
		}
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
	tagValue := "vm-" + vmUuid + ";" + "owner-" + ownerUUID
	cfgOpts := []proxmox.VirtualMachineOption{
		{
			Name:  "tags",
			Value: tagValue,
		},
		{
			Name:  "serial0",
			Value: "socket",
		},
		{
			Name:  "vga",
			Value: "std,clipboard=vnc",
		},
	}

	if IsVulnerableTarget(vmName, templateVM.Tags) {
		cfgOpts[0].Value = tagValue + ";vulnerable;cybersec;ctf"
		cfgOpts = append(cfgOpts, proxmox.VirtualMachineOption{
			Name:  "net0",
			Value: "model=virtio,bridge=vmbr0,tag=4094",
		})
	}

	err = newVM.ConfigSync(ctx, cfgOpts...)

	if err != nil {
		return err, "", 0
	}

	return nil, vmUuid, int(newVmID)
}
