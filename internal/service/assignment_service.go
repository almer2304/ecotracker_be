package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/repository"
	"github.com/ecotracker/backend/internal/utils"
	"github.com/sirupsen/logrus"
)

// AssignmentService mengelola logika penugasan collector ke pickup
type AssignmentService struct {
	pickupRepo    *repository.PickupRepository
	collectorRepo *repository.CollectorRepository
	timeout       time.Duration
	db            *sql.DB
}

func NewAssignmentService(
	pickupRepo *repository.PickupRepository,
	collectorRepo *repository.CollectorRepository,
	db *sql.DB,
	timeout time.Duration,
) *AssignmentService {
	return &AssignmentService{
		pickupRepo:    pickupRepo,
		collectorRepo: collectorRepo,
		timeout:       timeout,
		db:            db,
	}
}

// AssignClosestCollector adalah algoritma inti auto-assignment
// Mencari collector terdekat yang online & tidak busy, lalu menugaskan secara atomik
func (s *AssignmentService) AssignClosestCollector(ctx context.Context, pickupID string, pickupLat, pickupLon float64, excludeIDs []string) error {
	log := logrus.WithField("pickup_id", pickupID)

	// 1. Ambil semua collector yang tersedia (online, tidak busy, lokasi baru)
	collectors, err := s.collectorRepo.FindAvailable(ctx, excludeIDs)
	if err != nil {
		return fmt.Errorf("cari collector: %w", err)
	}

	if len(collectors) == 0 {
		log.Warn("Tidak ada collector tersedia untuk pickup ini")
		return domain.ErrNoCollectorAvailable
	}

	// 2. Hitung jarak dari setiap collector ke lokasi pickup
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

	// 4. Atomic transaction: assign pickup + set collector busy + catat history
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
		return fmt.Errorf("commit transaksi: %w", err)
	}

	log.WithFields(logrus.Fields{
		"collector_id": nearest.ID,
		"distance_km":  fmt.Sprintf("%.2f", nearest.DistanceKm),
		"timeout":      assignTimeout,
	}).Info("Pickup berhasil ditugaskan ke collector terdekat")

	return nil
}

// ReassignPickup melepaskan collector lama dan mencari pengganti
func (s *AssignmentService) ReassignPickup(ctx context.Context, pickup domain.Pickup) error {
	if pickup.CollectorID == nil {
		return nil
	}

	log := logrus.WithFields(logrus.Fields{
		"pickup_id":    pickup.ID,
		"old_collector": *pickup.CollectorID,
	})
	log.Info("Memulai reassignment pickup")

	// Kumpulkan ID collector yang pernah ditugaskan (dari assignment_history)
	excludeIDs := s.getTriedCollectorIDs(context.Background(), pickup.ID)

	// Atomic: release collector lama
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

	// Cari collector baru (kecualikan yang sudah pernah dicoba)
	err = s.AssignClosestCollector(ctx, pickup.ID, pickup.Lat, pickup.Lon, excludeIDs)
	if err != nil {
		if err == domain.ErrNoCollectorAvailable {
			log.Warn("Tidak ada collector pengganti, pickup tetap pending")
			// Update status ke pending agar bisa di-assign ulang nanti
			s.pickupRepo.UpdateStatus(ctx, pickup.ID, domain.StatusPending, nil)
			return nil
		}
		return err
	}

	log.Info("Pickup berhasil di-reassign")
	return nil
}

// getTriedCollectorIDs mengambil daftar ID collector yang pernah ditugaskan
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
