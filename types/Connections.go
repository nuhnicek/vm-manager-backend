package types

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/luthermonson/go-proxmox"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ProxmoxNodeClient struct {
	Client   *proxmox.Client
	NodeName string
	HostURI  string
}

type Conn struct {
	Mongo       *mongo.Client
	Redis       *redis.Client
	Proxmox     *proxmox.Client
	ProxmoxNode string
	Pool        []ProxmoxNodeClient
}

func (c Conn) GetPool() []ProxmoxNodeClient {
	if len(c.Pool) > 0 {
		return c.Pool
	}
	if c.Proxmox != nil {
		nodeName := c.ProxmoxNode
		if nodeName == "" {
			nodeName = os.Getenv("PROXMOX_NODE")
		}
		if nodeName == "" {
			nodes, err := c.Proxmox.Nodes(context.Background())
			if err == nil && len(nodes) > 0 {
				nodeName = nodes[0].Node
			} else {
				nodeName = "pve"
			}
		}
		return []ProxmoxNodeClient{
			{
				Client:   c.Proxmox,
				NodeName: nodeName,
			},
		}
	}
	return nil
}

func (c Conn) GetNodesInfo(ctx context.Context) []ProxmoxNodeInfo {
	pool := c.GetPool()
	var res []ProxmoxNodeInfo
	for _, nc := range pool {
		info := ProxmoxNodeInfo{
			Name:    nc.NodeName,
			HostURI: nc.HostURI,
			Status:  "offline",
		}
		version, err := nc.Client.Version(ctx)
		if err == nil && version != nil {
			info.Status = "online"
			info.Version = version.Version
		}
		node, err := nc.Client.Node(ctx, nc.NodeName)
		if err == nil && node != nil {
			vms, err := node.VirtualMachines(ctx)
			if err == nil {
				info.TotalVMs = len(vms)
				for _, vm := range vms {
					if vm.Template {
						info.Templates++
					} else if strings.EqualFold(vm.Status, "running") {
						info.RunningVMs++
					}
				}
			}
		}
		res = append(res, info)
	}
	return res
}

func (c Conn) GetNode(ctx context.Context) (*proxmox.Node, error) {
	nodeName := c.ProxmoxNode
	if nodeName == "" {
		nodeName = os.Getenv("PROXMOX_NODE")
	}
	if nodeName != "" && c.Proxmox != nil {
		return c.Proxmox.Node(ctx, nodeName)
	}
	if len(c.Pool) > 0 {
		return c.Pool[0].Client.Node(ctx, c.Pool[0].NodeName)
	}
	if c.Proxmox != nil {
		nodes, err := c.Proxmox.Nodes(ctx)
		if err == nil && len(nodes) > 0 {
			return c.Proxmox.Node(ctx, nodes[0].Node)
		}
		return c.Proxmox.Node(ctx, "pve")
	}
	return nil, fmt.Errorf("no Proxmox client configured")
}

func (c Conn) FindVM(ctx context.Context, vmid int) (*proxmox.VirtualMachine, *ProxmoxNodeClient, error) {
	pool := c.GetPool()
	var foundVM *proxmox.VirtualMachine
	var foundClient *ProxmoxNodeClient

	for i := range pool {
		nc := pool[i]
		node, err := nc.Client.Node(ctx, nc.NodeName)
		if err != nil {
			continue
		}
		vm, err := node.VirtualMachine(ctx, vmid)
		if err == nil && vm != nil {
			if !vm.Template {
				return vm, &nc, nil
			}
			if foundVM == nil {
				foundVM = vm
				foundClient = &nc
			}
		}
	}

	if foundVM != nil {
		return foundVM, foundClient, nil
	}
	return nil, nil, fmt.Errorf("VM %d not found in pool", vmid)
}

func (c Conn) FindTemplate(ctx context.Context, templateID int) (*proxmox.VirtualMachine, *ProxmoxNodeClient, error) {
	pool := c.GetPool()
	for i := range pool {
		nc := pool[i]
		node, err := nc.Client.Node(ctx, nc.NodeName)
		if err != nil {
			continue
		}
		vm, err := node.VirtualMachine(ctx, templateID)
		if err == nil && vm != nil && vm.Template {
			return vm, &nc, nil
		}
	}
	return nil, nil, fmt.Errorf("template %d not found in pool", templateID)
}

func (c Conn) NextPoolVMID(ctx context.Context) (int, error) {
	pool := c.GetPool()
	maxID := 199
	for _, nc := range pool {
		node, err := nc.Client.Node(ctx, nc.NodeName)
		if err != nil {
			continue
		}
		vms, err := node.VirtualMachines(ctx)
		if err != nil {
			continue
		}
		for _, vm := range vms {
			if int(vm.VMID) > maxID {
				maxID = int(vm.VMID)
			}
		}
	}
	return maxID + 1, nil
}
