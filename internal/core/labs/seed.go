package labs

import (
	"context"
	"fmt"
	"time"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func SeedDefaultLabs(conn types.Conn) error {
	coll := conn.Mongo.Database("lab411").Collection("lab_templates")

	count, err := coll.CountDocuments(context.TODO(), bson.M{})
	if err != nil {
		return fmt.Errorf("failed to check existing lab templates: %w", err)
	}

	defaultTemplateID := 0
	if conn.Proxmox != nil {
		if tpls, err := vms.ListTemplates(conn); err == nil && len(tpls) > 0 {
			defaultTemplateID = int(tpls[0].Id)
		}
	}

	if count > 0 {
		// If templates exist but are referencing a missing template (e.g. legacy 9000),
		// migrate them to the active template in the cluster!
		if defaultTemplateID > 0 {
			var existingTemplates []types.LabTemplate
			cur, err := coll.Find(context.TODO(), bson.M{})
			if err == nil {
				if err := cur.All(context.TODO(), &existingTemplates); err == nil {
					for _, tpl := range existingTemplates {
						modified := false
						for idx := range tpl.VMs {
							if tpl.VMs[idx].TemplateID == 9000 || tpl.VMs[idx].TemplateID == 0 {
								tpl.VMs[idx].TemplateID = defaultTemplateID
								modified = true
							}
						}
						if modified {
							_, _ = coll.UpdateOne(context.TODO(), bson.M{"_id": tpl.ID}, bson.M{"$set": bson.M{"vms": tpl.VMs}})
						}
					}
				}
			}
		}
		return nil
	}

	now := time.Now()
	defaultTemplates := []types.LabTemplate{
		{
			ID:          "lab-dmz-firewall",
			Name:        "Corporate DMZ & Firewall Architecture",
			Category:    "Cybersecurity & Architecture",
			Difficulty:  "Intermediate",
			Description: "Configure a multi-segment network topology containing a Gateway/Router, a public DMZ Web Server, an internal Workstation, and an external Attacker sandbox.",
			Objectives: []string{
				"Inspect isolated point-to-point 802.1Q VLAN tags between all nodes.",
				"Configure iptables/firewall rules on Gateway to allow HTTP/HTTPS to DMZ only.",
				"Test routing and packet filtering from Attacker to DMZ.",
				"Verify total isolation of the internal LAN from external subnets.",
			},
			VMs: []types.LabVMTemplate{
				{
					Name:        "Gateway-Router",
					TemplateID:  defaultTemplateID,
					Description: "Dual-homed router managing WAN, DMZ, and LAN traffic",
					RAM:         2048,
				},
				{
					Name:        "DMZ-WebServer",
					TemplateID:  defaultTemplateID,
					Description: "Public-facing web & application server in DMZ",
					RAM:         2048,
				},
				{
					Name:        "Internal-Workstation",
					TemplateID:  defaultTemplateID,
					Description: "Protected internal workstation on LAN subnet",
					RAM:         2048,
				},
				{
					Name:        "External-Attacker",
					TemplateID:  defaultTemplateID,
					Description: "Simulated adversary machine connected to WAN",
					RAM:         2048,
				},
			},
			Networks: []types.LabNetworkLink{
				{
					FromVM:  "External-Attacker",
					FromNIC: "net0",
					ToVM:    "Gateway-Router",
					ToNIC:   "net0",
					Label:   "WAN Interconnect",
				},
				{
					FromVM:  "Gateway-Router",
					FromNIC: "net1",
					ToVM:    "DMZ-WebServer",
					ToNIC:   "net0",
					Label:   "DMZ Subnet (172.16.10.0/24)",
				},
				{
					FromVM:  "Gateway-Router",
					FromNIC: "net2",
					ToVM:    "Internal-Workstation",
					ToNIC:   "net0",
					Label:   "Internal LAN (10.0.1.0/24)",
				},
			},
			CreatedBy: "system",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "lab-pentest-sandbox",
			Name:        "Offensive Security & Pen-Testing Sandbox",
			Category:    "Cybersecurity",
			Difficulty:  "Beginner",
			Description: "Isolated two-node range for reconnaissance, nmap network scanning, port enumeration, and vulnerability verification.",
			Objectives: []string{
				"Perform ping sweep and service detection on the isolated subnet.",
				"Analyze open ports and determine listening daemon versions.",
				"Conduct penetration testing within a secure, non-routable environment.",
			},
			VMs: []types.LabVMTemplate{
				{
					Name:        "Security-Auditor",
					TemplateID:  defaultTemplateID,
					Description: "Offensive tooling environment (Kali / Auditor)",
					RAM:         2048,
				},
				{
					Name:        "Target-Service",
					TemplateID:  defaultTemplateID,
					Description: "Vulnerable service machine with misconfigured services",
					RAM:         2048,
				},
			},
			Networks: []types.LabNetworkLink{
				{
					FromVM:  "Security-Auditor",
					FromNIC: "net0",
					ToVM:    "Target-Service",
					ToNIC:   "net0",
					Label:   "Isolated Attack Range (192.168.100.0/24)",
				},
			},
			CreatedBy: "system",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "lab-dynamic-routing",
			Name:        "Multi-Router Dynamic Routing & OSPF",
			Category:    "Networking",
			Difficulty:  "Advanced",
			Description: "Triangular three-router topology for configuring dynamic routing protocols (OSPF / BGP), metric costs, and link-state convergence.",
			Objectives: []string{
				"Assign point-to-point /30 IPv4 subnets across each transit link.",
				"Configure OSPF Area 0 across all three routers.",
				"Simulate interface failure and observe route recalculation.",
			},
			VMs: []types.LabVMTemplate{
				{
					Name:        "Router-Alpha",
					TemplateID:  defaultTemplateID,
					Description: "Border Router Alpha",
					RAM:         1024,
				},
				{
					Name:        "Router-Bravo",
					TemplateID:  defaultTemplateID,
					Description: "Core Router Bravo",
					RAM:         1024,
				},
				{
					Name:        "Router-Charlie",
					TemplateID:  defaultTemplateID,
					Description: "Distribution Router Charlie",
					RAM:         1024,
				},
			},
			Networks: []types.LabNetworkLink{
				{
					FromVM:  "Router-Alpha",
					FromNIC: "net0",
					ToVM:    "Router-Bravo",
					ToNIC:   "net0",
					Label:   "Link Alpha-Bravo (10.0.12.0/30)",
				},
				{
					FromVM:  "Router-Bravo",
					FromNIC: "net1",
					ToVM:    "Router-Charlie",
					ToNIC:   "net0",
					Label:   "Link Bravo-Charlie (10.0.23.0/30)",
				},
				{
					FromVM:  "Router-Charlie",
					FromNIC: "net1",
					ToVM:    "Router-Alpha",
					ToNIC:   "net1",
					Label:   "Link Charlie-Alpha (10.0.31.0/30)",
				},
			},
			CreatedBy: "system",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	for _, tpl := range defaultTemplates {
		_, _ = coll.InsertOne(context.TODO(), tpl)
	}

	return nil
}
