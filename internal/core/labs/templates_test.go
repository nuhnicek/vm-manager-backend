package labs

import (
	"testing"
	"vm-manager-backend/types"
)

func TestLabTemplateValidation(t *testing.T) {
	conn := types.Conn{}

	_, err := CreateLabTemplate(conn, types.LabTemplate{Name: ""}, "test-creator")
	if err == nil {
		t.Fatal("expected error for empty template name, got nil")
	}

	_, err = CreateLabTemplate(conn, types.LabTemplate{
		Name: "Test Lab",
		VMs:  []types.LabVMTemplate{},
	}, "test-creator")
	if err == nil {
		t.Fatal("expected error for template without VMs, got nil")
	}
}

func TestLabTemplateStructure(t *testing.T) {
	tpl := types.LabTemplate{
		ID:          "test-lab-1",
		Name:        "DMZ Architecture Test",
		Category:    "Cybersecurity",
		Difficulty:  "Intermediate",
		Description: "Testing lab topology",
		Objectives:  []string{"Objective 1", "Objective 2"},
		VMs: []types.LabVMTemplate{
			{Name: "Gateway", TemplateID: 9000, Description: "Router", RAM: 2048},
			{Name: "WebServer", TemplateID: 9000, Description: "Web", RAM: 1024},
		},
		Networks: []types.LabNetworkLink{
			{FromVM: "Gateway", FromNIC: "net0", ToVM: "WebServer", ToNIC: "net0", Label: "DMZ Link"},
		},
	}

	if tpl.Name != "DMZ Architecture Test" {
		t.Errorf("unexpected name: %s", tpl.Name)
	}
	if len(tpl.VMs) != 2 {
		t.Errorf("expected 2 VMs, got %d", len(tpl.VMs))
	}
	if len(tpl.Networks) != 1 {
		t.Errorf("expected 1 network link, got %d", len(tpl.Networks))
	}
	if tpl.Networks[0].Label != "DMZ Link" {
		t.Errorf("unexpected link label: %s", tpl.Networks[0].Label)
	}
}
