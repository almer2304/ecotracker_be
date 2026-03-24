package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/repository"
	"github.com/ecotracker/backend/internal/utils"
	ws "github.com/ecotracker/backend/internal/websocket"
	"github.com/sirupsen/logrus"
)

// AssignmentService mengelola logika penugasan collector ke pickup
type AssignmentService struct {
	pickupRepo    *repository.PickupRepository
	collectorRepo *repository.CollectorRepository
	authRepo      *repository.AuthRepository
	timeout       time.Duration
	db            *sql.DB
	notifier      *ws.Notifier // WebSocket notifier
}

func NewAssignmentService(
	pickupRepo *repository.PickupRepository,
	collectorRepo *repository.CollectorRepository,
	authRepo *repository.AuthRepository,
	db *sql.DB,
	timeout time.Duration,
	notifier *ws.Notifier,
) *AssignmentService {
	return &AssignmentService{
		pickupRepo:    pickupRepo,
		collectorRepo: collectorRepo,
		authRepo:      authRepo,
		timeout:       timeout,
		db:            db,
		notifier:      notifier,
	}
}

// AssignClosestCollector algoritma inti auto-assignment
func (s *AssignmentService) AssignClosestCollector(ctx context.Context, pickupID string, pickupLat, pickupLon float64, excludeIDs []string) error {
	log := logrus.WithField("pickup_id", pickupID)

	// 1. Ambil semua collector yang tersedia
	collectors, err := s.collectorRepo.FindAvailable(ctx, excludeIDs)
	if err != nil {
		return fmt.Errorf("cari collector: %w", err)
	}

	if len(collectors) == 0 {
		log.Warn("Tidak ada collector tersedia")
		return domain.ErrNoCollectorAvailable
	}

	// 2. Hitung jarak Haversine ke setiap collector
	withDistances := make([]utils.CollectorWithDistance, 0, len(collectors))
	for _, c := range collectors {
		if c.LastLat == nil || c.LastLon == nil {
			continue
		}
		dist := utils.HaversineDistance(pickupLat, pickupLon, *c.LastLat, *c.LastLon)
		withDistances = append(withDistances, utils.CollectorWithDistance{
			ID:         c.ID,
			Lat:        *c.LastLat,
			Lon:        *c.LastLon,
			DistanceKm: dist,
		})
	}

	if len(withDistances) == 0 {
		return domain.ErrNoCollectorAvailable
	}

	// 3. Urutkan dari jarak terdekat
	utils.SortByDistance(withDistances)
	nearest := withDistances[0]

	// 4. Atomic transaction: assign pickup + set collector busy
	tx, err := s.pickupRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback()

	assignTimeout := time.Now().Add(s.timeout)
	if err := s.pickupRepo.AssignToCollector(ctx, tx, pickupID, nearest.ID, assignTimeout, nearest.DistanceKm); err != nil {
		return fmt.Errorf("tugaskan collector: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	log.WithFields(logrus.Fields{
		"collector_id": nearest.ID,
		"distance_km":  fmt.Sprintf("%.2f", nearest.DistanceKm),
	}).Info("Pickup berhasil ditugaskan")

	// 5. Kirim notifikasi WebSocket ke collector (async)
	go func() {
		if s.notifier == nil {
			return
		}

		// Ambil detail pickup untuk notifikasi
		pickup, err := s.pickupRepo.GetByID(context.Background(), pickupID)
		if err != nil {
			return
		}

		userName := ""
		if pickup.User != nil {
			userName = pickup.User.Name
		}

		notifData := ws.NewPickupData{
			PickupID:  pickupID,
			Address:   pickup.Address,
			Lat:       pickup.Lat,
			Lon:       pickup.Lon,
			PhotoURL:  pickup.PhotoURL,
			Notes:     pickup.Notes,
			Distance:  nearest.DistanceKm,
			UserName:  userName,
			CreatedAt: pickup.CreatedAt.Format(time.RFC3339),
		}

		s.notifier.NotifyNewPickup(nearest.ID, notifData)

		// Notify user bahwa pickupnya sudah di-assign
		collectorName := ""
		for _, c := range collectors {
			if c.ID == nearest.ID {
				collectorName = c.Name
				break
			}
		}
		s.notifier.NotifyPickupAssigned(pickup.UserID, pickupID, collectorName)
	}()

	return nil
}

// ReassignPickup melepaskan collector lama dan mencari pengganti
func (s *AssignmentService) ReassignPickup(ctx context.Context, pickup domain.Pickup) error {
	if pickup.CollectorID == nil {
		return nil
	}

	log := logrus.WithFields(logrus.Fields{
		"pickup_id":     pickup.ID,
		"old_collector": *pickup.CollectorID,
	})
	log.Info("Memulai reassignment pickup")

	excludeIDs := s.getTriedCollectorIDs(context.Background(), pickup.ID)

	tx, err := s.pickupRepo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.pickupRepo.ReleaseCollector(ctx, tx, pickup.ID, *pickup.CollectorID, "timeout"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	err = s.AssignClosestCollector(ctx, pickup.ID, pickup.Lat, pickup.Lon, excludeIDs)
	if err != nil {
		if err == domain.ErrNoCollectorAvailable {
			log.Warn("Tidak ada collector pengganti, pickup kembali ke pending")
			s.pickupRepo.UpdateStatus(ctx, pickup.ID, domain.StatusPending, nil)
			return nil
		}
		return err
	}

	log.Info("Pickup berhasil di-reassign")
	return nil
}

func (s *AssignmentService) getTriedCollectorIDs(ctx context.Context, pickupID string) []string {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT collector_id FROM assignment_history WHERE pickup_id = $1`,
		pickupID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
