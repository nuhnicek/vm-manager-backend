package vms

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"
	"vm-manager-backend/types"
)

type ExternalVNCInfo struct {
	VMID       int       `json:"vmid"`
	Host       string    `json:"host"`
	Port       int       `json:"port"`
	Display    int       `json:"display"`
	Command    string    `json:"command"`
	VncURL     string    `json:"vnc_url"`
	FileURL    string    `json:"file_url"`
	CreatedAt  time.Time `json:"created_at"`
	ActiveConn int       `json:"active_conn"`
}

type vncProxySession struct {
	vmid       int
	port       int
	host       string
	targetNode string
	userUUID   string
	role       string
	conn       types.Conn
	listener   net.Listener
	stopChan   chan struct{}
	lastActive time.Time
	activeConn int
	mu         sync.Mutex
}

var (
	vncProxies   = make(map[int]*vncProxySession)
	vncProxiesMu sync.Mutex
)

func getBackendHost() string {
	if host := os.Getenv("BACKEND_IP"); host != "" {
		return host
	}
	return "10.0.10.52"
}

func StartExternalVNC(conn types.Conn, userUUID string, role string, vmid int, targetNode string) (*ExternalVNCInfo, error) {
	vncProxiesMu.Lock()
	defer vncProxiesMu.Unlock()

	if sess, exists := vncProxies[vmid]; exists {
		sess.mu.Lock()
		sess.lastActive = time.Now()
		info := &ExternalVNCInfo{
			VMID:       sess.vmid,
			Host:       sess.host,
			Port:       sess.port,
			Display:    sess.port - 5900,
			Command:    fmt.Sprintf("vncviewer %s:%d", sess.host, sess.port),
			VncURL:     fmt.Sprintf("vnc://%s:%d", sess.host, sess.port),
			FileURL:    fmt.Sprintf("/vms/console/external/file?vmid=%d", sess.vmid),
			CreatedAt:  sess.lastActive,
			ActiveConn: sess.activeConn,
		}
		sess.mu.Unlock()
		return info, nil
	}

	preferredPort := 5900 + (vmid % 100)
	if preferredPort < 5920 || preferredPort > 5999 {
		preferredPort = 5920 + (vmid % 70)
	}

	var listener net.Listener
	var actualPort int
	var err error

	listener, err = net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", preferredPort))
	if err == nil {
		actualPort = preferredPort
	} else {
		for p := 5920; p <= 5999; p++ {
			listener, err = net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", p))
			if err == nil {
				actualPort = p
				break
			}
		}
	}

	if listener == nil {
		return nil, fmt.Errorf("nepodařilo se přidělit volný TCP port pro VNC v rozsahu 5920-5999")
	}

	host := getBackendHost()
	stopChan := make(chan struct{})

	sess := &vncProxySession{
		vmid:       vmid,
		port:       actualPort,
		host:       host,
		targetNode: targetNode,
		userUUID:   userUUID,
		role:       role,
		conn:       conn,
		listener:   listener,
		stopChan:   stopChan,
		lastActive: time.Now(),
	}

	vncProxies[vmid] = sess

	go func() {
		defer func() {
			vncProxiesMu.Lock()
			delete(vncProxies, vmid)
			vncProxiesMu.Unlock()
			_ = listener.Close()
		}()

		for {
			tcpConn, err := listener.Accept()
			if err != nil {
				select {
				case <-stopChan:
					return
				default:
					return
				}
			}

			sess.mu.Lock()
			sess.activeConn++
			sess.lastActive = time.Now()
			sess.mu.Unlock()

			go func(c net.Conn) {
				defer func() {
					_ = c.Close()
					sess.mu.Lock()
					sess.activeConn--
					sess.lastActive = time.Now()
					sess.mu.Unlock()
				}()

				stream, err := OpenVMVNC(conn, userUUID, role, vmid, targetNode)
				if err != nil {
					return
				}
				defer stream.Close()

				bridgeDone := make(chan struct{})
				var once sync.Once
				closeBridge := func() {
					once.Do(func() {
						close(bridgeDone)
					})
				}

				go func() {
					defer closeBridge()
					buf := make([]byte, 65536)
					for {
						n, err := c.Read(buf)
						if err != nil {
							return
						}
						if n > 0 {
							packet := make([]byte, n)
							copy(packet, buf[:n])
							select {
							case <-bridgeDone:
								return
							case <-stopChan:
								return
							case stream.Send <- packet:
							}
						}
					}
				}()

				go func() {
					defer closeBridge()
					for {
						select {
						case <-bridgeDone:
							return
						case <-stopChan:
							return
						case err, ok := <-stream.Errs:
							if ok && err != nil {
								return
							}
						case msg, ok := <-stream.Recv:
							if !ok {
								return
							}
							if _, err := c.Write(msg); err != nil {
								return
							}
						}
					}
				}()

				<-bridgeDone
			}(tcpConn)
		}
	}()

	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				sess.mu.Lock()
				idle := sess.activeConn == 0 && time.Since(sess.lastActive) > 45*time.Minute
				sess.mu.Unlock()
				if idle {
					_ = StopExternalVNC(vmid)
					return
				}
			}
		}
	}()

	return &ExternalVNCInfo{
		VMID:       vmid,
		Host:       host,
		Port:       actualPort,
		Display:    actualPort - 5900,
		Command:    fmt.Sprintf("vncviewer %s:%d", host, actualPort),
		VncURL:     fmt.Sprintf("vnc://%s:%d", host, actualPort),
		FileURL:    fmt.Sprintf("/vms/console/external/file?vmid=%d", vmid),
		CreatedAt:  sess.lastActive,
		ActiveConn: 0,
	}, nil
}

func StopExternalVNC(vmid int) error {
	vncProxiesMu.Lock()
	defer vncProxiesMu.Unlock()

	if sess, exists := vncProxies[vmid]; exists {
		close(sess.stopChan)
		_ = sess.listener.Close()
		delete(vncProxies, vmid)
		return nil
	}
	return nil
}

func GenerateVNCProfile(vmid int) (string, string, error) {
	vncProxiesMu.Lock()
	sess, exists := vncProxies[vmid]
	vncProxiesMu.Unlock()

	host := getBackendHost()
	port := 5920 + (vmid % 70)
	if exists && sess != nil {
		host = sess.host
		port = sess.port
	}

	content := fmt.Sprintf("[Connection]\nHost=%s\nPort=%d\n[Options]\nUseLocalCursor=1\nFullScreen=0\nPreferredEncoding=Tight\nAutoReconnect=1\n", host, port)

	filename := fmt.Sprintf("vm-%d.vnc", vmid)
	return filename, content, nil
}
