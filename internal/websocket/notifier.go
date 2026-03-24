package websocket

import (
	"time"

	"github.com/sirupsen/logrus"
)

// Notifier adalah interface untuk mengirim notifikasi WebSocket
// Dipanggil dari service layer setelah operasi berhasil
type Notifier struct {
	hub *Hub
}

func NewNotifier(hub *Hub) *Notifier {
	return &Notifier{hub: hub}
}

// NotifyNewPickup memberitahu collector terdekat bahwa ada pickup baru
// Dipanggil dari AssignmentService setelah pickup berhasil di-assign
func (n *Notifier) NotifyNewPickup(collectorID string, data NewPickupData) {
	sent := n.hub.NotifyCollector(collectorID, MsgNewPickup, data)
	logrus.WithFields(logrus.Fields{
		"collector_id": collectorID,
		"pickup_id":    data.PickupID,
		"ws_sent":      sent,
	}).Info("Notifikasi pickup baru dikirim ke collector")
}

// NotifyPickupAssigned memberitahu user bahwa pickupnya sudah di-assign ke collector
func (n *Notifier) NotifyPickupAssigned(userID, pickupID, collectorName string) {
	data := PickupStatusData{
		PickupID:      pickupID,
		Status:        "assigned",
		CollectorName: &collectorName,
	}
	// Kirim ke user langsung
	n.hub.NotifyUser(userID, MsgPickupAssigned, data)
	// Kirim ke semua subscriber pickup ini
	n.hub.NotifyPickupSubscribers(pickupID, MsgPickupAssigned, data)
}

// NotifyPickupStatusUpdate memberitahu user tentang perubahan status pickup
func (n *Notifier) NotifyPickupStatusUpdate(userID, pickupID, status string) {
	data := PickupStatusData{
		PickupID: pickupID,
		Status:   status,
	}

	var msgType MessageType
	switch status {
	case "accepted":
		msgType = MsgPickupAccepted
	case "in_progress":
		msgType = MsgPickupStarted
	case "arrived":
		msgType = MsgPickupArrived
	case "completed":
		msgType = MsgPickupCompleted
	default:
		return
	}

	n.hub.NotifyUser(userID, msgType, data)
	n.hub.NotifyPickupSubscribers(pickupID, msgType, data)
}

// NotifyCollectorLocation broadcast lokasi collector ke user yang subscribe pickup aktif
func (n *Notifier) NotifyCollectorLocation(pickupID, collectorID string, lat, lon float64) {
	data := map[string]interface{}{
		"pickup_id":    pickupID,
		"collector_id": collectorID,
		"lat":          lat,
		"lon":          lon,
		"timestamp":    time.Now(),
	}
	n.hub.NotifyPickupSubscribers(pickupID, MsgCollectorLocation, data)
}

// GetStats statistik WebSocket connections
func (n *Notifier) GetStats() map[string]int {
	return n.hub.GetConnectedStats()
}
