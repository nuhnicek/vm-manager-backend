package balancer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
)

func TestNodeScoringAndPlacement(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": {}}`))
	}))
	defer ts.Close()

	client1 := proxmox.NewClient(ts.URL)
	client2 := proxmox.NewClient(ts.URL)

	pool := &PVENodePool{
		nodes: make(map[string]*PVENodeInstance),
	}

	pool.nodes["node-1"] = &PVENodeInstance{
		Config: types.StandaloneNode{
			ID:       "node-1",
			HostURI:  ts.URL,
			IsActive: true,
			Weight:   10,
		},
		Client:   client1,
		NodeName: "pve1",
		Telemetry: types.NodeTelemetry{
			NodeID:        "node-1",
			NodeName:      "pve1",
			IsOnline:      true,
			TotalRAMMB:    65536,
			FreeRAMMB:     40000,
			RAMUsagePct:   38.9,
			CPUUsagePct:   15.0,
			ActiveVMCount: 3,
			Templates:     []int{101, 200},
			LastUpdated:   time.Now(),
		},
	}

	pool.nodes["node-2"] = &PVENodeInstance{
		Config: types.StandaloneNode{
			ID:       "node-2",
			HostURI:  ts.URL,
			IsActive: true,
			Weight:   5,
		},
		Client:   client2,
		NodeName: "pve2",
		Telemetry: types.NodeTelemetry{
			NodeID:        "node-2",
			NodeName:      "pve2",
			IsOnline:      true,
			TotalRAMMB:    32768,
			FreeRAMMB:     4000,
			RAMUsagePct:   87.7,
			CPUUsagePct:   85.0,
			ActiveVMCount: 12,
			Templates:     []int{101},
			LastUpdated:   time.Now(),
		},
	}

	pool.nodes["node-3"] = &PVENodeInstance{
		Config: types.StandaloneNode{
			ID:       "node-3",
			HostURI:  "https://node3:8006",
			IsActive: true,
		},
		Client:   proxmox.NewClient("https://node3:8006"),
		NodeName: "pve3",
		Telemetry: types.NodeTelemetry{
			NodeID:      "node-3",
			NodeName:    "pve3",
			IsOnline:    false,
			Templates:   []int{101},
			LastUpdated: time.Now(),
		},
	}

	pool.nodes["node-4"] = &PVENodeInstance{
		Config: types.StandaloneNode{
			ID:       "node-4",
			HostURI:  ts.URL,
			IsActive: true,
		},
		Client:   client1,
		NodeName: "pve4",
		Telemetry: types.NodeTelemetry{
			NodeID:        "node-4",
			NodeName:      "pve4",
			IsOnline:      true,
			TotalRAMMB:    131072,
			FreeRAMMB:     90000,
			Templates:     []int{999},
			LastUpdated:   time.Now(),
		},
	}

	req := types.NodePlacementRequest{
		RequiredTemplateID: 101,
		RequiredRAMMB:      2048,
	}

	client, node, nodeName, err := pool.SelectNode(context.Background(), req)
	if err != nil {
		t.Fatalf("expected SelectNode to succeed, got error: %v", err)
	}
	if nodeName != "pve1" {
		t.Errorf("expected pve1 to be selected, got %s", nodeName)
	}
	if client == nil || node == nil {
		t.Errorf("expected non-nil client and node, got client=%v, node=%v", client, node)
	}

	reqHuge := types.NodePlacementRequest{
		RequiredTemplateID: 101,
		RequiredRAMMB:      50000,
	}
	_, _, _, errHuge := pool.SelectNode(context.Background(), reqHuge)
	if errHuge == nil {
		t.Errorf("expected failure when RAM headroom is insufficient, but got no error")
	}

	reqMissingTpl := types.NodePlacementRequest{
		RequiredTemplateID: 555,
		RequiredRAMMB:      1024,
	}
	_, _, _, errMissing := pool.SelectNode(context.Background(), reqMissingTpl)
	if errMissing == nil {
		t.Errorf("expected failure when template is missing across all nodes, but got no error")
	}
}
