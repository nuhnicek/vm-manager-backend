package balancer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// PVENodeInstance wraps a standalone Proxmox client and its runtime metrics.
type PVENodeInstance struct {
	Config    types.StandaloneNode
	Client    *proxmox.Client
	NodeName  string // Resolved PVE host name (e.g., "pve1")
	Telemetry types.NodeTelemetry
}

// PVENodePool manages connections and load telemetry across multiple independent PVE nodes.
type PVENodePool struct {
	mu     sync.RWMutex
	nodes  map[string]*PVENodeInstance // keyed by StandaloneNode.ID
	mongo  *mongo.Client
	stopCh chan struct{}
}

// NewPVENodePool initializes the multi-node manager and loads persisted nodes from MongoDB.
func NewPVENodePool(mongoClient *mongo.Client, initialNode types.StandaloneNode) *PVENodePool {
	pool := &PVENodePool{
		nodes:  make(map[string]*PVENodeInstance),
		mongo:  mongoClient,
		stopCh: make(chan struct{}),
	}

	// Register the primary node configured in .env
	if initialNode.HostURI != "" {
		_ = pool.registerNodeInternal(context.Background(), initialNode, false)
	}

	// Load any additional standalone nodes saved in MongoDB
	if mongoClient != nil {
		coll := mongoClient.Database("lab411").Collection("pve_nodes")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cur, err := coll.Find(ctx, bson.M{})
		if err == nil {
			var savedNodes []types.StandaloneNode
			if err := cur.All(ctx, &savedNodes); err == nil {
				for _, n := range savedNodes {
					if n.ID != initialNode.ID && n.HostURI != "" {
						_ = pool.registerNodeInternal(ctx, n, false)
					}
				}
			}
		}
	}

	// Initial synchronous telemetry refresh
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	pool.RefreshTelemetry(ctx)
	cancel()

	// Start background polling worker
	go pool.telemetryWorker()

	return pool
}

// telemetryWorker periodically updates load and health metrics from all nodes every 20 seconds.
func (p *PVENodePool) telemetryWorker() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			p.RefreshTelemetry(ctx)
			cancel()
		case <-p.stopCh:
			return
		}
	}
}

// Close terminates the background telemetry worker.
func (p *PVENodePool) Close() {
	close(p.stopCh)
}

func (p *PVENodePool) registerNodeInternal(ctx context.Context, node types.StandaloneNode, persist bool) error {
	if node.ID == "" {
		node.ID = fmt.Sprintf("pve-node-%d", time.Now().UnixNano()%100000)
	}
	if node.Weight <= 0 {
		node.Weight = 100
	}
	node.IsActive = true

	creds := proxmox.Credentials{
		Username: node.Username,
		Password: node.Password,
	}
	client := proxmox.NewClient(node.HostURI,
		proxmox.WithCredentials(&creds),
		proxmox.WithInsecureSkipVerify(),
	)

	// Discover resolved node name
	nodeName := node.Name
	if nodeName == "" {
		nodeStatuses, err := client.Nodes(ctx)
		if err == nil && len(nodeStatuses) > 0 {
			nodeName = nodeStatuses[0].Node
		} else {
			nodeName = "pve" // default fallback
		}
	}
	node.Name = nodeName

	inst := &PVENodeInstance{
		Config:   node,
		Client:   client,
		NodeName: nodeName,
		Telemetry: types.NodeTelemetry{
			NodeID:      node.ID,
			NodeName:    nodeName,
			HostURI:     node.HostURI,
			IsOnline:    false,
			LastUpdated: time.Now(),
		},
	}

	p.mu.Lock()
	p.nodes[node.ID] = inst
	p.mu.Unlock()

	if persist && p.mongo != nil {
		coll := p.mongo.Database("lab411").Collection("pve_nodes")
		node.UpdatedAt = time.Now()
		if node.CreatedAt.IsZero() {
			node.CreatedAt = time.Now()
		}
		_, err := coll.ReplaceOne(ctx, bson.M{"_id": node.ID}, node, nil)
		if err != nil {
			return fmt.Errorf("failed to save node in mongo: %w", err)
		}
	}

	return nil
}

// RegisterNode adds a new standalone Proxmox node to the pool and persists it in MongoDB.
func (p *PVENodePool) RegisterNode(ctx context.Context, node types.StandaloneNode) error {
	return p.registerNodeInternal(ctx, node, true)
}

// RemoveNode unregisters a node from the pool and deletes it from MongoDB.
func (p *PVENodePool) RemoveNode(ctx context.Context, nodeID string) error {
	p.mu.Lock()
	delete(p.nodes, nodeID)
	p.mu.Unlock()

	if p.mongo != nil {
		coll := p.mongo.Database("lab411").Collection("pve_nodes")
		_, err := coll.DeleteOne(ctx, bson.M{"_id": nodeID})
		return err
	}
	return nil
}

// RefreshTelemetry queries all nodes in parallel and updates their resource usage and templates.
func (p *PVENodePool) RefreshTelemetry(ctx context.Context) {
	p.mu.RLock()
	instances := make([]*PVENodeInstance, 0, len(p.nodes))
	for _, inst := range p.nodes {
		instances = append(instances, inst)
	}
	p.mu.RUnlock()

	var wg sync.WaitGroup
	for _, inst := range instances {
		wg.Add(1)
		go func(instance *PVENodeInstance) {
			defer wg.Done()
			p.pollSingleNodeTelemetry(ctx, instance)
		}(inst)
	}
	wg.Wait()
}

func (p *PVENodePool) pollSingleNodeTelemetry(ctx context.Context, inst *PVENodeInstance) {
	nodeObj, err := inst.Client.Node(ctx, inst.NodeName)
	if err != nil {
		// Try discovering nodes again if name changed
		if ns, nErr := inst.Client.Nodes(ctx); nErr == nil && len(ns) > 0 {
			inst.NodeName = ns[0].Node
			nodeObj, err = inst.Client.Node(ctx, inst.NodeName)
		}
	}

	if err != nil || nodeObj == nil {
		p.mu.Lock()
		inst.Telemetry.IsOnline = false
		inst.Telemetry.ErrorMessage = fmt.Sprintf("unreachable: %v", err)
		inst.Telemetry.Weight = inst.Config.Weight
		inst.Telemetry.IsActive = inst.Config.IsActive
		inst.Telemetry.LastUpdated = time.Now()
		p.mu.Unlock()
		return
	}

	// 1. Fetch node memory and CPU status
	totalRAMMB := 0
	freeRAMMB := 0
	usedRAMMB := 0
	ramUsagePct := 0.0
	cpuUsagePct := 0.0

	if err := nodeObj.Status(ctx); err == nil {
		if nodeObj.Memory.Total > 0 {
			totalRAMMB = int(nodeObj.Memory.Total / (1024 * 1024))
			usedRAMMB = int(nodeObj.Memory.Used / (1024 * 1024))
			freeRAMMB = int(nodeObj.Memory.Free / (1024 * 1024))
			if totalRAMMB > 0 {
				ramUsagePct = (float64(usedRAMMB) / float64(totalRAMMB)) * 100.0
			}
		}
		cpuUsagePct = nodeObj.CPU * 100.0
	}

	// 2. Fetch virtual machines to count load and identify available templates
	vms, err := nodeObj.VirtualMachines(ctx)
	activeCount := 0
	totalCount := 0
	var templateIDs []int

	if err == nil {
		totalCount = len(vms)
		for _, v := range vms {
			if v.Template {
				templateIDs = append(templateIDs, int(v.VMID))
			} else if strings.EqualFold(v.Status, "running") {
				activeCount++
			}
		}
	}

	p.mu.Lock()
	inst.Telemetry = types.NodeTelemetry{
		NodeID:        inst.Config.ID,
		NodeName:      inst.NodeName,
		HostURI:       inst.Config.HostURI,
		IsOnline:      true,
		TotalRAMMB:    totalRAMMB,
		UsedRAMMB:     usedRAMMB,
		FreeRAMMB:     freeRAMMB,
		RAMUsagePct:   ramUsagePct,
		CPUUsagePct:   cpuUsagePct,
		ActiveVMCount: activeCount,
		TotalVMCount:  totalCount,
		Templates:     templateIDs,
		Weight:        inst.Config.Weight,
		IsActive:      inst.Config.IsActive,
		LastUpdated:   time.Now(),
	}
	p.mu.Unlock()
}

// GetTelemetry returns the current snapshot of telemetry across all registered standalone nodes.
func (p *PVENodePool) GetTelemetry(ctx context.Context) []types.NodeTelemetry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]types.NodeTelemetry, 0, len(p.nodes))
	for _, inst := range p.nodes {
		result = append(result, inst.Telemetry)
	}
	return result
}

// GetClientForNode returns the Proxmox client and node object for a specific node ID or name.
func (p *PVENodePool) GetClientForNode(nodeID string) (*proxmox.Client, *proxmox.Node, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, inst := range p.nodes {
		if inst.Config.ID == nodeID || inst.NodeName == nodeID {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			node, err := inst.Client.Node(ctx, inst.NodeName)
			return inst.Client, node, err
		}
	}
	return nil, nil, fmt.Errorf("node %s not found in pool", nodeID)
}

// FindVM searches across all standalone nodes in parallel for the given VMID.
func (p *PVENodePool) FindVM(ctx context.Context, vmid int) (*proxmox.Client, *proxmox.Node, *proxmox.VirtualMachine, error) {
	p.mu.RLock()
	instances := make([]*PVENodeInstance, 0, len(p.nodes))
	for _, inst := range p.nodes {
		instances = append(instances, inst)
	}
	p.mu.RUnlock()

	type vmResult struct {
		client *proxmox.Client
		node   *proxmox.Node
		vm     *proxmox.VirtualMachine
	}

	resultCh := make(chan vmResult, len(instances))
	var wg sync.WaitGroup

	for _, inst := range instances {
		wg.Add(1)
		go func(instance *PVENodeInstance) {
			defer wg.Done()
			nodeObj, err := instance.Client.Node(ctx, instance.NodeName)
			if err != nil || nodeObj == nil {
				return
			}
			vm, err := nodeObj.VirtualMachine(ctx, vmid)
			if err == nil && vm != nil {
				select {
				case resultCh <- vmResult{client: instance.Client, node: nodeObj, vm: vm}:
				default:
				}
			}
		}(inst)
	}

	wg.Wait()
	close(resultCh)

	if res, ok := <-resultCh; ok {
		return res.client, res.node, res.vm, nil
	}

	return nil, nil, nil, fmt.Errorf("VM %d not found on any standalone Proxmox node", vmid)
}

// AllocateVMID delegates to the atomic MongoDB global VMID counter.
func (p *PVENodePool) AllocateVMID(ctx context.Context) (int, error) {
	return AllocateGlobalVMID(ctx, p.mongo)
}

// ListAllTemplates aggregates all templates from all standalone nodes into a unified list.
func (p *PVENodePool) ListAllTemplates(ctx context.Context) ([]types.Vm, error) {
	p.mu.RLock()
	instances := make([]*PVENodeInstance, 0, len(p.nodes))
	for _, inst := range p.nodes {
		instances = append(instances, inst)
	}
	p.mu.RUnlock()

	var allTemplates []types.Vm
	seen := make(map[string]bool)

	for _, inst := range instances {
		nodeObj, err := inst.Client.Node(ctx, inst.NodeName)
		if err != nil || nodeObj == nil {
			continue
		}
		vms, err := nodeObj.VirtualMachines(ctx)
		if err != nil {
			continue
		}

		for _, v := range vms {
			if v.Template {
				key := fmt.Sprintf("%s-%d", inst.NodeName, v.VMID)
				if !seen[key] {
					seen[key] = true
					vmUUID := fmt.Sprintf("tpl-%d", v.VMID)
					for _, tag := range strings.Split(v.Tags, ";") {
						trimmed := strings.TrimSpace(tag)
						if strings.HasPrefix(trimmed, "vm-") {
							vmUUID = strings.TrimPrefix(trimmed, "vm-")
							break
						}
					}

					allTemplates = append(allTemplates, types.Vm{
						Id:   v.VMID,
						Name: v.Name,
						UUID: vmUUID,
						Node: inst.NodeName,
					})
				}
			}
		}
	}

	if allTemplates == nil {
		allTemplates = make([]types.Vm, 0)
	}
	return allTemplates, nil
}

// GetNodes returns active Proxmox Node objects across all registered standalone servers.
func (p *PVENodePool) GetNodes(ctx context.Context) []*proxmox.Node {
	p.mu.RLock()
	instances := make([]*PVENodeInstance, 0, len(p.nodes))
	for _, inst := range p.nodes {
		instances = append(instances, inst)
	}
	p.mu.RUnlock()

	var nodes []*proxmox.Node
	for _, inst := range instances {
		if !inst.Config.IsActive {
			continue
		}
		nodeObj, err := inst.Client.Node(ctx, inst.NodeName)
		if err == nil && nodeObj != nil {
			nodes = append(nodes, nodeObj)
		}
	}
	return nodes
}

