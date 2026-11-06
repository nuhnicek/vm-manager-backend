# Lab411 VM Manager — API Reference Documentation

Comprehensive API documentation for the **Lab411** virtual laboratory backend service.

---

## Table of Contents

- [Overview & Architecture](#overview--architecture)
- [Authentication & Authorization](#authentication--authorization)
  - [Session Tokens](#session-tokens)
  - [Role-Based Access Control (RBAC)](#role-based-access-control-rbac)
- [API Endpoints Summary](#api-endpoints-summary)
- [1. System & Authentication](#1-system--authentication)
  - [POST /auth/login](#post-authlogin)
  - [GET /ping](#get-ping)
- [2. Virtual Machines](#2-virtual-machines)
  - [GET /vms](#get-vms)
  - [GET /vms/templates](#get-vmstemplates)
  - [POST /vms/create](#post-vmscreate)
  - [POST /vms/start](#post-vmsstart)
  - [POST /vms/stop](#post-vmsstop)
  - [POST /vms/delete](#post-vmsdelete)
  - [POST /vms/ip](#post-vmsip)
- [3. VM Networking & 802.1Q VLANs](#3-vm-networking--8021q-vlans)
  - [POST /vms/network/add-nic](#post-vmsnetworkadd-nic)
  - [POST /vms/network/remove](#post-vmsnetworkremove)
  - [POST /vms/network/delete](#post-vmsnetworkdelete)
  - [POST /vms/network/create](#post-vmsnetworkcreate)
  - [POST /vms/network/disconnect](#post-vmsnetworkdisconnect)
  - [POST /vms/network/lab-connect](#post-vmsnetworklab-connect)
  - [POST /vms/network/lab-disconnect](#post-vmsnetworklab-disconnect)
- [4. Remote Consoles & Clipboard](#4-remote-consoles--clipboard)
  - [WS /vms/console/ws](#ws-vmsconsolews)
  - [WS /vms/console/vnc](#ws-vmsconsolevnc)
  - [POST /vms/console/external](#post-vmsconsoleexternal)
  - [GET /vms/console/external/file](#get-vmsconsoleexternalfile)
  - [DELETE /vms/console/external/:vmid](#delete-vmsconsoleexternalvmid)
  - [GET /vms/console/clipboard](#get-vmsconsoleclipboard)
  - [POST /vms/console/clipboard](#post-vmsconsoleclipboard)
- [5. Appliance Catalog](#5-appliance-catalog)
  - [GET /catalog](#get-catalog)
  - [POST /catalog/deploy](#post-catalogdeploy)
- [6. Projects Management](#6-projects-management)
  - [GET /projects](#get-projects)
  - [POST /projects](#post-projects)
  - [PUT /projects/:id](#put-projectsid)
  - [DELETE /projects/:id](#delete-projectsid)
  - [POST /projects/:id/vms](#post-projectsidvms)
  - [DELETE /projects/:id/vms/:vmid](#delete-projectsidvmsvmid)
  - [POST /vms/:id/project](#post-vmsidproject)
- [7. Cyber Security Lab Blueprints](#7-cyber-security-lab-blueprints)
  - [GET /labs/templates](#get-labstemplates)
  - [GET /labs/templates/:id](#get-labstemplatesid)
  - [POST /labs/templates](#post-labstemplates)
  - [PUT /labs/templates/:id](#put-labstemplatesid)
  - [DELETE /labs/templates/:id](#delete-labstemplatesid)
  - [POST /labs/deploy](#post-labsdeploy)
  - [GET /labs/instances](#get-labsinstances)
  - [POST /labs/instances/:id/start](#post-labsinstancesidstart)
  - [POST /labs/instances/:id/stop](#post-labsinstancesidstop)
  - [DELETE /labs/instances/:id](#delete-labsinstancesid)
- [8. Teacher & Classroom Administration](#8-teacher--classroom-administration)
  - [GET /teacher/students](#get-teacherstudents)
  - [POST /teacher/students](#post-teacherstudents)
  - [DELETE /teacher/students/:id](#delete-teacherstudentsid)
  - [POST /teacher/students/:id/reset-password](#post-teacherstudentsidreset-password)
  - [GET /teacher/vms](#get-teachervms)
  - [POST /teacher/vms/bulk-action](#post-teachervmsbulk-action)
  - [POST /vms/templates/convert](#post-vmstemplatesconvert)
  - [POST /vms/templates/clone-as-template](#post-vmstemplatesclone-as-template)
- [9. Cluster Nodes & Load Balancer](#9-cluster-nodes--load-balancer)
  - [GET /nodes](#get-nodes)
  - [GET /teacher/nodes](#get-teachernodes)
  - [POST /teacher/nodes](#post-teachernodes)
  - [DELETE /teacher/nodes/:id](#delete-teachernodesid)
  - [POST /teacher/nodes/refresh](#post-teachernodesrefresh)
- [Workflow Example (cURL Walkthrough)](#workflow-example-curl-walkthrough)

---

## Overview & Architecture

Lab411 VM Manager Backend is an orchestration service and API gateway built with **Go** and the **Gin** HTTP framework. It communicates with one or more **Proxmox VE** hypervisor nodes, caches active user sessions in **Redis**, and persists users, projects, and lab templates in **MongoDB**.

- **Multi-Tenant VM Isolation**: VMs in Proxmox are tagged with `owner-<uuid>` and `vm-<uuid>`. Students only see and interact with their own VMs.
- **Dynamic 802.1Q VLANs**: Point-to-point and multi-node virtual networks dynamically allocate 802.1Q tags (100–4094) on Linux bridge `vmbr0` using an atomic MongoDB sequence counter.
- **Bi-directional WebSocket Consoles**:
  - `termproxy` tunnel (`/vms/console/ws`) for xterm.js serial consoles.
  - `vncwebsocket` tunnel (`/vms/console/vnc`) for noVNC graphical displays.
- **Standalone Load Balancer**: Dynamic placement of new clones across nodes based on active CPU/RAM telemetry and node weights.

---

## Authentication & Authorization

### Session Tokens

All protected endpoints require a valid session token obtained from `POST /auth/login`.

Provide the token using either method:
1. **HTTP Header** (standard REST calls):
   ```http
   Authorization: Bearer <session_token>
   ```
   *(Also accepts `Authorization: <session_token>`)*
2. **URL Query Parameter** (mandatory for browser WebSocket handshakes):
   ```http
   GET /vms/console/ws?vmid=105&token=<session_token>
   ```

Tokens are valid for **1 hour** from creation and are stored in Redis (`session:<uuid>` mapping to `<user_uuid>:<role>:<email>`).

### Role-Based Access Control (RBAC)

| Role | Access Level | Description |
|:---|:---|:---|
| `student` | Restricted | Can manage own VMs, own projects, deploy labs, connect to consoles. Cannot inspect other students' VMs or access `/teacher/*` endpoints. |
| `teacher` | Elevated | Full access to student accounts, classroom bulk power actions, template conversion, node telemetry, and all student VMs. |
| `admin` | Superuser | Equivalent to teacher with full infrastructure and pool modification capabilities. |

---

## API Endpoints Summary

| Method | Path | Role | Description |
|:---|:---|:---|:---|
| `POST` | `/auth/login` | Public | Authenticate user and obtain session token |
| `GET` | `/ping` | Authenticated | Validate session and retrieve user info |
| `GET` | `/nodes` | Teacher/Admin | List Proxmox cluster nodes and basic telemetry |
| `GET` | `/vms` | Authenticated | List virtual machines (tenant-filtered for students) |
| `GET` | `/vms/templates` | Authenticated | List base templates available for cloning |
| `POST` | `/vms/create` | Authenticated | Clone VM from template (linked clone) |
| `POST` | `/vms/start` | Authenticated | Power on a virtual machine |
| `POST` | `/vms/stop` | Authenticated | Power off a virtual machine |
| `POST` | `/vms/delete` | Authenticated | Destroy a virtual machine |
| `POST` | `/vms/ip` | Authenticated | Set static IP/gateway or DHCP on VM |
| `POST` | `/vms/network/add-nic` | Authenticated | Append next available VirtIO NIC (`net0`–`net31`) |
| `POST` | `/vms/network/remove` | Authenticated | Detach network interface from VM |
| `POST` | `/vms/network/delete` | Authenticated | Alias for `/vms/network/remove` |
| `POST` | `/vms/network/create` | Authenticated | Interconnect 2 VMs via isolated 802.1Q VLAN |
| `POST` | `/vms/network/disconnect`| Authenticated | Remove VLAN connection between 2 VMs |
| `POST` | `/vms/network/lab-connect` | Authenticated | Attach VM to shared lab network |
| `POST` | `/vms/network/lab-disconnect` | Authenticated | Detach VM from shared lab network |
| `GET` | `/vms/console/ws` | Authenticated (WS) | Interactive serial terminal stream (xterm.js) |
| `GET` | `/vms/console/vnc` | Authenticated (WS) | noVNC RFB binary stream proxy |
| `POST` | `/vms/console/external` | Authenticated | Start external TCP VNC proxy listener |
| `GET` | `/vms/console/external/file` | Authenticated | Download `.vnc` connection profile |
| `DELETE` | `/vms/console/external/:vmid` | Authenticated | Stop external TCP VNC proxy listener |
| `GET` | `/vms/console/clipboard` | Authenticated | Read guest clipboard via QEMU agent |
| `POST` | `/vms/console/clipboard` | Authenticated | Inject text into guest clipboard |
| `POST` | `/vms/:id/project` | Authenticated | Assign VM to project |
| `GET` | `/catalog` | Authenticated | List pre-configured appliances & CVE targets |
| `POST` | `/catalog/deploy` | Authenticated | Deploy catalog appliance |
| `GET` | `/projects` | Authenticated | List user projects |
| `POST` | `/projects` | Authenticated | Create a new project workspace |
| `PUT` | `/projects/:id` | Authenticated | Update project details |
| `DELETE` | `/projects/:id` | Authenticated | Delete project |
| `POST` | `/projects/:id/vms` | Authenticated | Add VM(s) to project |
| `DELETE` | `/projects/:id/vms/:vmid`| Authenticated | Remove VM from project |
| `GET` | `/labs/templates` | Authenticated | List multi-VM lab blueprints |
| `GET` | `/labs/templates/:id` | Authenticated | Get lab blueprint by ID |
| `POST` | `/labs/templates` | Teacher/Admin | Create lab blueprint |
| `PUT` | `/labs/templates/:id` | Teacher/Admin | Update lab blueprint |
| `DELETE` | `/labs/templates/:id` | Teacher/Admin | Delete lab blueprint |
| `POST` | `/labs/deploy` | Authenticated | Deploy multi-VM lab topology instance |
| `GET` | `/labs/instances` | Authenticated | List deployed lab instances |
| `POST` | `/labs/instances/:id/start` | Authenticated | Start all VMs in lab instance |
| `POST` | `/labs/instances/:id/stop` | Authenticated | Stop all VMs in lab instance |
| `DELETE` | `/labs/instances/:id` | Authenticated | Destroy lab instance and all member VMs |
| `GET` | `/teacher/students` | Teacher/Admin | List student accounts with metrics |
| `POST` | `/teacher/students` | Teacher/Admin | Create student account |
| `DELETE` | `/teacher/students/:id` | Teacher/Admin | Delete student account |
| `POST` | `/teacher/students/:id/reset-password` | Teacher/Admin | Reset student password |
| `GET` | `/teacher/vms` | Teacher/Admin | List all cluster VMs |
| `POST` | `/teacher/vms/bulk-action` | Teacher/Admin | Classroom bulk power action (stop_all) |
| `POST` | `/vms/templates/convert` | Teacher/Admin | Convert existing VM to template |
| `POST` | `/vms/templates/clone-as-template` | Teacher/Admin | Clone VM and convert clone into template |
| `GET` | `/teacher/nodes` | Teacher/Admin | List load balancing pool telemetry |
| `POST` | `/teacher/nodes` | Teacher/Admin | Register standalone node into balancing pool |
| `DELETE` | `/teacher/nodes/:id` | Teacher/Admin | Remove node from balancing pool |
| `POST` | `/teacher/nodes/refresh` | Teacher/Admin | Force telemetry refresh across nodes |

---

## 1. System & Authentication

### POST /auth/login

Authenticates a user with email or username and password against MongoDB (passwords hashed using SHA-256). Generates a random session UUID cached in Redis with a 1-hour expiration.

- **Access**: Public
- **Request Body**:
  ```json
  {
    "email": "student@lab411.cz",
    "password": "SecretPassword123"
  }
  ```
  *(Note: You can pass `"username"` instead of `"email"`)*

- **Response `200 OK`**:
  ```json
  {
    "message": "Login successful",
    "token": "d741cbef-81a1-4328-9177-3bc8c8699b0c",
    "user": "student@lab411.cz",
    "role": "student",
    "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  }
  ```

- **Errors**:
  - `400 Bad Request`: `{"error": "username and password required"}`
  - `401 Unauthorized`: `{"error": "invalid username or password"}`

---

### GET /ping

Validates the current session token and returns the caller's UUID, email, and role.

- **Access**: Authenticated
- **Headers**: `Authorization: Bearer <token>`
- **Response `200 OK`**:
  ```json
  {
    "message": "pong",
    "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "role": "student",
    "email": "student@lab411.cz"
  }
  ```
- **Response `401 Unauthorized`**: If token is missing, invalid, or expired.

---

## 2. Virtual Machines

### GET /vms

Lists virtual machines. For students, returns only VMs tagged with `owner-<uuid>`. For teachers and admins, returns all cluster VMs.

- **Access**: Authenticated
- **Response `200 OK`**:
  ```json
  [
    {
      "id": 105,
      "uuid": "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
      "type": "qemu",
      "name": "kali-workstation",
      "status": "Running",
      "ram_mb": 4096,
      "node": "pve1",
      "owner_uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "owner_email": "student@lab411.cz",
      "project_id": "671cb0a84d123e4567890abc",
      "project_name": "Penetration Testing",
      "project_color": "#ff5722",
      "ip_address": "192.168.10.15",
      "configured_ip": "192.168.10.15/24",
      "gateway": "192.168.10.1",
      "networks": ["net0: 100", "net1: 105"],
      "available_nics": ["net2", "net3"],
      "connections": [
        {
          "target_vm_id": "106",
          "local_nic": "net1",
          "target_nic": "net1"
        }
      ],
      "has_internet": true,
      "is_attacker": true,
      "is_vulnerable": false
    }
  ]
  ```

---

### GET /vms/templates

Lists base Proxmox virtual machines that are marked as templates (`Template: true`), which can be cloned.

- **Access**: Authenticated
- **Response `200 OK`**:
  ```json
  [
    {
      "id": 100,
      "name": "template-ubuntu-24.04",
      "ram_mb": 2048,
      "node": "pve1",
      "status": "Template"
    }
  ]
  ```

---

### POST /vms/create

Provisions a linked clone from an existing template. Tags the VM with `owner-<uuid>` and `vm-<uuid>`, injects `serial0=socket` for console support, and optionally links the VM to a project.

- **Access**: Authenticated
- **Request Body**:
  ```json
  {
    "name": "my-kali-vm",
    "template_id": 100,
    "node": "pve1",
    "project_id": "671cb0a84d123e4567890abc"
  }
  ```
  *(Note: `template_id` can be an integer like `100`, string `"100"`, or node-prefixed string `"pve1:100"`)*

- **Response `200 OK`**:
  ```json
  {
    "uuid": "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
    "vmid": 105,
    "project_id": "671cb0a84d123e4567890abc"
  }
  ```

---

### POST /vms/start

Powers on a virtual machine. Validates caller ownership.

- **Access**: Authenticated
- **Request Body**:
  ```json
  {
    "id": 105,
    "node": "pve1"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "status": "started",
    "id": 105
  }
  ```

---

### POST /vms/stop

Powers off / stops a virtual machine. Validates caller ownership.

- **Access**: Authenticated
- **Request Body**:
  ```json
  {
    "id": 105,
    "node": "pve1"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "status": "stopped",
    "id": 105
  }
  ```

---

### POST /vms/delete

Destroys a virtual machine permanently and cleans up any project bindings.

- **Access**: Authenticated
- **Request Body**:
  ```json
  {
    "id": 105,
    "node": "pve1"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "status": "deleted",
    "id": 105
  }
  ```

---

### POST /vms/ip

Configures static IP CIDR and optional default gateway or switches the interface to DHCP.

- **Access**: Authenticated
- **Request Body**:
  ```json
  {
    "vmid": 105,
    "nic": "net0",
    "ip": "192.168.10.15/24",
    "gateway": "192.168.10.1",
    "is_dhcp": false,
    "node": "pve1"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "message": "IP adresa byla úspěšně nastavena",
    "vmid": 105
  }
  ```

---

## 3. VM Networking & 802.1Q VLANs

### POST /vms/network/add-nic

Appends the next available VirtIO network interface (`net0` through `net31`) on bridge `vmbr0` without VLAN tag.

- **Access**: Authenticated
- **Request Body**: `{"vmid": 105}`
- **Response `200 OK`**:
  ```json
  {
    "status": "added",
    "nic": "net1"
  }
  ```

---

### POST /vms/network/remove & POST /vms/network/delete

Detaches and removes a network interface from the virtual machine.

- **Access**: Authenticated
- **Request Body**:
  ```json
  {
    "vmid": 105,
    "nic": "net1"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "status": "removed",
    "nic": "net1"
  }
  ```

---

### POST /vms/network/create

Creates an isolated point-to-point or multi-point virtual network interconnect between two virtual machines using a dynamically assigned 802.1Q VLAN tag (100–4094).

- **Access**: Authenticated
- **Request Body**:
  ```json
  {
    "vmid1": 105,
    "vmid2": 106,
    "nic1": "net1",
    "nic2": "net1"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "status": "created"
  }
  ```

---

### POST /vms/network/disconnect

Removes the VLAN interconnect between two virtual machines.

- **Access**: Authenticated
- **Request Body**:
  ```json
  {
    "vmid1": 105,
    "vmid2": 106,
    "nic": "net1"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "status": "disconnected"
  }
  ```

---

### POST /vms/network/lab-connect

Attaches the VM to the shared lab environment network VLAN.

- **Access**: Authenticated
- **Request Body**: `{"vmid": 105}`
- **Response `200 OK`**:
  ```json
  {
    "message": "machine successfully connected to lab network"
  }
  ```

---

### POST /vms/network/lab-disconnect

Detaches the VM from the shared lab network VLAN.

- **Access**: Authenticated
- **Request Body**: `{"vmid": 105}`
- **Response `200 OK`**:
  ```json
  {
    "message": "machine successfully disconnected from lab network"
  }
  ```

---

## 4. Remote Consoles & Clipboard

### WS /vms/console/ws

Upgrades HTTP to WebSocket and establishes an interactive bi-directional serial terminal stream connecting to Proxmox `termproxy` for `serial0`.

- **Protocol**: `ws://` or `wss://`
- **Query Parameters**:
  - `vmid` (integer, required): VMID of the machine.
  - `token` (string, required): Active session token.
  - `node` (string, optional): Target Proxmox node name.
- **Frontend Integration Example (xterm.js)**:
  ```javascript
  const socket = new WebSocket(`ws://localhost:8080/vms/console/ws?vmid=105&token=${sessionToken}`);
  socket.binaryType = "arraybuffer";
  const term = new Terminal();
  term.open(document.getElementById('terminal'));

  socket.onmessage = (event) => term.write(new Uint8Array(event.data));
  term.onData((data) => socket.send(data));
  ```

---

### WS /vms/console/vnc

Upgrades HTTP to WebSocket and streams raw binary RFB protocol frames from Proxmox `vncwebsocket`. Supports in-browser noVNC display.

- **Protocol**: `ws://` or `wss://`
- **Subprotocol**: `binary` (`Sec-WebSocket-Protocol: binary`)
- **Query Parameters**:
  - `vmid` (integer, required): VMID of the machine.
  - `token` (string, required): Active session token.
  - `node` (string, optional): Target Proxmox node name.

---

### POST /vms/console/external

Starts a local TCP proxy port listener on the backend that proxies external VNC traffic directly to the VM's Proxmox VNC socket. Allows connecting via desktop VNC viewers (TigerVNC, RealVNC, Remmina).

- **Access**: Authenticated
- **Request Body**:
  ```json
  {
    "vmid": 105,
    "node": "pve1"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "vmid": 105,
    "host": "backend.lab411.cz",
    "port": 5901,
    "display": 1,
    "command": "vncviewer backend.lab411.cz:5901",
    "vnc_url": "vnc://backend.lab411.cz:5901",
    "file_url": "/vms/console/external/file?vmid=105",
    "created_at": "2026-11-03T12:00:00Z",
    "active_conn": 1
  }
  ```

---

### GET /vms/console/external/file

Downloads a `.vnc` connection configuration file pointing to the running external VNC proxy port for 1-click desktop launching.

- **Access**: Authenticated
- **Query Parameter**: `vmid=105`
- **Response `200 OK`**: Content-Type `application/x-vnc` (attachment download)

---

### DELETE /vms/console/external/:vmid

Stops and shuts down the external TCP VNC proxy listener for the specified VMID.

- **Access**: Authenticated
- **Path Parameter**: `vmid`
- **Response `200 OK`**:
  ```json
  {
    "message": "VNC proxy stopped",
    "vmid": 105
  }
  ```

---

### GET /vms/console/clipboard

Reads the current clipboard text from the guest OS via QEMU Guest Agent.

- **Access**: Authenticated
- **Query Parameter**: `vmid=105`
- **Response `200 OK`**:
  ```json
  {
    "text": "Extracted clipboard content from guest OS"
  }
  ```

---

### POST /vms/console/clipboard

Injects text into the VM guest OS clipboard via QEMU Guest Agent.

- **Access**: Authenticated
- **Request Body**:
  ```json
  {
    "vmid": 105,
    "text": "Text to inject into guest clipboard",
    "node": "pve1"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "message": "Clipboard updated successfully"
  }
  ```

---

## 5. Appliance Catalog

### GET /catalog

Lists pre-configured VM appliances, including clean operating systems and security vulnerability practice targets (e.g., PHP backdoor CVE, DVWA, Metasploitable).

- **Access**: Authenticated
- **Response `200 OK`**:
  ```json
  [
    {
      "id": "cve-php-8.1.0-dev",
      "name": "PHP 8.1.0-dev RCE Target",
      "type": "vm",
      "category": "Web Vulnerability",
      "description": "Vulnerable PHP 8.1.0-dev backdoor target",
      "template_id": 100,
      "default_ram": 2048,
      "default_disk_gb": 20,
      "default_cpus": 2,
      "is_vulnerable": true,
      "ready": true,
      "cve_id": "CVE-2021-3677",
      "cvss": "9.8",
      "port": 80,
      "exploit_hint": "Check User-Agentt header",
      "exploit_doc": "Send zerodiumsystem('id'); in User-Agentt header"
    }
  ]
  ```

---

### POST /catalog/deploy

Deploys an appliance from the catalog with custom hardware configuration, optional static IP/gateway or DHCP, and optional project association.

- **Access**: Authenticated
- **Request Body**:
  ```json
  {
    "catalog_id": "cve-php-8.1.0-dev",
    "name": "php-rce-lab",
    "ram": 2048,
    "cores": 2,
    "disk": 20,
    "project_id": "671cb0a84d123e4567890abc",
    "ip": "192.168.10.20/24",
    "gateway": "192.168.10.1",
    "is_dhcp": false,
    "node": "pve1"
  }
  ```
- **Response `200 OK`**: Returns the deployed `VM` object.

---

## 6. Projects Management

### GET /projects

Lists projects owned by the authenticated user (or all projects if teacher/admin).

- **Access**: Authenticated
- **Response `200 OK`**:
  ```json
  [
    {
      "id": "671cb0a84d123e4567890abc",
      "name": "Cyber Defense Lab",
      "description": "Red vs Blue security playground",
      "color": "#ff5722",
      "owner_uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "owner_email": "student@lab411.cz",
      "vm_ids": [105, 106],
      "created_at": "2026-11-03T12:00:00Z",
      "updated_at": "2026-11-03T12:00:00Z"
    }
  ]
  ```

---

### POST /projects

Creates a new project.

- **Access**: Authenticated
- **Request Body**:
  ```json
  {
    "name": "Cyber Defense Lab",
    "description": "Red vs Blue security playground",
    "color": "#ff5722"
  }
  ```
- **Response `200 OK`**: Returns created `Project` object.

---

### PUT /projects/:id

Updates project name, description, or color.

- **Access**: Authenticated
- **Request Body**:
  ```json
  {
    "name": "Updated Lab Name",
    "description": "Updated description",
    "color": "#4caf50"
  }
  ```
- **Response `200 OK`**: Returns updated `Project` object.

---

### DELETE /projects/:id

Deletes a project and removes project assignments from all member VMs.

- **Access**: Authenticated
- **Response `200 OK`**: `{"message": "project deleted successfully"}`

---

### POST /projects/:id/vms

Assigns one or more virtual machines to the specified project.

- **Access**: Authenticated
- **Request Body**: `{"vm_id": 105}` or `{"vm_ids": [105, 106]}`
- **Response `200 OK`**: `{"message": "VM added to project"}`

---

### DELETE /projects/:id/vms/:vmid

Removes a virtual machine from a project.

- **Access**: Authenticated
- **Response `200 OK`**: `{"message": "VM removed from project"}`

---

### POST /vms/:id/project

Assigns a VM to a project by VMID.

- **Access**: Authenticated
- **Path Parameter**: `id` (VMID)
- **Request Body**: `{"project_id": "671cb0a84d123e4567890abc"}`
- **Response `200 OK`**: `{"message": "VM project assigned successfully"}`

---

## 7. Cyber Security Lab Blueprints

### GET /labs/templates

Returns all available multi-VM lab blueprints.

- **Access**: Authenticated
- **Response `200 OK`**:
  ```json
  [
    {
      "id": "671cb0a84d123e4567890abc",
      "name": "Web Application Penetration Testing",
      "category": "Cybersecurity",
      "difficulty": "Intermediate",
      "description": "Multi-node hands-on lab exploring web vulnerabilities",
      "objectives": [
        "Discover vulnerable web service",
        "Obtain reverse shell",
        "Escalate privileges to root"
      ],
      "vms": [
        {
          "name": "Kali-Attacker",
          "template_id": 100,
          "ram_mb": 4096,
          "default_user": "kali",
          "default_password": "kali"
        },
        {
          "name": "Target-Server",
          "template_id": 101,
          "ram_mb": 2048,
          "default_user": "ubuntu",
          "default_password": "ubuntu"
        }
      ],
      "networks": [
        {
          "from_vm": "Kali-Attacker",
          "from_nic": "net1",
          "to_vm": "Target-Server",
          "to_nic": "net1",
          "label": "Isolated Target VLAN"
        }
      ]
    }
  ]
  ```

---

### GET /labs/templates/:id

Returns a specific lab template blueprint by ID.

- **Access**: Authenticated
- **Response `200 OK`**: Returns `LabTemplate` object.

---

### POST /labs/templates

Creates a new multi-VM lab blueprint.

- **Access**: Teacher or Admin only
- **Request Body**: `LabTemplate` object
- **Response `201 Created`**: Returns created `LabTemplate` object.

---

### PUT /labs/templates/:id

Updates an existing lab template.

- **Access**: Teacher or Admin only
- **Request Body**: `LabTemplate` object
- **Response `200 OK`**: `{"status": "updated", "id": "..."}`

---

### DELETE /labs/templates/:id

Deletes a lab template.

- **Access**: Teacher or Admin only
- **Response `200 OK`**: `{"status": "deleted", "id": "..."}`

---

### POST /labs/deploy

Deploys an entire multi-VM lab topology. Clones all blueprint member VMs, dynamically provisions isolated 802.1Q VLAN interconnects, and tags all VMs with tenant ownership.

- **Access**: Authenticated (Teachers can also deploy to a specific `student_uuid`)
- **Request Body**:
  ```json
  {
    "template_id": "671cb0a84d123e4567890abc",
    "student_uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "lab_prefix": "team1"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "id": "671cb0a84d123e4567890abc",
    "template_id": "671cb0a84d123e4567890abc",
    "lab_name": "Web Application Penetration Testing",
    "owner_uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "owner_email": "student@lab411.cz",
    "status": "running",
    "vms": [
      {
        "template_vm_name": "Kali-Attacker",
        "vmid": 105,
        "name": "lab-team1-Kali-Attacker",
        "default_user": "kali",
        "default_password": "kali"
      },
      {
        "template_vm_name": "Target-Server",
        "vmid": 106,
        "name": "lab-team1-Target-Server",
        "default_user": "ubuntu",
        "default_password": "ubuntu"
      }
    ]
  }
  ```

---

### GET /labs/instances

Lists active deployed lab instances belonging to the caller (or all instances if teacher/admin).

- **Access**: Authenticated
- **Response `200 OK`**: Array of `LabInstance` objects.

---

### POST /labs/instances/:id/start

Powers on all virtual machines in the specified lab instance.

- **Access**: Authenticated
- **Response `200 OK`**: `{"status": "started", "id": "..."}`

---

### POST /labs/instances/:id/stop

Powers off all virtual machines in the specified lab instance.

- **Access**: Authenticated
- **Response `200 OK`**: `{"status": "stopped", "id": "..."}`

---

### DELETE /labs/instances/:id

Tears down the lab instance and permanently destroys all member VMs in Proxmox.

- **Access**: Authenticated
- **Response `200 OK`**: `{"status": "deleted", "id": "..."}`

---

## 8. Teacher & Classroom Administration

### GET /teacher/students

Lists all student accounts along with total VM count, running VM count, and active deployed lab count.

- **Access**: Teacher or Admin only
- **Response `200 OK`**:
  ```json
  [
    {
      "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "email": "student1@lab411.cz",
      "role": "student",
      "vm_count": 3,
      "running_vms": 1,
      "lab_count": 1
    }
  ]
  ```

---

### POST /teacher/students

Creates a new student or teacher account in MongoDB. Passwords are saved as SHA-256 hashes.

- **Access**: Teacher or Admin only
- **Request Body**:
  ```json
  {
    "email": "student2@lab411.cz",
    "password": "SecurePassword123",
    "role": "student"
  }
  ```
- **Response `201 Created`**:
  ```json
  {
    "uuid": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "email": "student2@lab411.cz",
    "role": "student"
  }
  ```

---

### DELETE /teacher/students/:id

Deletes a student account by UUID.

- **Access**: Teacher or Admin only
- **Response `200 OK`**: `{"status": "deleted", "uuid": "..."}`

---

### POST /teacher/students/:id/reset-password

Sets a new password for the specified student account.

- **Access**: Teacher or Admin only
- **Request Body**: `{"password": "NewSecretPassword123"}`
- **Response `200 OK`**: `{"status": "password_reset", "uuid": "..."}`

---

### GET /teacher/vms

Returns every virtual machine across all nodes in the cluster, regardless of owner.

- **Access**: Teacher or Admin only
- **Response `200 OK`**: Array of `VM` objects.

---

### POST /teacher/vms/bulk-action

Executes a bulk power command across all student VMs in the cluster (e.g. `stop_all` at the end of class).

- **Access**: Teacher or Admin only
- **Request Body**:
  ```json
  {
    "action": "stop_all"
  }
  ```
- **Response `200 OK`**: `{"status": "completed", "action": "stop_all"}`

---

### POST /vms/templates/convert

Converts an existing virtual machine into a Proxmox template.

- **Access**: Teacher or Admin only
- **Request Body**:
  ```json
  {
    "vmid": 105,
    "node": "pve1"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "message": "VM successfully converted to template",
    "template": { ... }
  }
  ```

---

### POST /vms/templates/clone-as-template

Clones a source VM and converts the newly cloned VM into a template.

- **Access**: Teacher or Admin only
- **Request Body**:
  ```json
  {
    "source_vmid": 105,
    "source_node": "pve1",
    "template_name": "ubuntu-24.04-custom",
    "clone_first": true,
    "target_node_id": "pve2",
    "description": "Baseline image with Wireshark and Metasploit pre-installed",
    "ram_mb": 4096
  }
  ```
- **Response `201 Created`**:
  ```json
  {
    "message": "Template successfully created from clone",
    "template": { ... }
  }
  ```

---

## 9. Cluster Nodes & Load Balancer

### GET /nodes

Returns Proxmox cluster nodes and basic telemetry (Teachers/Admins only; returns empty list for students).

- **Access**: Authenticated
- **Response `200 OK`**:
  ```json
  [
    {
      "name": "pve1",
      "host_uri": "https://10.0.10.2:8006",
      "status": "online",
      "version": "8.2.4",
      "total_vms": 12,
      "running_vms": 5,
      "templates": 4
    }
  ]
  ```

---

### GET /teacher/nodes

Returns real-time telemetry from the dynamic load balancer pool across all standalone and cluster nodes.

- **Access**: Teacher or Admin only
- **Response `200 OK`**:
  ```json
  [
    {
      "node_id": "pve1",
      "node_name": "pve1",
      "host_uri": "https://10.0.10.2:8006",
      "is_online": true,
      "total_ram_mb": 65536,
      "used_ram_mb": 24576,
      "free_ram_mb": 40960,
      "ram_usage_pct": 37.5,
      "cpu_usage_pct": 12.8,
      "active_vm_count": 5,
      "total_vm_count": 14,
      "templates": [100, 101, 102],
      "weight": 100,
      "is_active": true,
      "last_updated": "2026-11-03T12:00:00Z"
    }
  ]
  ```

---

### POST /teacher/nodes

Registers a standalone Proxmox node into the dynamic load balancing pool.

- **Access**: Teacher or Admin only
- **Request Body**:
  ```json
  {
    "name": "pve2",
    "host_uri": "https://10.0.10.4:8006",
    "username": "root@pam",
    "password": "ProxmoxPassword123",
    "weight": 100,
    "is_active": true
  }
  ```
- **Response `201 Created`**:
  ```json
  {
    "message": "Node successfully registered in balancing pool",
    "node": { ... }
  }
  ```

---

### DELETE /teacher/nodes/:id

Removes a node from the load balancing pool.

- **Access**: Teacher or Admin only
- **Path Parameter**: `id` (Node name or ID)
- **Response `200 OK`**: `{"message": "Node successfully removed from pool"}`

---

### POST /teacher/nodes/refresh

Forces an immediate telemetry refresh and template scan across all nodes in the balancing pool.

- **Access**: Teacher or Admin only
- **Response `200 OK`**:
  ```json
  {
    "message": "Telemetry refreshed",
    "telemetry": [ ... ]
  }
  ```

---

## Workflow Example (cURL Walkthrough)

Here is an end-to-end walkthrough illustrating common actions using `curl`:

### 1. Authenticate & Obtain Session Token

```bash
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "student@lab411.cz", "password": "SecretPassword123"}')

TOKEN=$(echo $LOGIN_RESPONSE | jq -r .token)
echo "Session token: $TOKEN"
```

### 2. Verify Session Validity

```bash
curl -s -X GET http://localhost:8080/ping \
  -H "Authorization: Bearer $TOKEN"
```

### 3. List Available Templates

```bash
curl -s -X GET http://localhost:8080/vms/templates \
  -H "Authorization: Bearer $TOKEN"
```

### 4. Clone a Virtual Machine

```bash
CREATE_RESPONSE=$(curl -s -X POST http://localhost:8080/vms/create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "my-kali", "template_id": 100}')

VMID=$(echo $CREATE_RESPONSE | jq -r .vmid)
echo "Created VMID: $VMID"
```

### 5. Add a Secondary Network Interface (NIC)

```bash
curl -s -X POST http://localhost:8080/vms/network/add-nic \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"vmid\": $VMID}"
```

### 6. Power On the Virtual Machine

```bash
curl -s -X POST http://localhost:8080/vms/start \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"id\": $VMID}"
```

### 7. Connect to Serial Console via WebSocket

Using `wscat`:
```bash
wscat -c "ws://localhost:8080/vms/console/ws?vmid=$VMID&token=$TOKEN"
```

### 8. Deploy a Complete Multi-VM Lab

```bash
curl -s -X POST http://localhost:8080/labs/deploy \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"template_id": "671cb0a84d123e4567890abc", "lab_prefix": "team1"}'
```
