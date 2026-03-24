package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MessageType tipe pesan WebSocket
type MessageType string

const (
	// Server → Client
	MsgNewPickup        MessageType = "new_pickup"         // Ada pickup baru untuk collector
	MsgPickupAssigned   MessageType = "pickup_assigned"    // Pickup sudah di-assign ke collector (ke user)
	MsgPickupAccepted   MessageType = "pickup_accepted"    // Collector terima pickup (ke user)
	MsgPickupStarted    MessageType = "pickup_started"     // Collector mulai jalan (ke user)
	MsgPickupArrived    MessageType = "pickup_arrived"     // Collector tiba (ke user)
	MsgPickupCompleted  MessageType = "pickup_completed"   // Pickup selesai (ke user)
	MsgCollectorLocation MessageType = "collector_location" // Update lokasi collector (ke user)
	MsgPong             MessageType = "pong"
	MsgError            MessageType = "error"

	// Client → Server
	MsgPing             MessageType = "ping"
	MsgCollectorOnline  MessageType = "collector_online"
	MsgCollectorOffline MessageType = "collector_offline"
	MsgSubscribePickup  MessageType = "subscribe_pickup"   // User subscribe ke pickup tertentu
)

// Message format pesan WebSocket
type Message struct {
	Type      MessageType     `json:"type"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// NewPickupData data pickup baru untuk collector
type NewPickupData struct {
	PickupID  string  `json:"pickup_id"`
	Address   string  `json:"address"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	PhotoURL  *string `json:"photo_url,omitempty"`
	Notes     *string `json:"notes,omitempty"`
	Distance  float64 `json:"distance_km"`
	UserName  string  `json:"user_name"`
	CreatedAt string  `json:"created_at"`
}

// PickupStatusData data update status pickup
type PickupStatusData struct {
	PickupID      string  `json:"pickup_id"`
	Status        string  `json:"status"`
	CollectorName *string `json:"collector_name,omitempty"`
	CollectorLat  *float64 `json:"collector_lat,omitempty"`
	CollectorLon  *float64 `json:"collector_lon,omitempty"`
}

// LocationData data lokasi collector
type LocationData struct {
	CollectorID string  `json:"collector_id"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
}

// Client merepresentasikan satu koneksi WebSocket
type Client struct {
	ID          string
	Role        string // "user" atau "collector"
	Conn        interface{ WriteMessage(int, []byte) error; Close() error }
	Send        chan []byte
	Hub         *Hub
	mu          sync.Mutex

	// Untuk collector
	IsOnline    bool
	LastLat     *float64
	LastLon     *float64

	// Untuk user - pickup yang di-subscribe
	SubscribedPickups map[string]bool
}

// SendMessage mengirim pesan ke client ini
func (c *Client) SendMessage(msgType MessageType, data interface{}) error {
	var rawData json.RawMessage
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		rawData = b
	}

	msg := Message{
		Type:      msgType,
		Data:      rawData,
		Timestamp: time.Now(),
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case c.Send <- b:
		return nil
	default:
		return nil
	}
}

// Hub mengelola semua koneksi WebSocket aktif
type Hub struct {
	// Semua client yang terkoneksi: clientID → *Client
	clients map[string]*Client

	// Collector yang online: collectorID → *Client
	collectors map[string]*Client

	// User yang online: userID → *Client
	users map[string]*Client

	// Pickup subscriptions: pickupID → set of clientIDs
	pickupSubs map[string]map[string]bool

	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub membuat Hub baru
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		collectors: make(map[string]*Client),
		users:      make(map[string]*Client),
		pickupSubs: make(map[string]map[string]bool),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
	}
}

// Run menjalankan event loop Hub
func (h *Hub) Run() {
	logrus.Info("WebSocket Hub berjalan")
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			if client.Role == "collector" {
				h.collectors[client.ID] = client
				logrus.WithField("collector_id", client.ID).Info("Collector terhubung via WebSocket")
			} else if client.Role == "user" {
				h.users[client.ID] = client
				logrus.WithField("user_id", client.ID).Info("User terhubung via WebSocket")
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				delete(h.collectors, client.ID)
				delete(h.users, client.ID)
				// Bersihkan subscriptions
				for pickupID, subs := range h.pickupSubs {
					delete(subs, client.ID)
					if len(subs) == 0 {
						delete(h.pickupSubs, pickupID)
					}
				}
				close(client.Send)
				logrus.WithFields(logrus.Fields{
					"client_id": client.ID,
					"role":      client.Role,
				}).Info("Client WebSocket terputus")
			}
			h.mu.Unlock()
		}
	}
}

// Register mendaftarkan client baru
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister menghapus client
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// NotifyCollector mengirim notifikasi ke collector tertentu
func (h *Hub) NotifyCollector(collectorID string, msgType MessageType, data interface{}) bool {
	h.mu.RLock()
	client, ok := h.collectors[collectorID]
	h.mu.RUnlock()

	if !ok {
		return false
	}

	err := client.SendMessage(msgType, data)
	return err == nil
}

// NotifyUser mengirim notifikasi ke user tertentu
func (h *Hub) NotifyUser(userID string, msgType MessageType, data interface{}) bool {
	h.mu.RLock()
	client, ok := h.users[userID]
	h.mu.RUnlock()

	if !ok {
		return false
	}

	err := client.SendMessage(msgType, data)
	return err == nil
}

// NotifyPickupSubscribers mengirim notifikasi ke semua yang subscribe pickup tertentu
func (h *Hub) NotifyPickupSubscribers(pickupID string, msgType MessageType, data interface{}) {
	h.mu.RLock()
	subs, ok := h.pickupSubs[pickupID]
	if !ok {
		h.mu.RUnlock()
		return
	}
	// Copy subscriber IDs
	subIDs := make([]string, 0, len(subs))
	for id := range subs {
		subIDs = append(subIDs, id)
	}
	h.mu.RUnlock()

	for _, id := range subIDs {
		h.mu.RLock()
		client, ok := h.clients[id]
		h.mu.RUnlock()
		if ok {
			client.SendMessage(msgType, data)
		}
	}
}

// SubscribePickup user/collector subscribe ke update pickup tertentu
func (h *Hub) SubscribePickup(clientID, pickupID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.pickupSubs[pickupID] == nil {
		h.pickupSubs[pickupID] = make(map[string]bool)
	}
	h.pickupSubs[pickupID][clientID] = true
}

// GetOnlineCollectors mengambil daftar collector yang sedang online via WS
func (h *Hub) GetOnlineCollectors() []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*Client, 0, len(h.collectors))
	for _, c := range h.collectors {
		if c.IsOnline {
			result = append(result, c)
		}
	}
	return result
}

// IsCollectorConnected cek apakah collector sedang terkoneksi via WS
func (h *Hub) IsCollectorConnected(collectorID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.collectors[collectorID]
	return ok
}

// GetConnectedStats statistik koneksi aktif
func (h *Hub) GetConnectedStats() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return map[string]int{
		"total":      len(h.clients),
		"users":      len(h.users),
		"collectors": len(h.collectors),
	}
}
