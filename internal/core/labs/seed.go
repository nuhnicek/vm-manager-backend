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
			Name:        "Podniková DMZ a architektura firewallu",
			Category:    "Kyberbezpečnost a architektura",
			Difficulty:  "Středně pokročilý",
			Description: "Konfigurace vícesegmentové síťové topologie obsahující bránu/směrovač, veřejný DMZ webový server, interní stanici a externí testovací prostředí útočníka.",
			Objectives: []string{
				"Analýza izolovaných point-to-point 802.1Q VLAN značek mezi uzly.",
				"Konfigurace iptables/firewall pravidel na bráně pro povolení HTTP/HTTPS pouze do DMZ.",
				"Testování směrování a filtrace paketů z útočného stroje do DMZ.",
				"Ověření úplné izolace interní sítě LAN od externích podsítí.",
			},
			VMs: []types.LabVMTemplate{
				{
					Name:        "Brana-Router",
					TemplateID:  defaultTemplateID,
					Description: "Směrovač se dvěma rozhraními pro řízení provozu WAN, DMZ a LAN",
					RAM:         2048,
				},
				{
					Name:        "DMZ-WebovyServer",
					TemplateID:  defaultTemplateID,
					Description: "Veřejně přístupný webový a aplikační server v DMZ",
					RAM:         2048,
				},
				{
					Name:        "Interni-Stanice",
					TemplateID:  defaultTemplateID,
					Description: "Zabezpečená interní pracovní stanice v podsíti LAN",
					RAM:         2048,
				},
				{
					Name:        "Externi-Utocnik",
					TemplateID:  defaultTemplateID,
					Description: "Simulovaný útočný stroj připojený k WAN",
					RAM:         2048,
				},
			},
			Networks: []types.LabNetworkLink{
				{
					FromVM:  "Externi-Utocnik",
					FromNIC: "net0",
					ToVM:    "Brana-Router",
					ToNIC:   "net0",
					Label:   "WAN propojení",
				},
				{
					FromVM:  "Brana-Router",
					FromNIC: "net1",
					ToVM:    "DMZ-WebovyServer",
					ToNIC:   "net0",
					Label:   "Podsíť DMZ (172.16.10.0/24)",
				},
				{
					FromVM:  "Brana-Router",
					FromNIC: "net2",
					ToVM:    "Interni-Stanice",
					ToNIC:   "net0",
					Label:   "Interní LAN (10.0.1.0/24)",
				},
			},
			CreatedBy: "system",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "lab-pentest-sandbox",
			Name:        "Penetrační testování a bezpečnostní sandbox",
			Category:    "Kyberbezpečnost",
			Difficulty:  "Začátečník",
			Description: "Izolované prostředí o dvou uzlech pro průzkum, skenování sítě nmapem, enumeraci portů a ověřování zranitelností.",
			Objectives: []string{
				"Provedení ping sweepu a detekce služeb na izolované podsíti.",
				"Analýza otevřených portů a zjištění verzí běžících démonů.",
				"Provedení penetračního testování v zabezpečeném, nesměrovatelném prostředí.",
			},
			VMs: []types.LabVMTemplate{
				{
					Name:        "Bezpecnostni-Auditor",
					TemplateID:  defaultTemplateID,
					Description: "Prostředí s ofenzivními nástroji (Kali / Auditor)",
					RAM:         2048,
				},
				{
					Name:        "Cilova-Sluzba",
					TemplateID:  defaultTemplateID,
					Description: "Zranitelný stroj se špatně nakonfigurovanými službami",
					RAM:         2048,
				},
			},
			Networks: []types.LabNetworkLink{
				{
					FromVM:  "Bezpecnostni-Auditor",
					FromNIC: "net0",
					ToVM:    "Cilova-Sluzba",
					ToNIC:   "net0",
					Label:   "Izolovaný útočný rozsah (192.168.100.0/24)",
				},
			},
			CreatedBy: "system",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "lab-dynamic-routing",
			Name:        "Dynamické směrování více routerů a OSPF",
			Category:    "Počítačové sítě",
			Difficulty:  "Pokročilý",
			Description: "Trojúhelníková topologie tří směrovačů pro konfiguraci protokolů dynamického směrování (OSPF / BGP), metrických cen a konvergence linkových stavů.",
			Objectives: []string{
				"Přiřazení point-to-point /30 IPv4 podsítí na každé tranzitní lince.",
				"Konfigurace OSPF Area 0 napříč všemi třemi směrovači.",
				"Simulace výpadku rozhraní a sledování přepočtu směrovacích cest.",
			},
			VMs: []types.LabVMTemplate{
				{
					Name:        "Router-Alfa",
					TemplateID:  defaultTemplateID,
					Description: "Hraniční směrovač Alfa",
					RAM:         1024,
				},
				{
					Name:        "Router-Bravo",
					TemplateID:  defaultTemplateID,
					Description: "Páteřní směrovač Bravo",
					RAM:         1024,
				},
				{
					Name:        "Router-Charlie",
					TemplateID:  defaultTemplateID,
					Description: "Distribuční směrovač Charlie",
					RAM:         1024,
				},
			},
			Networks: []types.LabNetworkLink{
				{
					FromVM:  "Router-Alfa",
					FromNIC: "net0",
					ToVM:    "Router-Bravo",
					ToNIC:   "net0",
					Label:   "Linka Alfa-Bravo (10.0.12.0/30)",
				},
				{
					FromVM:  "Router-Bravo",
					FromNIC: "net1",
					ToVM:    "Router-Charlie",
					ToNIC:   "net0",
					Label:   "Linka Bravo-Charlie (10.0.23.0/30)",
				},
				{
					FromVM:  "Router-Charlie",
					FromNIC: "net1",
					ToVM:    "Router-Alfa",
					ToNIC:   "net1",
					Label:   "Linka Charlie-Alfa (10.0.31.0/30)",
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
