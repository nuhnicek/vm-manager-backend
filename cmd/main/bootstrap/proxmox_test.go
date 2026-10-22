package bootstrap_test

import (
	"context"
	"os"
	"testing"
	"vm-manager-backend/cmd/main/bootstrap"

	"github.com/joho/godotenv"
)

func TestProxmoxPoolNodes(t *testing.T) {
	_ = godotenv.Load("../../.env")
	err, config := bootstrap.Enviroment()
	if err != nil {
		t.Fatalf("Enviroment failed: %v", err)
	}

	if len(config.ProxmoxURIs) != 4 {
		t.Fatalf("expected 4 Proxmox URIs in config, got %d: %v", len(config.ProxmoxURIs), config.ProxmoxURIs)
	}

	user := os.Getenv("PROXMOX_USERNAME")
	pass := os.Getenv("PROXMOX_PASSWORD")

	pool, err := bootstrap.ConnectProxmoxPool(user, pass, config.ProxmoxURIs)
	if err != nil {
		t.Fatalf("ConnectProxmoxPool failed: %v", err)
	}

	if len(pool) != 4 {
		t.Errorf("expected 4 connected nodes in pool, got %d", len(pool))
	}

	for i, node := range pool {
		version, err := node.Client.Version(context.Background())
		if err != nil {
			t.Errorf("Node %d (%s, name=%s) failed Version(): %v", i, node.HostURI, node.NodeName, err)
		} else {
			t.Logf("Node %d: URI=%s, NodeName=%s, PVE Version=%s (release %s)", i, node.HostURI, node.NodeName, version.Version, version.Release)
		}
	}
}
