# Lab411 VM Manager — Backend

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Gin Framework](https://img.shields.io/badge/Gin-v1.10%2B-008ECF?logo=go&logoColor=white)](https://gin-gonic.com/)
[![Proxmox VE](https://img.shields.io/badge/Proxmox_VE-API2-E57000?logo=proxmox&logoColor=white)](https://www.proxmox.com/)
[![Redis](https://img.shields.io/badge/Redis-Session_Cache-DC382D?logo=redis&logoColor=white)](https://redis.io/)
[![MongoDB](https://img.shields.io/badge/MongoDB-Datastore-47A248?logo=mongodb&logoColor=white)](https://www.mongodb.com/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

The orchestration engine and API gateway for the **Lab411** virtual laboratory system. Built in Go with Gin, this service interfaces directly with Proxmox VE clusters to manage multi-tenant virtual machines, dynamically provision isolated 802.1Q VLAN networks, and proxy real-time WebSocket console streams (noVNC and serial termproxy).

---

## Architecture Overview

```mermaid
flowchart TD
    Client["Lab411 Desktop Client<br/>(Wails / React / noVNC)"]

    subgraph Backend["Lab411 Backend (Go / Gin)"]
        Router["HTTP & WebSocket Router"]
        AuthMiddleware["Auth Middleware (Bearer Token)"]
        VMMgmt["VM Lifecycle & Templates"]
        VLANMgr["VLAN Provisioner"]
        ConsoleProxy["Console Stream Proxy"]
    end

    subgraph Infrastructure["Infrastructure Layer"]
        Proxmox["Proxmox VE API2<br/>(QEMU / KVM / vmbr0)"]
        Redis[("Redis<br/>(Active Sessions)")]
        MongoDB[("MongoDB<br/>(Users & VLAN Counters)")]
    end

    Client -->|REST & WSS| Router
    Router --> AuthMiddleware
    AuthMiddleware -->|Validate Token| Redis
    Router --> VMMgmt
    Router --> VLANMgr
    Router --> ConsoleProxy

    VMMgmt -->|Clone, Power, Inspect| Proxmox
    VLANMgr -->|Configure VirtIO & Tags| Proxmox
    VLANMgr -->|Atomic Counter| MongoDB
    ConsoleProxy -->|Termproxy & VNC WebSocket| Proxmox
```

---

## Features

- **Multi-Tenant VM Isolation**: VMs are tagged in Proxmox with `owner-<uuid>` and `vm-<uuid>`. Users can only inspect, power-cycle, or modify virtual machines assigned to their tenant ID.
- **Fast Template Cloning**: Provisions linked clones (`Full: false`) from existing Proxmox VM templates in seconds, minimizing disk overhead.
- **Automated Serial Console Provisioning**: Injects `serial0=socket` into new VMs upon cloning so serial terminal access is available out of the box.
- **Software-Defined VLAN Interconnects**: Point-to-point and multi-node virtual networks dynamically allocated via 802.1Q tags on Linux bridge `vmbr0` (tags 100–4094), using atomic sequence counters in MongoDB.
- **Bidirectional Console Proxying**:
  - `/vms/console/vnc`: Binary WebSocket tunnel to Proxmox `vncwebsocket` for graphical display (RFB / noVNC).
  - `/vms/console/ws`: Interactive terminal tunnel to Proxmox `termproxy` for xterm.js serial consoles.
- **Token-Based Authentication**: Secure SHA-256 hashed credentials stored in MongoDB, with ephemeral 1-hour session tokens cached in Redis.

---

## API Reference

> [!TIP]
> **Interactive Swagger UI**: When running the backend server, visit [`http://localhost:8080/docs`](http://localhost:8080/docs) in your browser for the full interactive API documentation.
>
> **Full Specifications**:
> - [Comprehensive API Documentation (Markdown)](docs/API.md)
> - [OpenAPI 3.0 Specification (YAML)](docs/openapi.yaml) or served at `/openapi.yaml`
> - [OpenAPI 3.0 Specification (JSON)](docs/openapi.json) or served at `/openapi.json`

All protected endpoints and `/ping` require the `Authorization` header containing a valid session token obtained from `/auth/login` (or query param `?token=<token>` for WebSocket connections).

| Method | Endpoint | Role | Description | Payload / Parameters |
|:---|:---|:---|:---|:---|
| `GET` | `/docs` | Public | Interactive Swagger UI API documentation | Browser view |
| `GET` | `/openapi.yaml` | Public | OpenAPI 3.0 specification in YAML format | Direct download / import |
| `GET` | `/openapi.json` | Public | OpenAPI 3.0 specification in JSON format | Direct download / import |
| `POST` | `/auth/login` | Public | Authenticate user and receive session token | `{"email": "...", "password": "..."}` |
| `GET` | `/ping` | Auth | Verify session validity and user profile | Headers: `Authorization: <token>` |
| `GET` | `/nodes` | Teacher | List cluster nodes and basic telemetry | Returns node list |
| `GET` | `/vms` | Auth | List user's VMs with status, RAM, NICs, and links | Query Proxmox node for `owner-<uuid>` |
| `GET` | `/vms/templates` | Auth | List available template VMs | Returns template IDs and names |
| `POST` | `/vms/create` | Auth | Clone a VM from template | `{"name": "my-vm", "template_id": 100}` |
| `POST` | `/vms/start` | Auth | Power on a virtual machine | `{"id": 101}` |
| `POST` | `/vms/stop` | Auth | Power off a virtual machine | `{"id": 101}` |
| `POST` | `/vms/delete` | Auth | Destroy a virtual machine | `{"id": 101}` |
| `POST` | `/vms/ip` | Auth | Configure static IP/gateway or DHCP | `{"vmid": 101, "ip": "192.168.10.5/24"}` |
| `POST` | `/vms/network/add-nic` | Auth | Add next available VirtIO NIC (`net0`–`net31`) | `{"vmid": 101}` |
| `POST` | `/vms/network/remove` | Auth | Detach a network interface | `{"vmid": 101, "nic": "net1"}` |
| `POST` | `/vms/network/delete` | Auth | Detach a network interface (alias) | `{"vmid": 101, "nic": "net1"}` |
| `POST` | `/vms/network/create` | Auth | Interconnect two VMs via dedicated VLAN | `{"vmid1": 101, "vmid2": 102, "nic1": "net1", "nic2": "net1"}` |
| `POST` | `/vms/network/disconnect` | Auth | Remove link between two VMs | `{"vmid1": 101, "vmid2": 102}` |
| `POST` | `/vms/network/lab-connect` | Auth | Connect VM to shared lab network | `{"vmid": 101}` |
| `POST` | `/vms/network/lab-disconnect` | Auth | Disconnect VM from shared lab network | `{"vmid": 101}` |
| `WS` | `/vms/console/vnc` | Auth | noVNC RFB binary stream proxy | Query params: `?vmid=101&token=<token>` |
| `WS` | `/vms/console/ws` | Auth | xterm.js serial terminal proxy | Query params: `?vmid=101&token=<token>` |
| `POST` | `/vms/console/external` | Auth | Start external TCP VNC proxy listener | `{"vmid": 101}` |
| `GET` | `/vms/console/external/file` | Auth | Download `.vnc` connection profile | Query params: `?vmid=101` |
| `DELETE` | `/vms/console/external/:vmid` | Auth | Stop external TCP VNC proxy listener | VMID in path |
| `GET` | `/vms/console/clipboard` | Auth | Read guest OS clipboard via QEMU agent | Query params: `?vmid=101` |
| `POST` | `/vms/console/clipboard` | Auth | Set guest OS clipboard via QEMU agent | `{"vmid": 101, "text": "..."}` |
| `GET` | `/catalog` | Auth | List pre-configured appliances & CVE targets | Returns catalog appliances |
| `POST` | `/catalog/deploy` | Auth | Deploy appliance from catalog | `{"catalog_id": "...", "name": "..."}` |
| `GET` | `/projects` | Auth | List user projects | Returns project list |
| `POST` | `/projects` | Auth | Create project workspace | `{"name": "...", "description": "..."}` |
| `PUT` | `/projects/:id` | Auth | Update project details | Project ID in path |
| `DELETE` | `/projects/:id` | Auth | Delete project | Project ID in path |
| `POST` | `/projects/:id/vms` | Auth | Add VM(s) to project | `{"vm_id": 101}` or `{"vm_ids": [...]}` |
| `DELETE` | `/projects/:id/vms/:vmid` | Auth | Remove VM from project | Project ID and VMID in path |
| `POST` | `/vms/:id/project` | Auth | Assign VM to project | VMID in path, `{"project_id": "..."}` |
| `GET` | `/labs/templates` | Auth | List all available lab templates | Returns lab catalog |
| `GET` | `/labs/templates/:id` | Auth | Get specific lab template details | Template ID in path |
| `POST` | `/labs/templates` | Teacher | Create lab template | `{"name": "...", "vms": [...], "networks": [...]}` |
| `PUT` | `/labs/templates/:id` | Teacher | Update lab template | Template ID in path |
| `DELETE` | `/labs/templates/:id` | Teacher | Delete lab template | Template ID in path |
| `POST` | `/labs/deploy` | Auth | Deploy multi-VM lab topology with VLANs | `{"template_id": "...", "student_uuid": "..."}` |
| `GET` | `/labs/instances` | Auth | List active deployed lab instances | Returns user or student lab environments |
| `POST` | `/labs/instances/:id/start` | Auth | Power on all VMs in lab instance | Lab instance ID in path |
| `POST` | `/labs/instances/:id/stop` | Auth | Power off all VMs in lab instance | Lab instance ID in path |
| `DELETE` | `/labs/instances/:id` | Auth | Tear down lab instance and destroy VMs | Lab instance ID in path |
| `GET` | `/teacher/students` | Teacher | List student accounts and VM counts | Returns student list with metrics |
| `POST` | `/teacher/students` | Teacher | Create student or teacher account | `{"email": "...", "password": "...", "role": "..."}` |
| `DELETE` | `/teacher/students/:id` | Teacher | Delete student account | Student UUID in path |
| `POST` | `/teacher/students/:id/reset-password` | Teacher | Reset student account password | `{"password": "..."}` |
| `GET` | `/teacher/vms` | Teacher | List all VMs across all students | Full cluster VM list |
| `POST` | `/teacher/vms/bulk-action` | Teacher | Classroom bulk action (e.g. stop all) | `{"action": "stop_all"}` |
| `POST` | `/vms/templates/convert` | Teacher | Convert VM to template | `{"vmid": 101}` |
| `POST` | `/vms/templates/clone-as-template` | Teacher | Clone VM and convert clone into template | `{"source_vmid": 101, "template_name": "..."}` |
| `GET` | `/teacher/nodes` | Teacher | Node telemetry from balancing pool | Returns load balancer status |
| `POST` | `/teacher/nodes` | Teacher | Register standalone node into balancing pool | `{"name": "...", "host_uri": "...", ...}` |
| `DELETE` | `/teacher/nodes/:id` | Teacher | Remove node from balancing pool | Node ID in path |
| `POST` | `/teacher/nodes/refresh` | Teacher | Refresh telemetry across cluster nodes | Triggers immediate metric poll |

---

## Getting Started

### Prerequisites

- **Go**: 1.24 or newer
- **Docker & Docker Compose**: for local MongoDB and Redis instances
- **Proxmox VE Cluster**: accessible over HTTPS API (port 8006) with an administrative or service account (`root@pam` or API user)

### 1. Start Support Services

Start the local MongoDB and Redis containers:

```bash
docker compose up -d
```

### 2. Configure Environment

Copy the template configuration file:

```bash
cp .env.example .env
```

Edit `.env` to match your cluster environment:

```ini
PORT=:8080
MONGO_URI=mongodb://root:password@localhost:27017/
REDIS_URI=redis://localhost:6379/0

# Proxmox VE connection details
PROXMOX_URI=https://<your-proxmox-ip>:8006/api2/json
PROXMOX_USERNAME=root@pam
PROXMOX_PASSWORD=your_proxmox_password
```

### 3. Run the Server

```bash
go run cmd/main/main.go
```

The server starts on `http://localhost:8080` (or the configured `PORT`).

---

## Testing

Run the package test suite:

```bash
go test -v ./...
```

Unit tests cover:
- Proxmox template listing and cloning workflows (`internal/core/vms/templates_test.go`)
- Next-available VirtIO network slot allocation (`internal/core/vms/vlan/net_test.go`)

---

## Project Structure

```text
vm-manager-backend/
├── cmd/
│   └── main/
│       ├── bootstrap/      # App init: Mongo, Redis, Proxmox client, routes
│       └── main.go         # Entry point (Gin engine setup)
├── internal/
│   ├── core/
│   │   ├── auth/           # Login authentication & session generator
│   │   ├── middleware/     # Bearer token validation middleware
│   │   └── vms/            # VM lifecycle, console proxying, templates, VLANs
│   └── delivery/
│       └── handlers/       # Gin HTTP & WebSocket route handlers
├── pkg/                    # Utility helpers (SHA-256)
├── types/                  # Shared data models (VM, Config, Connections)
├── docker-compose.yml      # Local MongoDB & Redis service definitions
├── go.mod
└── .env.example
```

---

## License

This project is licensed under the [MIT License](LICENSE).
