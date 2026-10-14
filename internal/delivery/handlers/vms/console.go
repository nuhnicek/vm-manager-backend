package vms

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow cross-origin WebSocket connections from frontend/wails
	},
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	Subprotocols:    []string{"binary"},
}

func ConsoleWS(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuidVal, exists := c.Get("uuid")
		if !exists {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		userUUID := uuidVal.(string)

		vmidStr := c.Query("vmid")
		if vmidStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "vmid query parameter is required"})
			return
		}

		vmid, err := strconv.Atoi(vmidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
			return
		}

		// Upgrade HTTP connection to WebSocket
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			fmt.Printf("failed to upgrade websocket: %v\n", err)
			return
		}
		defer ws.Close()

		stream, err := vms.OpenVMConsole(conn, userUUID, vmid)
		if err != nil {
			errMsg := fmt.Sprintf("\r\n\x1b[31m[Console Error: %v]\x1b[0m\r\n", err)
			_ = ws.WriteMessage(websocket.TextMessage, []byte(errMsg))
			_ = ws.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, err.Error()),
				time.Now().Add(time.Second))
			return
		}
		defer stream.Close()

		done := make(chan struct{})
		var once sync.Once
		closeDone := func() {
			once.Do(func() {
				close(done)
			})
		}

		// Goroutine: Proxmox terminal -> Client
		go func() {
			defer closeDone()
			for {
				select {
				case <-done:
					return
				case msg, ok := <-stream.Recv:
					if !ok {
						return
					}
					if err := ws.WriteMessage(websocket.BinaryMessage, msg); err != nil {
						return
					}
				case err, ok := <-stream.Errs:
					if !ok || err == nil {
						return
					}
					_ = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\n\x1b[33m[Console ended: %v]\x1b[0m\r\n", err)))
					return
				}
			}
		}()

		// Main Goroutine: Client -> Proxmox terminal
		for {
			_, p, err := ws.ReadMessage()
			if err != nil {
				closeDone()
				break
			}

			if len(p) > 0 {
				select {
				case stream.Send <- p:
				case <-done:
					return
				}
			}
		}
	}
}

func VNCWS(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuidVal, exists := c.Get("uuid")
		if !exists {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		userUUID := uuidVal.(string)

		vmidStr := c.Query("vmid")
		if vmidStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "vmid query parameter is required"})
			return
		}

		vmid, err := strconv.Atoi(vmidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
			return
		}

		upgradeHeader := http.Header{}
		for _, sub := range websocket.Subprotocols(c.Request) {
			if sub == "binary" {
				upgradeHeader.Set("Sec-WebSocket-Protocol", "binary")
				break
			}
		}

		ws, err := upgrader.Upgrade(c.Writer, c.Request, upgradeHeader)
		if err != nil {
			fmt.Printf("failed to upgrade VNC websocket: %v\n", err)
			return
		}
		defer ws.Close()

		stream, err := vms.OpenVMVNC(conn, userUUID, vmid)
		if err != nil {
			_ = ws.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, err.Error()),
				time.Now().Add(time.Second))
			return
		}
		defer stream.Close()

		done := make(chan struct{})
		var once sync.Once
		closeDone := func() {
			once.Do(func() {
				close(done)
			})
		}

		// Proxmox VNC -> Client
		go func() {
			defer closeDone()
			for {
				select {
				case <-done:
					return
				case msg, ok := <-stream.Recv:
					if !ok {
						return
					}
					if err := ws.WriteMessage(websocket.BinaryMessage, msg); err != nil {
						return
					}
				case err, ok := <-stream.Errs:
					if !ok || err == nil {
						return
					}
					return
				}
			}
		}()

		// Client -> Proxmox VNC
		for {
			_, p, err := ws.ReadMessage()
			if err != nil {
				closeDone()
				break
			}

			if len(p) > 0 {
				select {
				case stream.Send <- p:
				case <-done:
					return
				}
			}
		}
	}
}
