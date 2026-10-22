package bootstrap

import (
	"context"
	"fmt"
	"vm-manager-backend/types"

	"github.com/luthermonson/go-proxmox"
)

func ConnectProxmox(username string, password string, uri string) (*proxmox.Client, error) {
	credentials := proxmox.Credentials{
		Username: username,
		Password: password,
	}
	client := proxmox.NewClient(uri,
		proxmox.WithCredentials(&credentials),
		proxmox.WithInsecureSkipVerify(),
	)

	_, err := client.Version(context.Background())
	if err != nil {
		return nil, err
	}
	return client, nil
}

func ConnectProxmoxPool(username string, password string, uris []string) ([]types.ProxmoxNodeClient, error) {
	var pool []types.ProxmoxNodeClient
	var lastErr error

	for _, uri := range uris {
		client, err := ConnectProxmox(username, password, uri)
		if err != nil {
			fmt.Printf("Warning: failed to connect to Proxmox node at %s: %v\n", uri, err)
			lastErr = err
			continue
		}

		nodeName := ""
		nodes, err := client.Nodes(context.Background())
		if err == nil && len(nodes) > 0 {
			nodeName = nodes[0].Node
		}
		if nodeName == "" {
			nodeName = "pve"
		}

		pool = append(pool, types.ProxmoxNodeClient{
			Client:   client,
			NodeName: nodeName,
			HostURI:  uri,
		})
	}

	if len(pool) == 0 {
		return nil, fmt.Errorf("failed to connect to any Proxmox nodes in pool: %w", lastErr)
	}

	return pool, nil
}
