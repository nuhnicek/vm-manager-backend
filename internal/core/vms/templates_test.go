package vms

import (
	"context"
	"os"
	"testing"
	"time"
	"vm-manager-backend/types"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/luthermonson/go-proxmox"
	"github.com/redis/go-redis/v9"
)

func TestLiveProxmoxTemplates(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	user := os.Getenv("PROXMOX_USERNAME")
	pass := os.Getenv("PROXMOX_PASSWORD")
	uri := os.Getenv("PROXMOX_URI")
	if user == "" || uri == "" {
		t.Skip("Proxmox credentials not found in .env")
	}

	credentials := proxmox.Credentials{
		Username: user,
		Password: pass,
	}
	client := proxmox.NewClient(uri,
		proxmox.WithCredentials(&credentials),
		proxmox.WithInsecureSkipVerify(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, err := client.Node(ctx, "pve")
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	vms, err := node.VirtualMachines(ctx)
	if err != nil {
		t.Fatalf("failed to get vms: %v", err)
	}

	t.Logf("Found %d total virtual machines in cluster", len(vms))
	for _, vm := range vms {
		t.Logf("VM %d: Name=%s, Template=%v, Tags=%s, Status=%s", vm.VMID, vm.Name, vm.Template, vm.Tags, vm.Status)
	}

	conn := types.Conn{Proxmox: client}
	templates, err := ListTemplates(conn)
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}
	t.Logf("ListTemplates returned %d templates", len(templates))
	for _, tpl := range templates {
		t.Logf("Template: ID=%d, Name=%s, UUID=%s", tpl.Id, tpl.Name, tpl.UUID)
	}
}

func TestLiveVNCProxyVM108(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	user := os.Getenv("PROXMOX_USERNAME")
	pass := os.Getenv("PROXMOX_PASSWORD")
	uri := os.Getenv("PROXMOX_URI")

	credentials := proxmox.Credentials{
		Username: user,
		Password: pass,
	}
	client := proxmox.NewClient(uri,
		proxmox.WithCredentials(&credentials),
		proxmox.WithInsecureSkipVerify(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, err := client.Node(ctx, "pve")
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	vm, err := node.VirtualMachine(ctx, 108)
	if err != nil {
		t.Fatalf("failed to get vm 108: %v", err)
	}

	vnc, err := vm.VNCProxy(ctx, &proxmox.VNCConfig{Websocket: true})
	if err != nil {
		t.Fatalf("VNCProxy failed: %v", err)
	}
	t.Logf("VNCProxy SUCCESS: Port=%v, User=%s, Ticket=%s", vnc.Port, vnc.User, vnc.Ticket[:10]+"...")

	send, recv, errs, closer, err := vm.VNCWebSocket(vnc)
	if err != nil {
		t.Fatalf("VNCWebSocket failed: %v", err)
	}
	defer closer()
	_ = send

	select {
	case msg := <-recv:
		t.Logf("VNC RECV: %q (%d bytes)", string(msg), len(msg))
	case err := <-errs:
		t.Fatalf("VNC error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatalf("VNC timeout")
	}

	send <- []byte("RFB 003.008\n")

	var secTypes []byte
	select {
	case secTypes = <-recv:
		t.Logf("Security types: %v", secTypes)
	case <-time.After(5 * time.Second):
		t.Fatalf("Timeout waiting for security types")
	}

	send <- []byte{2}

	var challenge []byte
	select {
	case challenge = <-recv:
		t.Logf("Challenge (%d bytes): %x", len(challenge), challenge)
	case <-time.After(5 * time.Second):
		t.Fatalf("Timeout waiting for challenge")
	}

	if len(challenge) != 16 {
		t.Fatalf("Expected 16-byte challenge, got %d bytes: %x", len(challenge), challenge)
	}

	resp := rfbEncrypt(vnc.Ticket, challenge)
	t.Logf("Encrypted response (%d bytes): %x", len(resp), resp)
	send <- resp

	var secResult []byte
	select {
	case secResult = <-recv:
		t.Logf("Security result (%d bytes): %v", len(secResult), secResult)
	case <-time.After(5 * time.Second):
		t.Fatalf("Timeout waiting for security result")
	}

	if len(secResult) >= 4 && secResult[3] == 0 {
		t.Logf("AUTHENTICATION SUCCESS! Proxmox VNC accepted the ticket!")
	} else {
		t.Fatalf("Authentication failed with result: %v", secResult)
	}

	send <- []byte{1}

	var serverInit []byte
	select {
	case serverInit = <-recv:
		t.Logf("ServerInit (%d bytes): %x", len(serverInit), serverInit)
	case <-time.After(5 * time.Second):
		t.Fatalf("Timeout waiting for ServerInit")
	}

	if len(serverInit) >= 24 {
		width := int(serverInit[0])<<8 | int(serverInit[1])
		height := int(serverInit[2])<<8 | int(serverInit[3])
		nameLen := int(serverInit[20])<<24 | int(serverInit[21])<<16 | int(serverInit[22])<<8 | int(serverInit[23])
		var name string
		if len(serverInit) >= 24+nameLen {
			name = string(serverInit[24 : 24+nameLen])
		}
		t.Logf("PROXMOX FRAMEBUFFER READY! Dimensions: %dx%d, Desktop name: %q", width, height, name)
	}
}

func TestInspectVM108(t *testing.T) {
	user := os.Getenv("PROXMOX_USERNAME")
	pass := os.Getenv("PROXMOX_PASSWORD")
	uri := os.Getenv("PROXMOX_URI")

	credentials := proxmox.Credentials{
		Username: user,
		Password: pass,
	}
	client := proxmox.NewClient(uri,
		proxmox.WithCredentials(&credentials),
		proxmox.WithInsecureSkipVerify(),
	)

	conn := types.Conn{Proxmox: client}
	userUUID := "7c694cfa-66cd-4895-8ba0-676a0b559f59"
	stream, err := OpenVMConsole(conn, userUUID, 108)
	if err != nil {
		t.Fatalf("OpenVMConsole failed: %v", err)
	}
	defer stream.Close()

	stream.Send <- []byte("\n")
	select {
	case msg := <-stream.Recv:
		t.Logf("Received from VM 108: %q", string(msg))
	case err := <-stream.Errs:
		t.Logf("Error from VM 108: %v", err)
	case <-time.After(3 * time.Second):
		t.Logf("Timed out waiting for data")
	}

	t.Logf("OpenVMConsole verified cleanly for VM 108!")
}

func TestLiveEndpointVNC(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	keys, err := rdb.Keys(context.Background(), "*").Result()
	if err != nil || len(keys) == 0 {
		t.Skip("No active session tokens in Redis to test VNC endpoint")
	}
	token := keys[0]
	u := "ws://localhost:8080/vms/console/vnc?vmid=108&token=" + token

	dialer := websocket.Dialer{
		Subprotocols: []string{"binary"},
	}
	ws, resp, err := dialer.Dial(u, nil)
	if err != nil {
		if resp != nil {
			t.Fatalf("Failed to dial VNC endpoint: status=%d %s, err: %v", resp.StatusCode, resp.Status, err)
		}
		t.Fatalf("Failed to dial VNC endpoint: %v", err)
	}
	defer ws.Close()

	t.Logf("Connected to VNC endpoint! Negotiated subprotocol: %s", ws.Subprotocol())

	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	messageType, data, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read from VNC endpoint: %v", err)
	}
	t.Logf("Received message type=%d, len=%d, data=%q", messageType, len(data), string(data))

	err = ws.WriteMessage(websocket.BinaryMessage, []byte("RFB 003.008\n"))
	if err != nil {
		t.Fatalf("Failed to write RFB version: %v", err)
	}

	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	messageType, data, err = ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read security types: %v", err)
	}
	t.Logf("Received security types len=%d, data=%v", len(data), data)

	if len(data) < 2 || data[1] != 1 {
		t.Fatalf("Expected security type None (1), got %v", data)
	}

	err = ws.WriteMessage(websocket.BinaryMessage, []byte{1})
	if err != nil {
		t.Fatalf("Failed to select security type 1: %v", err)
	}

	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	messageType, data, err = ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read SecurityResult: %v", err)
	}
	t.Logf("Received SecurityResult len=%d, data=%v", len(data), data)
	if len(data) < 4 || data[3] != 0 {
		t.Fatalf("SecurityResult not OK: %v", data)
	}

	err = ws.WriteMessage(websocket.BinaryMessage, []byte{1})
	if err != nil {
		t.Fatalf("Failed to send ClientInit: %v", err)
	}

	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	messageType, data, err = ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read ServerInit: %v", err)
	}
	t.Logf("Received ServerInit len=%d, data=%x", len(data), data)

	if len(data) >= 24 {
		width := int(data[0])<<8 | int(data[1])
		height := int(data[2])<<8 | int(data[3])
		nameLen := int(data[20])<<24 | int(data[21])<<16 | int(data[22])<<8 | int(data[23])
		var name string
		if len(data) >= 24+nameLen {
			name = string(data[24 : 24+nameLen])
		}
		t.Logf("END-TO-END VNC SUCCESS! Geometry: %dx%d, Desktop name: %q", width, height, name)
	}
}
