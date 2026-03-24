package websocket

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow semua origin (sesuaikan di production)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler mengelola WebSocket endpoint
type Handler struct {
	hub *Hub
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// ServeWS endpoint: GET /ws?token=<jwt_token>
// Client harus kirim JWT token via query param atau header
func (h *Handler) ServeWS(c *gin.Context) {
	// Ambil info user dari context (sudah di-set oleh AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid"})
		return
	}
	userRole, _ := c.Get("user_role")

	// Upgrade ke WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.WithError(err).Error("Gagal upgrade ke WebSocket")
		return
	}

	// Buat client baru
	client := &Client{
		ID:                userID.(string),
		Role:              userRole.(string),
		Conn:              conn,
		Send:              make(chan []byte, 256),
		Hub:               h.hub,
		SubscribedPickups: make(map[string]bool),
	}

	// Daftarkan ke hub
	h.hub.Register(client)

	// Jalankan goroutine untuk read dan write
	go client.writePump(conn)
	go client.readPump(conn, h.hub)
}

// readPump membaca pesan dari client
func (c *Client) readPump(conn *websocket.Conn, hub *Hub) {
	defer func() {
		hub.Unregister(c)
		conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.WithError(err).Warn("WebSocket closed unexpectedly")
			}
			break
		}

		// Parse pesan
		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			c.SendMessage(MsgError, map[string]string{"message": "Format pesan tidak valid"})
			continue
		}

		// Handle pesan dari client
		c.handleMessage(msg, hub)
	}
}

// writePump menulis pesan ke client
func (c *Client) writePump(conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel ditutup
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Kirim semua pesan yang pending sekaligus
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// Kirim ping untuk keep-alive
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage memproses pesan yang diterima dari client
func (c *Client) handleMessage(msg Message, hub *Hub) {
	switch msg.Type {

	case MsgPing:
		c.SendMessage(MsgPong, map[string]string{"status": "ok"})

	case MsgCollectorOnline:
		// Collector set status online
		if c.Role == "collector" {
			c.IsOnline = true
			logrus.WithField("collector_id", c.ID).Info("Collector set online via WS")
		}

	case MsgCollectorOffline:
		// Collector set status offline
		if c.Role == "collector" {
			c.IsOnline = false
			logrus.WithField("collector_id", c.ID).Info("Collector set offline via WS")
		}

	case MsgSubscribePickup:
		// User/collector subscribe ke update pickup tertentu
		var data struct {
			PickupID string `json:"pickup_id"`
		}
		if err := json.Unmarshal(msg.Data, &data); err == nil && data.PickupID != "" {
			hub.SubscribePickup(c.ID, data.PickupID)
			c.SubscribedPickups[data.PickupID] = true
			logrus.WithFields(logrus.Fields{
				"client_id": c.ID,
				"pickup_id": data.PickupID,
			}).Debug("Client subscribe ke pickup")
		}
	}
}
