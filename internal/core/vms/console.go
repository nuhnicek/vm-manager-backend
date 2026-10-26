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

func OpenVMConsole(conn types.Conn, userUUID string, role string, vmid int, targetNodes ...string) (*ConsoleStream, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var term *proxmox.Term
	var wsURL string
	var clientForSession *proxmox.Client

	vm, nodeClient, err := conn.FindVM(ctx, vmid, targetNodes...)
	if err == nil && vm != nil {
		if vm.Template {
			return nil, fmt.Errorf("cannot open console on template %d", vmid)
		}

		if !VerifyVMOwner(vm, userUUID, role) {
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

		t, err := vm.TermProxy(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize termproxy: %w", err)
		}
		term = t

		rawURL := ""
		if nodeClient != nil && nodeClient.HostURI != "" {
			rawURL = nodeClient.HostURI
		} else {
			rawURL = os.Getenv("PROXMOX_URI")
			if idx := strings.Index(rawURL, ","); idx != -1 {
				rawURL = rawURL[:idx]
			}
		}
		wsBase := strings.TrimRight(strings.Replace(rawURL, "https://", "wss://", 1), "/")
		wsBase = strings.Replace(wsBase, "http://", "ws://", 1)
		wsURL = fmt.Sprintf("%s/nodes/%s/qemu/%d/vncwebsocket?port=%v&vncticket=%s",
			wsBase, vm.Node, vm.VMID, term.Port, url.QueryEscape(term.Ticket))

		if nodeClient != nil && nodeClient.Client != nil {
			clientForSession = nodeClient.Client
		} else {
			clientForSession = conn.Proxmox
		}
	} else {
		ct, ctNodeClient, ctErr := conn.FindContainer(ctx, vmid, targetNodes...)
		if ctErr != nil || ct == nil {
			return nil, fmt.Errorf("machine %d not found in pool", vmid)
		}

		if !VerifyContainerOwner(ct, userUUID, role) {
			return nil, fmt.Errorf("unauthorized: container %d does not belong to user", vmid)
		}

		if !strings.EqualFold(ct.Status, "running") {
			return nil, fmt.Errorf("container %d is stopped (please start it first)", vmid)
		}

		t, err := ct.TermProxy(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize termproxy: %w", err)
		}
		term = t

		rawURL := ""
		if ctNodeClient != nil && ctNodeClient.HostURI != "" {
			rawURL = ctNodeClient.HostURI
		} else {
			rawURL = os.Getenv("PROXMOX_URI")
			if idx := strings.Index(rawURL, ","); idx != -1 {
				rawURL = rawURL[:idx]
			}
		}
		wsBase := strings.TrimRight(strings.Replace(rawURL, "https://", "wss://", 1), "/")
		wsBase = strings.Replace(wsBase, "http://", "ws://", 1)
		wsURL = fmt.Sprintf("%s/nodes/%s/lxc/%d/vncwebsocket?port=%v&vncticket=%s",
			wsBase, ct.Node, ct.VMID, term.Port, url.QueryEscape(term.Ticket))

		if ctNodeClient != nil && ctNodeClient.Client != nil {
			clientForSession = ctNodeClient.Client
		} else {
			clientForSession = conn.Proxmox
		}
	}

	dialer := &websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
		Subprotocols:     []string{"binary"},
	}

	headers := http.Header{}
	if clientForSession != nil {
		if sess := clientForSession.Session(); sess != nil {
			headers.Add("Cookie", "PVEAuthCookie="+sess.Ticket)
			headers.Add("CSRFPreventionToken", sess.CSRFPreventionToken)
		}
	}

	pveWS, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to dial Proxmox terminal websocket: %w", err)
	}

	authPayload := fmt.Sprintf("%s:%s\n", term.User, term.Ticket)
	if err := pveWS.WriteMessage(websocket.BinaryMessage, []byte(authPayload)); err != nil {
		_ = pveWS.Close()
		return nil, fmt.Errorf("failed to authenticate terminal ticket: %w", err)
	}

	_, resp, err := pveWS.ReadMessage()
	if err != nil || string(resp) != "OK" {
		_ = pveWS.Close()
		return nil, fmt.Errorf("invalid terminal handshake response: %s", string(resp))
	}

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

func OpenVMVNC(conn types.Conn, userUUID string, role string, vmid int, targetNodes ...string) (*ConsoleStream, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var pveSend chan []byte
	var pveRecv chan []byte
	var pveErrs chan error
	var pveCloser func() error
	var vncTicket string

	vm, _, err := conn.FindVM(ctx, vmid, targetNodes...)
	if err == nil && vm != nil {
		if vm.Template {
			return nil, fmt.Errorf("cannot open console on template %d", vmid)
		}

		if !VerifyVMOwner(vm, userUUID, role) {
			return nil, fmt.Errorf("unauthorized: VM %d does not belong to user", vmid)
		}

		if !strings.EqualFold(vm.Status, "running") {
			return nil, fmt.Errorf("VM %d is stopped (please start it first)", vmid)
		}

		vnc, err := vm.VNCProxy(ctx, &proxmox.VNCConfig{Websocket: true})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize VNC proxy: %w", err)
		}
		vncTicket = vnc.Ticket

		pveSend, pveRecv, pveErrs, pveCloser, err = vm.VNCWebSocket(vnc)
		if err != nil {
			return nil, fmt.Errorf("failed to open VNC websocket: %w", err)
		}
	} else {
		ct, _, ctErr := conn.FindContainer(ctx, vmid, targetNodes...)
		if ctErr != nil || ct == nil {
			return nil, fmt.Errorf("machine %d not found in pool", vmid)
		}

		if !VerifyContainerOwner(ct, userUUID, role) {
			return nil, fmt.Errorf("unauthorized: container %d does not belong to user", vmid)
		}

		if !strings.EqualFold(ct.Status, "running") {
			return nil, fmt.Errorf("container %d is stopped (please start it first)", vmid)
		}

		vnc, err := ct.VNCProxy(ctx, proxmox.VNCProxyOptions{Websocket: "1"})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize VNC proxy: %w", err)
		}
		vncTicket = vnc.Ticket

		pveSend, pveRecv, pveErrs, pveCloser, err = ct.VNCWebSocket(vnc)
		if err != nil {
			return nil, fmt.Errorf("failed to open VNC websocket: %w", err)
		}
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

	go func() {
		defer safeClose()

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

		select {
		case <-done:
			return
		case clientRecv <- serverVersion:
		}

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

		select {
		case <-done:
			return
		case pveSend <- clientVersion:
		}

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
			select {
			case <-done:
				return
			case pveSend <- []byte{2}:
			}

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

			resp := rfbEncrypt(vncTicket, challenge)
			select {
			case <-done:
				return
			case pveSend <- resp:
			}

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

			select {
			case <-done:
				return
			case clientRecv <- []byte{1, 1}:
			}

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

			select {
			case <-done:
				return
			case clientRecv <- []byte{0, 0, 0, 0}:
			}

			if len(clientChosen) > 1 {
				select {
				case <-done:
					return
				case pveSend <- clientChosen[1:]:
				}
			}
		} else {
			select {
			case <-done:
				return
			case clientRecv <- pveSecTypes:
			}
		}

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

func StreamVMVNC(conn types.Conn, userUUID string, role string, vmid int, clientWS *websocket.Conn, targetNodes ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	vm, _, err := conn.FindVM(ctx, vmid, targetNodes...)
	if err != nil {
		return fmt.Errorf("failed to get VM %d: %w", vmid, err)
	}

	if vm.Template {
		return fmt.Errorf("cannot open console on template %d", vmid)
	}

	if !VerifyVMOwner(vm, userUUID, role) {
		return fmt.Errorf("unauthorized: VM %d does not belong to user", vmid)
	}

	if !strings.EqualFold(vm.Status, "running") {
		return fmt.Errorf("VM %d is stopped (please start it first)", vmid)
	}

	vnc, err := vm.VNCProxy(ctx, &proxmox.VNCConfig{Websocket: true})
	if err != nil {
		return fmt.Errorf("failed to initialize VNC proxy: %w", err)
	}

	pveSend, pveRecv, pveErrs, pveCloser, err := vm.VNCWebSocket(vnc)
	if err != nil {
		return fmt.Errorf("failed to open VNC websocket: %w", err)
	}

	done := make(chan struct{})
	var closeOnce sync.Once
	safeClose := func() {
		closeOnce.Do(func() {
			close(done)
			_ = pveCloser()
		})
	}
	defer safeClose()

	var serverVersion []byte
	select {
	case <-done:
		return nil
	case err, ok := <-pveErrs:
		if ok && err != nil {
			return err
		}
		return fmt.Errorf("proxmox connection closed prematurely")
	case serverVersion = <-pveRecv:
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for Proxmox RFB version")
	}

	if err := clientWS.WriteMessage(websocket.BinaryMessage, serverVersion); err != nil {
		return err
	}

	_ = clientWS.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, clientVersion, err := clientWS.ReadMessage()
	if err != nil {
		return fmt.Errorf("reading client RFB version: %w", err)
	}

	select {
	case <-done:
		return nil
	case pveSend <- clientVersion:
	}

	var pveSecTypes []byte
	select {
	case <-done:
		return nil
	case err, ok := <-pveErrs:
		if ok && err != nil {
			return err
		}
		return fmt.Errorf("proxmox connection closed prematurely")
	case pveSecTypes = <-pveRecv:
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for Proxmox security types")
	}

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
		select {
		case <-done:
			return nil
		case pveSend <- []byte{2}:
		}

		var challenge []byte
		select {
		case <-done:
			return nil
		case err, ok := <-pveErrs:
			if ok && err != nil {
				return err
			}
			return fmt.Errorf("proxmox connection closed prematurely")
		case challenge = <-pveRecv:
		case <-time.After(10 * time.Second):
			return fmt.Errorf("timeout waiting for Proxmox VNC challenge")
		}

		if len(challenge) != 16 {
			return fmt.Errorf("invalid VNC challenge length %d", len(challenge))
		}

		resp := rfbEncrypt(vnc.Ticket, challenge)
		select {
		case <-done:
			return nil
		case pveSend <- resp:
		}

		var pveSecResult []byte
		select {
		case <-done:
			return nil
		case err, ok := <-pveErrs:
			if ok && err != nil {
				return err
			}
			return fmt.Errorf("proxmox connection closed prematurely")
		case pveSecResult = <-pveRecv:
		case <-time.After(10 * time.Second):
			return fmt.Errorf("timeout waiting for Proxmox security result")
		}

		if len(pveSecResult) < 4 || pveSecResult[3] != 0 {
			return fmt.Errorf("Proxmox VNC authentication failed: %v", pveSecResult)
		}

		if err := clientWS.WriteMessage(websocket.BinaryMessage, []byte{1, 1}); err != nil {
			return err
		}

		_ = clientWS.SetReadDeadline(time.Now().Add(10 * time.Second))
		_, clientChosen, err := clientWS.ReadMessage()
		if err != nil {
			return fmt.Errorf("reading client security choice: %w", err)
		}

		if err := clientWS.WriteMessage(websocket.BinaryMessage, []byte{0, 0, 0, 0}); err != nil {
			return err
		}

		if len(clientChosen) > 1 {
			select {
			case <-done:
				return nil
			case pveSend <- clientChosen[1:]:
			}
		}
	} else {
		if err := clientWS.WriteMessage(websocket.BinaryMessage, pveSecTypes); err != nil {
			return err
		}
	}

	_ = clientWS.SetReadDeadline(time.Time{})

	go func() {
		defer safeClose()
		for {
			_, p, err := clientWS.ReadMessage()
			if err != nil {
				return
			}
			if len(p) > 0 {
				select {
				case <-done:
					return
				case pveSend <- p:
				}
			}
		}
	}()

	for {
		select {
		case <-done:
			return nil
		case msg, ok := <-pveRecv:
			if !ok {
				return nil
			}
			if err := clientWS.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				return err
			}
		case err, ok := <-pveErrs:
			if ok && err != nil {
				return err
			}
			return nil
		}
	}
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
