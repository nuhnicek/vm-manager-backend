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

All `/vms/*` endpoints and `/ping` require the `Authorization` header containing a valid session token obtained from `/auth/login`.

| Method | Endpoint | Description | Payload / Parameters |
|:---|:---|:---|:---|
| `POST` | `/auth/login` | Authenticate user and receive session token | `{"email": "...", "password": "..."}` |
| `GET` | `/ping` | Verify session validity | Headers: `Authorization: <token>` |
| `GET` | `/vms` | List user's VMs with status, RAM, NICs, and links | Query Proxmox node for `owner-<uuid>` |
| `GET` | `/vms/templates` | List available template VMs | Returns template IDs and names |
| `POST` | `/vms/create` | Clone a VM from template | `{"name": "my-vm", "template_id": 100}` |
| `POST` | `/vms/start` | Power on a virtual machine | `{"id": 101}` |
| `POST` | `/vms/stop` | Power off a virtual machine | `{"id": 101}` |
| `POST` | `/vms/delete` | Destroy a virtual machine | `{"id": 101}` |
| `POST` | `/vms/network/add-nic` | Add next available VirtIO NIC (`net0`–`net31`) | `{"vmid": 101}` |
| `POST` | `/vms/network/remove` | Detach a network interface | `{"vmid": 101, "nic": "net1"}` |
| `POST` | `/vms/network/create` | Interconnect two VMs via dedicated VLAN | `{"vmid1": 101, "vmid2": 102, "nic1": "net1", "nic2": "net1"}` |
| `POST` | `/vms/network/disconnect` | Remove link between two VMs | `{"vmid1": 101, "vmid2": 102}` |
| `WS` | `/vms/console/vnc` | noVNC RFB binary stream proxy | Query params: `?vmid=101&token=<token>` |
| `WS` | `/vms/console/ws` | xterm.js serial terminal proxy | Query params: `?vmid=101&token=<token>` |

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
