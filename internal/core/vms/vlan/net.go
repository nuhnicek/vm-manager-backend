package vlan

import (
	"fmt"

	"github.com/luthermonson/go-proxmox"
)

func GetNextAvailableNet(vm *proxmox.VirtualMachine) (string, error) {
	if vm.VirtualMachineConfig == nil {
		return "", fmt.Errorf("vm configuration is empty or not loaded")
	}

	for i := 1; i <= 31; i++ {
		netKey := fmt.Sprintf("net%d", i)
		if vm.VirtualMachineConfig.Nets == nil {
			return netKey, nil
		}
		if _, exists := vm.VirtualMachineConfig.Nets[netKey]; !exists {
			return netKey, nil
		}
	}

	return "", fmt.Errorf("maximum network interfaces reached for VM")
}
