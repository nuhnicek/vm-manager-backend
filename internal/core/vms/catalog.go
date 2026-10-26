package vms

import (
	"context"
	"fmt"
	"strings"
	"time"
	"vm-manager-backend/internal/core/projects"
	"vm-manager-backend/types"

	"github.com/google/uuid"
	"github.com/luthermonson/go-proxmox"
)

var defaultCatalog = []types.CatalogItem{
	{
		ID:           "lxc-alpine-324",
		Name:         "Alpine Linux 3.24",
		Type:         "lxc",
		Category:     "Rychlé kontejnery",
		Description:  "Bleskurychlý ultralehký kontejner. Start pod 2 sekundy, minimální spotřeba RAM a disku. Ideální pro mikroslužby a testování.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   256,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: false,
		Ready:        true,
	},
	{
		ID:           "lxc-ubuntu-2404",
		Name:         "Ubuntu 24.04 LTS",
		Type:         "lxc",
		Category:     "Rychlé kontejnery",
		Description:  "Moderní Ubuntu Server kontejner s plnou podporou APT repozitářů, Pythonu, Node.js a síťových nástrojů.",
		OSTemplate:   "local:vztmpl/ubuntu-24.04-standard_24.04-2_amd64.tar.zst",
		DefaultRAM:   1024,
		DefaultDisk:  8,
		DefaultCPUs:  1,
		IsVulnerable: false,
		Ready:        true,
	},
	{
		ID:           "lxc-debian-12",
		Name:         "Debian 12 Bookworm",
		Type:         "lxc",
		Category:     "Rychlé kontejnery",
		Description:  "Stabilní a osvědčená distribuce Debian vhodná pro síťové laboratoře, servery a výuku administrace.",
		OSTemplate:   "local:vztmpl/debian-12-standard_12.12-1_amd64.tar.zst",
		DefaultRAM:   512,
		DefaultDisk:  8,
		DefaultCPUs:  1,
		IsVulnerable: false,
		Ready:        true,
	},
	{
		ID:           "lxc-arch",
		Name:         "Arch Linux Base",
		Type:         "lxc",
		Category:     "Rychlé kontejnery",
		Description:  "Rolling-release systém se správcem balíčků Pacman. Vhodné pro pokročilé uživatele a nejnovější verze knihoven.",
		OSTemplate:   "local:vztmpl/archlinux-base_20260420-1_amd64.tar.zst",
		DefaultRAM:   512,
		DefaultDisk:  8,
		DefaultCPUs:  1,
		IsVulnerable: false,
		Ready:        true,
	},
	{
		ID:           "lxc-rocky-9",
		Name:         "Rocky Linux 9",
		Type:         "lxc",
		Category:     "Rychlé kontejnery",
		Description:  "Binárně kompatibilní RHEL distribuce vhodná pro simulaci podnikového prostředí a práci s RPM balíčky.",
		OSTemplate:   "local:vztmpl/rockylinux-9-default_20240912_amd64.tar.xz",
		DefaultRAM:   1024,
		DefaultDisk:  8,
		DefaultCPUs:  1,
		IsVulnerable: false,
		Ready:        true,
	},
	{
		ID:           "vuln-metasploitable2",
		Name:         "Metasploitable 2 (VulnHub)",
		Type:         "qemu",
		Category:     "Zranitelné cíle",
		Description:  "Legendární zranitelný cvičný stroj pro penetrační testy a etický hacking. Obsahuje přes 20 záměrně zranitelných síťových služeb (vsftpd backdoor, distcc, Samba, UnrealIRCd, MySQL, telnet).",
		TemplateID:   9006,
		DefaultRAM:   512,
		DefaultDisk:  8,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
	},
	{
		ID:           "vuln-weblab",
		Name:         "WebLab Vulnerable App",
		Type:         "qemu",
		Category:     "Zranitelné cíle",
		Description:  "Zranitelná webová laboratoř pro trénink OWASP zranitelností (SQL Injection, XSS, Command Injection, LFI/RFI, CSRF, insecure uploady).",
		TemplateID:   9005,
		DefaultRAM:   1024,
		DefaultDisk:  10,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
	},
	{
		ID:           "vuln-ctf-box",
		Name:         "VulnHub Linux CTF Box (LXC)",
		Type:         "lxc",
		Category:     "Zranitelné cíle",
		Description:  "Extrémně rychlý zranitelný Linux kontejner pro nácvik CTF výzev, eskalace práv (privesc) a zranitelností síťových démonů. Okamžitý start do 2 sekund.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   512,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
	},
	{
		ID:           "sec-kali-gui",
		Name:         "Kali Linux (Attacker Station)",
		Type:         "qemu",
		Category:     "Bezpečnostní nástroje",
		Description:  "Plně vybavená útočná a penetrační stanice s grafickým prostředím, obsahující nástroje Nmap, Metasploit Framework, Wireshark, Burp Suite a SQLmap.",
		TemplateID:   9010,
		DefaultRAM:   4096,
		DefaultDisk:  20,
		DefaultCPUs:  2,
		IsVulnerable: false,
		Ready:        true,
	},
}

func ListCatalog(conn types.Conn) ([]types.CatalogItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := make([]types.CatalogItem, len(defaultCatalog))
	copy(res, defaultCatalog)

	pool := conn.GetPool()
	if len(pool) == 0 {
		return res, nil
	}

	for i := range res {
		item := &res[i]
		if item.Type == "qemu" && item.TemplateID > 0 {
			_, _, err := conn.FindTemplate(ctx, item.TemplateID)
			if err != nil {
				item.Ready = false
			}
		}
	}

	return res, nil
}

func DeployCatalogItem(conn types.Conn, userUUID, role string, req types.DeployCatalogRequest) (*types.Vm, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var selectedItem *types.CatalogItem
	for _, item := range defaultCatalog {
		if item.ID == req.CatalogID {
			selectedItem = &item
			break
		}
	}

	if selectedItem == nil {
		return nil, fmt.Errorf("položka katalogu '%s' nebyla nalezena", req.CatalogID)
	}

	pool := conn.GetPool()
	if len(pool) == 0 {
		return nil, fmt.Errorf("není dostupný žádný Proxmox uzel")
	}

	targetNode := req.TargetNode
	var selectedNodeClient *types.ProxmoxNodeClient
	for i := range pool {
		if targetNode == "" || pool[i].NodeName == targetNode {
			selectedNodeClient = &pool[i]
			targetNode = pool[i].NodeName
			break
		}
	}
	if selectedNodeClient == nil {
		selectedNodeClient = &pool[0]
		targetNode = pool[0].NodeName
	}

	node, err := selectedNodeClient.Client.Node(ctx, targetNode)
	if err != nil {
		return nil, fmt.Errorf("chyba připojení k uzlu %s: %w", targetNode, err)
	}

	newVMID, err := conn.NextPoolVMID(ctx)
	if err != nil {
		return nil, fmt.Errorf("nepodařilo se vygenerovat nové VMID: %w", err)
	}

	vmUUID := uuid.New().String()
	name := strings.TrimSpace(req.Name)
	if name == "" {
		baseName := strings.ToLower(strings.ReplaceAll(selectedItem.Name, " ", "-"))
		name = fmt.Sprintf("%s-%d", baseName, newVMID)
	}

	if selectedItem.Type == "qemu" {
		templateVM, _, err := conn.FindTemplate(ctx, selectedItem.TemplateID, targetNode)
		if err != nil {
			return nil, fmt.Errorf("šablona %d nebyla nalezena: %w", selectedItem.TemplateID, err)
		}

		cloneOpts := &proxmox.VirtualMachineCloneOptions{
			NewID:   newVMID,
			Name:    name,
			Full:    proxmox.IntOrBool(true),
			Storage: "local-lvm",
		}

		clonedID, task, err := templateVM.Clone(ctx, cloneOpts)
		if err != nil {
			return nil, fmt.Errorf("klonování šablony selhalo: %w", err)
		}

		if task != nil {
			if err := task.WaitFor(ctx, 3); err != nil {
				return nil, fmt.Errorf("čekání na naklonování VM selhalo: %w", err)
			}
		}

		newVM, err := node.VirtualMachine(ctx, int(clonedID))
		if err != nil {
			return nil, fmt.Errorf("nepodařilo se načíst vytvořený stroj %d: %w", clonedID, err)
		}

		tagList := []string{"vm-" + vmUUID, "owner-" + userUUID}
		if selectedItem.IsVulnerable {
			tagList = append(tagList, "vulnerable", "cybersec")
		}
		tags := strings.Join(tagList, ";")

		cfgOpts := []proxmox.VirtualMachineOption{
			{Name: "tags", Value: tags},
			{Name: "serial0", Value: "socket"},
		}

		if req.RAM > 0 {
			cfgOpts = append(cfgOpts, proxmox.VirtualMachineOption{Name: "memory", Value: req.RAM})
		}
		if req.Cores > 0 {
			cfgOpts = append(cfgOpts, proxmox.VirtualMachineOption{Name: "cores", Value: req.Cores})
		}

		if req.IsDHCP {
			cfgOpts = append(cfgOpts, proxmox.VirtualMachineOption{Name: "ipconfig0", Value: "ip=dhcp"})
		} else if req.IP != "" {
			ipVal := strings.TrimSpace(req.IP)
			if !strings.Contains(ipVal, "/") {
				ipVal += "/24"
			}
			if req.Gateway != "" {
				ipVal += fmt.Sprintf(",gw=%s", strings.TrimSpace(req.Gateway))
			}
			cfgOpts = append(cfgOpts, proxmox.VirtualMachineOption{Name: "ipconfig0", Value: fmt.Sprintf("ip=%s", ipVal)})
		}

		_ = newVM.ConfigSync(ctx, cfgOpts...)

		startTask, err := newVM.Start(ctx)
		if err == nil && startTask != nil {
			_ = startTask.WaitFor(ctx, 3)
		}

		if req.ProjectID != "" {
			_ = projects.AddVMToProject(conn, req.ProjectID, int(clonedID), userUUID, role == "teacher" || role == "admin")
		}

		return &types.Vm{
			Id:           proxmox.StringOrUint64(clonedID),
			Name:         name,
			UUID:         vmUUID,
			Type:         "qemu",
			Status:       "Running",
			RAM:          selectedItem.DefaultRAM,
			Node:         targetNode,
			Owner:        userUUID,
			IsVulnerable: selectedItem.IsVulnerable,
			Category:     selectedItem.Category,
			ProjectID:    req.ProjectID,
		}, nil
	}

	cores := selectedItem.DefaultCPUs
	if req.Cores > 0 {
		cores = req.Cores
	}
	ram := selectedItem.DefaultRAM
	if req.RAM > 0 {
		ram = req.RAM
	}
	disk := selectedItem.DefaultDisk
	if req.Disk > 0 {
		disk = req.Disk
	}

	var net0Val string
	if req.IsDHCP {
		net0Val = "name=eth0,bridge=vmbr0,ip=dhcp,type=veth"
	} else if req.IP != "" {
		ipVal := strings.TrimSpace(req.IP)
		if !strings.Contains(ipVal, "/") {
			ipVal += "/24"
		}
		net0Val = fmt.Sprintf("name=eth0,bridge=vmbr0,ip=%s,type=veth", ipVal)
		if req.Gateway != "" {
			net0Val += fmt.Sprintf(",gw=%s", strings.TrimSpace(req.Gateway))
		}
	} else {
		net0Val = "name=eth0,bridge=vmbr0,ip=dhcp,type=veth"
	}

	password := req.Password
	if password == "" {
		password = "Lab411Admin!"
	}

	tagList := []string{"vm-" + vmUUID, "owner-" + userUUID, "container", "lxc"}
	if selectedItem.IsVulnerable {
		tagList = append(tagList, "vulnerable", "cybersec", "ctf")
	}
	tags := strings.Join(tagList, ";")

	ctOpts := []proxmox.ContainerOption{
		{Name: "ostemplate", Value: selectedItem.OSTemplate},
		{Name: "hostname", Value: name},
		{Name: "cores", Value: cores},
		{Name: "memory", Value: ram},
		{Name: "swap", Value: 512},
		{Name: "rootfs", Value: fmt.Sprintf("local-lvm:%d", disk)},
		{Name: "net0", Value: net0Val},
		{Name: "password", Value: password},
		{Name: "unprivileged", Value: 1},
		{Name: "features", Value: "nesting=1"},
		{Name: "tags", Value: tags},
		{Name: "start", Value: 1},
	}

	task, err := node.NewContainer(ctx, newVMID, ctOpts...)
	if err != nil {
		return nil, fmt.Errorf("vytvoření kontejneru selhalo: %w", err)
	}

	if task != nil {
		if err := task.WaitFor(ctx, 5); err != nil {
			return nil, fmt.Errorf("čekání na dokončení kontejneru selhalo: %w", err)
		}
	}

	ct, err := node.Container(ctx, newVMID)
	if err == nil && ct != nil && !strings.EqualFold(ct.Status, "running") {
		startTask, err := ct.Start(ctx)
		if err == nil && startTask != nil {
			_ = startTask.WaitFor(ctx, 3)
		}
	}

	if req.ProjectID != "" {
		_ = projects.AddVMToProject(conn, req.ProjectID, newVMID, userUUID, role == "teacher" || role == "admin")
	}

	return &types.Vm{
		Id:           proxmox.StringOrUint64(newVMID),
		Name:         name,
		UUID:         vmUUID,
		Type:         "lxc",
		Status:       "Running",
		RAM:          ram,
		Node:         targetNode,
		Owner:        userUUID,
		IsVulnerable: selectedItem.IsVulnerable,
		Category:     selectedItem.Category,
		ProjectID:    req.ProjectID,
	}, nil
}
