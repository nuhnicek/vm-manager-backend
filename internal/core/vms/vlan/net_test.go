package vlan

import (
	"testing"

	"github.com/luthermonson/go-proxmox"
)

func TestGetNextAvailableNet(t *testing.T) {
	tests := []struct {
		name     string
		vm       *proxmox.VirtualMachine
		expected string
		wantErr  bool
	}{
		{
			name: "no nets map initialized",
			vm: &proxmox.VirtualMachine{
				VirtualMachineConfig: &proxmox.VirtualMachineConfig{},
			},
			expected: "net1",
			wantErr:  false,
		},
		{
			name: "net1 taken",
			vm: &proxmox.VirtualMachine{
				VirtualMachineConfig: &proxmox.VirtualMachineConfig{
					Nets: map[string]string{
						"net1": "model=virtio,bridge=vmbr1,tag=100,firewall=1",
					},
				},
			},
			expected: "net2",
			wantErr:  false,
		},
		{
			name: "net1 and net2 taken",
			vm: &proxmox.VirtualMachine{
				VirtualMachineConfig: &proxmox.VirtualMachineConfig{
					Nets: map[string]string{
						"net1": "model=virtio,bridge=vmbr1,tag=100,firewall=1",
						"net2": "model=virtio,bridge=vmbr1,tag=101,firewall=1",
					},
				},
			},
			expected: "net3",
			wantErr:  false,
		},
		{
			name: "nil config",
			vm: &proxmox.VirtualMachine{
				VirtualMachineConfig: nil,
			},
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetNextAvailableNet(tt.vm)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetNextAvailableNet() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("GetNextAvailableNet() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseVLAN(t *testing.T) {
	tests := []struct {
		config   string
		expected int
	}{
		{"model=virtio,bridge=vmbr1,tag=100,firewall=1", 100},
		{"virtio=BC:24:11:00:11:22,bridge=vmbr1,tag=4050", 4050},
		{"model=virtio,bridge=vmbr0,firewall=1", 0},
		{"", 0},
	}

	for _, tt := range tests {
		got := ParseVLAN(tt.config)
		if got != tt.expected {
			t.Errorf("ParseVLAN(%q) = %d, want %d", tt.config, got, tt.expected)
		}
	}
}
