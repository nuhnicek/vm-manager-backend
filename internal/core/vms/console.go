package vms

import (
	"context"
	"crypto/des"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"vm-manager-backend/types"

	"github.com/gorilla/websocket"
	"github.com/luthermonson/go-proxmox"
)

type ConsoleStream struct {
	Send  chan []byte
	Recv  chan []byte
	Errs  chan error
	Close func() error
}

func OpenVMConsole(conn types.Conn, userUUID string, vmid int) (*ConsoleStream, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	node, err := conn.Proxmox.Node(ctx, "pve")
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	vm, err := node.VirtualMachine(ctx, vmid)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM %d: %w", vmid, err)
	}

	if vm.Template {
		return nil, fmt.Errorf("cannot open console on template %d", vmid)
	}

	if !VerifyVMOwner(vm, userUUID) {
		return nil, fmt.Errorf("unauthorized: VM %d does not belong to user", vmid)
	}

	if !strings.EqualFold(vm.Status, "running") {
		return nil, fmt.Errorf("VM %d is stopped (please start it first)", vmid)
	}

	if vm.VirtualMachineConfig != nil && len(vm.VirtualMachineConfig.Serials) == 0 {
		_ = vm.ConfigSync(ctx, proxmox.VirtualMachineOption{
			Name:  "serial0",
			Value: "socket",
		})
		return nil, fmt.Errorf("serial console interface was missing for VM %d and has now been configured; please restart (stop and start) the VM to enable console access", vmid)
	}

	term, err := vm.TermProxy(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize termproxy: %w", err)
	}

	rawURL := os.Getenv("PROXMOX_URI")
	if rawURL == "" {
		rawURL = "https://10.0.1.37:8006/api2/json"
	}
	wsBase := strings.TrimRight(strings.Replace(rawURL, "https://", "wss://", 1), "/")
	wsBase = strings.Replace(wsBase, "http://", "ws://", 1)
	wsURL := fmt.Sprintf("%s/nodes/%s/qemu/%d/vncwebsocket?port=%v&vncticket=%s",
		wsBase, vm.Node, vm.VMID, term.Port, url.QueryEscape(term.Ticket))

	dialer := &websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
		Subprotocols:     []string{"binary"},
	}

	headers := http.Header{}
	if sess := conn.Proxmox.Session(); sess != nil {
		headers.Add("Cookie", "PVEAuthCookie="+sess.Ticket)
		headers.Add("CSRFPreventionToken", sess.CSRFPreventionToken)
	}

	pveWS, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to dial Proxmox terminal websocket: %w", err)
	}

	// Authenticate ticket with Proxmox
	authPayload := fmt.Sprintf("%s:%s\n", term.User, term.Ticket)
	if err := pveWS.WriteMessage(websocket.BinaryMessage, []byte(authPayload)); err != nil {
		_ = pveWS.Close()
		return nil, fmt.Errorf("failed to authenticate terminal ticket: %w", err)
	}

	// Proxmox sends "OK" upon successful authentication
	_, resp, err := pveWS.ReadMessage()
	if err != nil || string(resp) != "OK" {
		_ = pveWS.Close()
		return nil, fmt.Errorf("invalid terminal handshake response: %s", string(resp))
	}

	// Set initial terminal size (80 cols, 24 rows)
	_ = pveWS.WriteMessage(websocket.BinaryMessage, []byte("1:80:24:"))

	send := make(chan []byte, 128)
	recv := make(chan []byte, 128)
	errs := make(chan error, 16)

	done := make(chan struct{})
	var closeOnce sync.Once
	closeFn := func() error {
		closeOnce.Do(func() {
			close(done)
			_ = pveWS.Close()
		})
		return nil
	}

	// Keepalive ticker
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_ = pveWS.WriteMessage(websocket.BinaryMessage, []byte("2"))
			}
		}
	}()

	// Proxmox -> Client
	go func() {
		defer closeFn()
		for {
			select {
			case <-done:
				return
			default:
				_, msg, err := pveWS.ReadMessage()
				if err != nil {
					select {
					case <-done:
					default:
						select {
						case errs <- err:
						default:
						}
					}
					return
				}
				if len(msg) > 0 {
					select {
					case <-done:
						return
					case recv <- msg:
					}
				}
			}
		}
	}()

	// Client -> Proxmox
	go func() {
		defer closeFn()
		for {
			select {
			case <-done:
				return
			case msg, ok := <-send:
				if !ok {
					return
				}
				packet := append([]byte(fmt.Sprintf("0:%d:", len(msg))), msg...)
				if err := pveWS.WriteMessage(websocket.BinaryMessage, packet); err != nil {
					return
				}
			}
		}
	}()

	// Send initial newline to wake serial getty
	go func() {
		time.Sleep(100 * time.Millisecond)
		select {
		case send <- []byte("\n"):
		case <-done:
		}
	}()

	return &ConsoleStream{
		Send:  send,
		Recv:  recv,
		Errs:  errs,
		Close: closeFn,
	}, nil
}

// OpenVMVNC initializes a Proxmox VNC proxy session (RFB protocol)
// and opens a live bidirectional WebSocket stream to the VM's graphical screen.
// It transparently authenticates with Proxmox using the ephemeral VNC ticket,
// allowing the client (e.g. noVNC) to connect seamlessly without password prompts.
func OpenVMVNC(conn types.Conn, userUUID string, vmid int) (*ConsoleStream, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	node, err := conn.Proxmox.Node(ctx, "pve")
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	vm, err := node.VirtualMachine(ctx, vmid)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM %d: %w", vmid, err)
	}

	if vm.Template {
		return nil, fmt.Errorf("cannot open console on template %d", vmid)
	}

	if !VerifyVMOwner(vm, userUUID) {
		return nil, fmt.Errorf("unauthorized: VM %d does not belong to user", vmid)
	}

	if !strings.EqualFold(vm.Status, "running") {
		return nil, fmt.Errorf("VM %d is stopped (please start it first)", vmid)
	}

	vnc, err := vm.VNCProxy(ctx, &proxmox.VNCConfig{Websocket: true})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize VNC proxy: %w", err)
	}

	pveSend, pveRecv, pveErrs, pveCloser, err := vm.VNCWebSocket(vnc)
	if err != nil {
		return nil, fmt.Errorf("failed to open VNC websocket: %w", err)
	}

	clientSend := make(chan []byte, 256)
	clientRecv := make(chan []byte, 256)
	clientErrs := make(chan error, 16)

	done := make(chan struct{})
	var closeOnce sync.Once
	safeClose := func() error {
		closeOnce.Do(func() {
			close(done)
			_ = pveCloser()
		})
		return nil
	}

	// Goroutine: Perform transparent RFB handshake, then proxy data
	go func() {
		defer safeClose()

		// 1. Proxmox sends server RFB version (e.g. "RFB 003.008\n")
		var serverVersion []byte
		select {
		case <-done:
			return
		case err, ok := <-pveErrs:
			if ok && err != nil {
				select {
				case clientErrs <- err:
				case <-done:
				}
			}
			return
		case serverVersion = <-pveRecv:
		case <-time.After(10 * time.Second):
			select {
			case clientErrs <- fmt.Errorf("timeout waiting for Proxmox RFB version"):
			case <-done:
			}
			return
		}

		// Forward server version to client
		select {
		case <-done:
			return
		case clientRecv <- serverVersion:
		}

		// 2. Client sends client RFB version (e.g. "RFB 003.008\n")
		var clientVersion []byte
		select {
		case <-done:
			return
		case clientVersion = <-clientSend:
		case <-time.After(10 * time.Second):
			select {
			case clientErrs <- fmt.Errorf("timeout waiting for client RFB version"):
			case <-done:
			}
			return
		}

		// Forward client version to Proxmox
		select {
		case <-done:
			return
		case pveSend <- clientVersion:
		}

		// 3. Proxmox sends supported security types (e.g. [1, 2] = 1 type: VNCAuth)
		var pveSecTypes []byte
		select {
		case <-done:
			return
		case err, ok := <-pveErrs:
			if ok && err != nil {
				select {
				case clientErrs <- err:
				case <-done:
				}
			}
			return
		case pveSecTypes = <-pveRecv:
		case <-time.After(10 * time.Second):
			select {
			case clientErrs <- fmt.Errorf("timeout waiting for Proxmox security types"):
			case <-done:
			}
			return
		}

		// Check if Proxmox requires VNCAuth (type 2)
		requiresVNCAuth := false
		if len(pveSecTypes) >= 2 {
			count := int(pveSecTypes[0])
			for i := 1; i <= count && i < len(pveSecTypes); i++ {
				if pveSecTypes[i] == 2 {
					requiresVNCAuth = true
					break
				}
			}
		}

		if requiresVNCAuth {
			// Select security type 2 on Proxmox
			select {
			case <-done:
				return
			case pveSend <- []byte{2}:
			}

			// Proxmox sends 16-byte challenge
			var challenge []byte
			select {
			case <-done:
				return
			case err, ok := <-pveErrs:
				if ok && err != nil {
					select {
					case clientErrs <- err:
					case <-done:
					}
				}
				return
			case challenge = <-pveRecv:
			case <-time.After(10 * time.Second):
				select {
				case clientErrs <- fmt.Errorf("timeout waiting for Proxmox VNC challenge"):
				case <-done:
				}
				return
			}

			if len(challenge) != 16 {
				select {
				case clientErrs <- fmt.Errorf("invalid VNC challenge length %d", len(challenge)):
				case <-done:
				}
				return
			}

			// Encrypt challenge with vnc.Ticket using DES
			resp := rfbEncrypt(vnc.Ticket, challenge)
			select {
			case <-done:
				return
			case pveSend <- resp:
			}

			// Proxmox sends SecurityResult (4 bytes: 0 0 0 0 = OK)
			var pveSecResult []byte
			select {
			case <-done:
				return
			case err, ok := <-pveErrs:
				if ok && err != nil {
					select {
					case clientErrs <- err:
					case <-done:
					}
				}
				return
			case pveSecResult = <-pveRecv:
			case <-time.After(10 * time.Second):
				select {
				case clientErrs <- fmt.Errorf("timeout waiting for Proxmox security result"):
				case <-done:
				}
				return
			}

			if len(pveSecResult) < 4 || pveSecResult[3] != 0 {
				select {
				case clientErrs <- fmt.Errorf("Proxmox VNC authentication failed: %v", pveSecResult):
				case <-done:
				}
				return
			}

			// Tell client security type 1 (None) is supported
			select {
			case <-done:
				return
			case clientRecv <- []byte{1, 1}:
			}

			// Wait for client to select security type 1
			var clientChosen []byte
			select {
			case <-done:
				return
			case clientChosen = <-clientSend:
			case <-time.After(10 * time.Second):
				select {
				case clientErrs <- fmt.Errorf("timeout waiting for client security choice"):
				case <-done:
				}
				return
			}

			// In RFB 3.8, send SecurityResult OK (0 0 0 0) to client
			select {
			case <-done:
				return
			case clientRecv <- []byte{0, 0, 0, 0}:
			}

			// If client sent extra bytes beyond the 1-byte chosen type, forward them
			if len(clientChosen) > 1 {
				select {
				case <-done:
					return
				case pveSend <- clientChosen[1:]:
				}
			}
		} else {
			// Pass through if not VNCAuth
			select {
			case <-done:
				return
			case clientRecv <- pveSecTypes:
			}
		}

		// Handshake complete! Run bidirectional transparent proxy
		go func() {
			for {
				select {
				case <-done:
					return
				case msg, ok := <-clientSend:
					if !ok {
						safeClose()
						return
					}
					select {
					case <-done:
						return
					case pveSend <- msg:
					}
				}
			}
		}()

		for {
			select {
			case <-done:
				return
			case msg, ok := <-pveRecv:
				if !ok {
					return
				}
				select {
				case <-done:
					return
				case clientRecv <- msg:
				}
			case err, ok := <-pveErrs:
				if ok && err != nil {
					select {
					case <-done:
					case clientErrs <- err:
					}
				}
				return
			}
		}
	}()

	return &ConsoleStream{
		Send:  clientSend,
		Recv:  clientRecv,
		Errs:  clientErrs,
		Close: safeClose,
	}, nil
}

func rfbEncrypt(password string, challenge []byte) []byte {
	key := make([]byte, 8)
	for i := 0; i < len(password) && i < 8; i++ {
		var b byte = 0
		for j := 0; j < 8; j++ {
			if (password[i] & (1 << j)) != 0 {
				b |= (1 << (7 - j))
			}
		}
		key[i] = b
	}
	cipher, _ := des.NewCipher(key)
	resp := make([]byte, 16)
	cipher.Encrypt(resp[0:8], challenge[0:8])
	cipher.Encrypt(resp[8:16], challenge[8:16])
	return resp
}
