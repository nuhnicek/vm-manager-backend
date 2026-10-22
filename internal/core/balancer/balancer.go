package balancer

import (
	"context"
	"fmt"
	"sort"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
)

type scoredCandidate struct {
	instance *PVENodeInstance
	node     *proxmox.Node
	score    float64
}

func (p *PVENodePool) SelectNode(ctx context.Context, req types.NodePlacementRequest) (*proxmox.Client, *proxmox.Node, string, error) {
	p.mu.RLock()
	instances := make([]*PVENodeInstance, 0, len(p.nodes))
	for _, inst := range p.nodes {
		instances = append(instances, inst)
	}
	p.mu.RUnlock()

	if len(instances) == 0 {
		return nil, nil, "", fmt.Errorf("no Proxmox standalone nodes configured in pool")
	}

	if req.PreferNodeID != "" {
		for _, inst := range instances {
			if (inst.Config.ID == req.PreferNodeID || inst.NodeName == req.PreferNodeID) && inst.Config.IsActive {
				nodeObj, err := inst.Client.Node(ctx, inst.NodeName)
				if err == nil && nodeObj != nil {
					return inst.Client, nodeObj, inst.NodeName, nil
				}
			}
		}
	}

	var candidates []scoredCandidate
	var missingTemplateNodes []string
	var lowMemoryNodes []string

	for _, inst := range instances {
		if !inst.Config.IsActive || !inst.Telemetry.IsOnline {
			continue
		}

		if req.RequiredTemplateID > 0 {
			hasTemplate := false
			for _, tplID := range inst.Telemetry.Templates {
				if tplID == req.RequiredTemplateID {
					hasTemplate = true
					break
				}
			}
			if !hasTemplate {
				missingTemplateNodes = append(missingTemplateNodes, inst.NodeName)
				continue
			}
		}

		effectiveFreeRAM := inst.Telemetry.FreeRAMMB - 256
		if req.RequiredRAMMB > 0 && effectiveFreeRAM < req.RequiredRAMMB {
			lowMemoryNodes = append(lowMemoryNodes, fmt.Sprintf("%s (free: %dMB, req: %dMB)", inst.NodeName, inst.Telemetry.FreeRAMMB, req.RequiredRAMMB))
			continue
		}

		if inst.Client == nil {
			continue
		}

		nodeObj, err := inst.Client.Node(ctx, inst.NodeName)
		if err != nil || nodeObj == nil {
			continue
		}

		weight := float64(inst.Config.Weight)
		if weight <= 0 {
			weight = 100.0
		}

		freeRAMScore := float64(inst.Telemetry.FreeRAMMB)
		cpuPenalty := inst.Telemetry.CPUUsagePct * 1000.0
		vmLoadPenalty := float64(inst.Telemetry.ActiveVMCount * 256)
		weightBoost := weight * 10.0

		totalScore := freeRAMScore - cpuPenalty - vmLoadPenalty + weightBoost

		candidates = append(candidates, scoredCandidate{
			instance: inst,
			node:     nodeObj,
			score:    totalScore,
		})
	}

	if len(candidates) == 0 {
		if len(missingTemplateNodes) > 0 && len(lowMemoryNodes) == 0 {
			return nil, nil, "", fmt.Errorf("requested template ID %d is not present on available nodes: %v", req.RequiredTemplateID, missingTemplateNodes)
		}
		if len(lowMemoryNodes) > 0 {
			return nil, nil, "", fmt.Errorf("insufficient RAM on online nodes: %v", lowMemoryNodes)
		}
		return nil, nil, "", fmt.Errorf("no healthy standalone Proxmox nodes available matching placement criteria")
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	best := candidates[0]
	return best.instance.Client, best.node, best.instance.NodeName, nil
}
