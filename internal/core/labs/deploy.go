package labs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/internal/core/vms/vlan"
	"vm-manager-backend/types"

	"github.com/google/uuid"
	"github.com/luthermonson/go-proxmox"
)

func DeployLab(conn types.Conn, templateID string, targetUserUUID string, targetUserEmail string, customPrefix string) (*types.LabInstance, error) {
	tpl, err := GetLabTemplate(conn, templateID)
	if err != nil {
		return nil, err
	}

	labInstanceID := "lab-" + uuid.New().String()[:8]
	prefix := customPrefix
	if prefix == "" {
		prefix = strings.ReplaceAll(strings.ToLower(tpl.Name), " ", "-")
		if len(prefix) > 12 {
			prefix = prefix[:12]
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	totalLabRAM := 0
	primaryTemplateID := 0
	for _, vmTpl := range tpl.VMs {
		totalLabRAM += vmTpl.RAM
		if primaryTemplateID == 0 && vmTpl.TemplateID > 0 {
			primaryTemplateID = vmTpl.TemplateID
		}
	}

	var preferredNode *proxmox.Node
	if conn.NodePool != nil {
		_, preferredNode, _, _ = conn.NodePool.SelectNode(ctx, types.NodePlacementRequest{
			RequiredRAMMB:      totalLabRAM,
			RequiredTemplateID: primaryTemplateID,
		})
	}

	vmMap := make(map[string]int)
	var instanceVMs []types.LabInstanceVM

	for _, vmTpl := range tpl.VMs {
		targetTemplateID := vmTpl.TemplateID
		var node *proxmox.Node
		var baseVM *proxmox.VirtualMachine

		if vmTpl.Node != "" {
			node, baseVM, err = vms.FindVMNode(ctx, conn.Proxmox, targetTemplateID, vmTpl.Node)
		} else if preferredNode != nil {
			if vmObj, err := preferredNode.VirtualMachine(ctx, targetTemplateID); err == nil && vmObj != nil && vmObj.Template {
				node = preferredNode
				baseVM = vmObj
			}
		}

		if baseVM == nil {
			node, baseVM, err = vms.FindVMNode(ctx, conn.Proxmox, targetTemplateID)
			if err != nil || baseVM == nil || !baseVM.Template {
				availableTpls, listErr := vms.ListTemplates(conn)
				if listErr == nil && len(availableTpls) > 0 {
					fallbackID := int(availableTpls[0].Id)
					fbNode, fbVM, fbErr := vms.FindVMNode(ctx, conn.Proxmox, fallbackID)
					if fbErr == nil && fbVM != nil && fbVM.Template {
						node = fbNode
						baseVM = fbVM
						targetTemplateID = fallbackID
					}
				}
			}
		}

		if baseVM == nil || !baseVM.Template {
			return nil, fmt.Errorf("no valid Proxmox VM template available to clone node '%s' (specified ID %d)", vmTpl.Name, vmTpl.TemplateID)
		}

		var nextID int
		if conn.NodePool != nil {
			nextID, err = conn.NodePool.AllocateVMID(ctx)
		} else {
			cluster, cErr := conn.Proxmox.Cluster(ctx)
			if cErr == nil && cluster != nil {
				nextID, err = cluster.NextID(ctx)
			}
		}
		if err != nil || nextID <= 0 {
			return nil, fmt.Errorf("failed to allocate global VMID: %w", err)
		}

		vmName := fmt.Sprintf("%s-%s-%s", prefix, strings.ToLower(vmTpl.Name), labInstanceID[4:])
		vmUUID := uuid.New().String()

		options := &proxmox.VirtualMachineCloneOptions{
			NewID: nextID,
			Name:  vmName,
			Full:  false,
		}

		newVmID, task, err := baseVM.Clone(ctx, options)
		if err != nil {
			return nil, fmt.Errorf("failed to clone VM for %s from template %d on node %s: %w", vmTpl.Name, targetTemplateID, node.Name, err)
		}

		if err := task.WaitFor(ctx, 3); err != nil {
			return nil, fmt.Errorf("clone task failed for %s: %w", vmTpl.Name, err)
		}

		clonedVM, err := node.VirtualMachine(ctx, newVmID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve cloned VM %d on node %s: %w", newVmID, node.Name, err)
		}

		defaultUser := vmTpl.DefaultUser
		if defaultUser == "" {
			defaultUser = "ubuntu"
		}
		defaultPass := vmTpl.DefaultPass
		if defaultPass == "" {
			defaultPass = "ubuntu"
		}

		tagValue := fmt.Sprintf("vm-%s;owner-%s;lab-%s;tplvm-%s", vmUUID, targetUserUUID, labInstanceID, vmTpl.Name)
		syncOptions := []proxmox.VirtualMachineOption{
			{Name: "tags", Value: tagValue},
			{Name: "serial0", Value: "socket"},
			{Name: "ciuser", Value: defaultUser},
			{Name: "cipassword", Value: defaultPass},
		}

		if vmTpl.RAM > 0 {
			syncOptions = append(syncOptions, proxmox.VirtualMachineOption{
				Name:  "memory",
				Value: strconv.Itoa(vmTpl.RAM),
			})
		}

		err = clonedVM.ConfigSync(ctx, syncOptions...)
		if err != nil {
			return nil, fmt.Errorf("failed to configure cloned VM %d: %w", newVmID, err)
		}

		vmMap[vmTpl.Name] = int(newVmID)
		instanceVMs = append(instanceVMs, types.LabInstanceVM{
			TemplateVMName: vmTpl.Name,
			VMID:           int(newVmID),
			Name:           vmName,
			DefaultUser:    defaultUser,
			DefaultPass:    defaultPass,
		})
	}

	for _, link := range tpl.Networks {
		vm1ID, ok1 := vmMap[link.FromVM]
		vm2ID, ok2 := vmMap[link.ToVM]
		if !ok1 || !ok2 {
			continue
		}

		err = vlan.ConnectVLAN(
			strconv.Itoa(vm1ID),
			link.FromNIC,
			strconv.Itoa(vm2ID),
			link.ToNIC,
			conn,
			targetUserUUID,
		)
		if err != nil {
			fmt.Printf("warning: failed to wire link between %s and %s: %v\n", link.FromVM, link.ToVM, err)
		}
	}

	instance := types.LabInstance{
		ID:         labInstanceID,
		TemplateID: tpl.ID,
		LabName:    tpl.Name,
		OwnerUUID:  targetUserUUID,
		OwnerEmail: targetUserEmail,
		Status:     "ready",
		VMs:        instanceVMs,
		CreatedAt:  time.Now(),
	}

	coll := conn.Mongo.Database("lab411").Collection("lab_instances")
	_, err = coll.InsertOne(context.TODO(), instance)
	if err != nil {
		return nil, fmt.Errorf("failed to record lab instance in mongo: %w", err)
	}

	return &instance, nil
}
