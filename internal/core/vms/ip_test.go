package vms

import (
	"testing"
	"vm-manager-backend/types"
)

func TestParseNetIdx(t *testing.T) {
	cases := []struct {
		nic      string
		expected int
	}{
		{"net0", 0},
		{"net1", 1},
		{"net2", 2},
		{"net10", 10},
		{"NET3", 3},
		{"", 0},
		{"eth0", 0},
		{"invalid", 0},
	}

	for _, c := range cases {
		result := parseNetIdx(c.nic)
		if result != c.expected {
			t.Errorf("parseNetIdx(%q) = %d; expected %d", c.nic, result, c.expected)
		}
	}
}

func TestSetMachineIPValidation(t *testing.T) {
	conn := types.Conn{}
	err := SetMachineIP(conn, "test-user", "student", types.SetIPRequest{
		VMID:   100,
		IsDHCP: false,
		IP:     "",
	})
	if err == nil {
		t.Fatalf("expected error for empty IP when is_dhcp is false")
	}

	err = SetMachineIP(conn, "test-user", "student", types.SetIPRequest{
		VMID:   100,
		IsDHCP: false,
		IP:     "invalid-ip",
	})
	if err == nil {
		t.Fatalf("expected error for invalid IP format")
	}
}
