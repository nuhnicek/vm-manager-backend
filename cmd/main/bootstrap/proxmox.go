package bootstrap

import (
	"context"

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
