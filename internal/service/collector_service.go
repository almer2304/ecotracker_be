package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/repository"
	ws "github.com/ecotracker/backend/internal/websocket"
	"github.com/sirupsen/logrus"
)

// CollectorService mengelola logika bisnis untuk collector
type CollectorService struct {
	authRepo     *repository.AuthRepository
	pickupRepo   *repository.PickupRepository
	categoryRepo *repository.CategoryRepository
	pointLogRepo *repository.PointLogRepository
	badgeService *BadgeService
	db           *sql.DB
	notifier     *ws.Notifier
}

// SetNotifier meng-inject WebSocket notifier ke CollectorService
func (s *CollectorService) SetNotifier(n *ws.Notifier) {
	s.notifier = n
}

func NewCollectorService(
	authRepo *repository.AuthRepository,
	pickupRepo *repository.PickupRepository,
	categoryRepo *repository.CategoryRepository,
	pointLogRepo *repository.PointLogRepository,
	badgeService *BadgeService,
	db *sql.DB,
) *CollectorService {
	return &CollectorService{
		authRepo:     authRepo,
		pickupRepo:   pickupRepo,
		categoryRepo: categoryRepo,
		pointLogRepo: pointLogRepo,
		badgeService: badgeService,
		db:           db,
	}
}

// UpdateStatus mengubah status online/offline collector
func (s *CollectorService) UpdateStatus(ctx context.Context, collectorID string, isOnline bool) error {
	return s.authRepo.UpdateOnlineStatus(ctx, collectorID, isOnline)
}

// UpdateLocation memperbarui lokasi GPS collector
func (s *CollectorService) UpdateLocation(ctx context.Context, collectorID string, lat, lon float64) error {
	return s.authRepo.UpdateLocation(ctx, collectorID, lat, lon)
}

// GetAssignedPickup mengambil pickup yang sedang ditugaskan ke collector
func (s *CollectorService) GetAssignedPickup(ctx context.Context, collectorID string) (*domain.Pickup, error) {
	return s.pickupRepo.GetAssignedPickupByCollector(ctx, collectorID)
}

// AcceptPickup collector menerima pickup yang ditugaskan
func (s *CollectorService) AcceptPickup(ctx context.Context, collectorID, pickupID string) (*domain.Pickup, error) {
	pickup, err := s.pickupRepo.GetByID(ctx, pickupID)
	if err != nil {
		return nil, err
	}

	if pickup.CollectorID == nil || *pickup.CollectorID != collectorID {
		return nil, domain.ErrForbidden
	}

	if pickup.Status != domain.StatusAssigned && pickup.Status != domain.StatusReassigned {
		return nil, domain.ErrInvalidPickupStatus
	}

	acceptedField := "accepted_at"
	if err := s.pickupRepo.UpdateStatus(ctx, pickupID, domain.StatusAccepted, &acceptedField); err != nil {
		return nil, fmt.Errorf("accept pickup: %w", err)
	}

	return s.pickupRepo.GetByID(ctx, pickupID)
}

// StartPickup collector mulai menuju lokasi pickup
func (s *CollectorService) StartPickup(ctx context.Context, collectorID, pickupID string) (*domain.Pickup, error) {
	pickup, err := s.validateCollectorPickup(ctx, collectorID, pickupID, domain.StatusAccepted)
	if err != nil {
		return nil, err
	}
	_ = pickup

	startedField := "started_at"
	if err := s.pickupRepo.UpdateStatus(ctx, pickupID, domain.StatusInProgress, &startedField); err != nil {
		return nil, err
	}

	return s.pickupRepo.GetByID(ctx, pickupID)
}

// ArriveAtPickup collector tiba di lokasi user
func (s *CollectorService) ArriveAtPickup(ctx context.Context, collectorID, pickupID string) (*domain.Pickup, error) {
	_, err := s.validateCollectorPickup(ctx, collectorID, pickupID, domain.StatusInProgress)
	if err != nil {
		return nil, err
	}

	arrivedField := "arrived_at"
	if err := s.pickupRepo.UpdateStatus(ctx, pickupID, domain.StatusArrived, &arrivedField); err != nil {
		return nil, err
	}

	return s.pickupRepo.GetByID(ctx, pickupID)
}

// CompletePickup menyelesaikan pickup secara atomik
func (s *CollectorService) CompletePickup(ctx context.Context, collectorID, pickupID string, req *domain.CompletePickupRequest) (*domain.Pickup, error) {
	pickup, err := s.validateCollectorPickup(ctx, collectorID, pickupID, domain.StatusArrived)
	if err != nil {
		return nil, err
	}

	var totalWeight float64
	var totalPoints int
	var itemResults []struct {
		PickupID   string
		CategoryID string
		WeightKg   float64
		Points     int
	}

	for _, item := range req.Items {
		cat, err := s.categoryRepo.GetByID(ctx, item.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("kategori tidak ditemukan: %s", item.CategoryID)
		}

		points := int(item.WeightKg * float64(cat.PointsPerKg))
		totalWeight += item.WeightKg
		totalPoints += points

		itemResults = append(itemResults, struct {
			PickupID   string
			CategoryID string
			WeightKg   float64
			Points     int
		}{pickupID, item.CategoryID, item.WeightKg, points})
	}

	tx, err := s.pickupRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback()

	if err := s.pickupRepo.CompletePickup(ctx, tx, pickupID, totalWeight, totalPoints); err != nil {
		return nil, fmt.Errorf("complete pickup: %w", err)
	}

	if err := s.pickupRepo.AddPickupItems(ctx, tx, itemResults); err != nil {
		return nil, fmt.Errorf("simpan items: %w", err)
	}

	newBalance, err := s.authRepo.UpdatePoints(ctx, tx, pickup.UserID, totalPoints)
	if err != nil {
		return nil, fmt.Errorf("tambah poin: %w", err)
	}

	desc := fmt.Sprintf("Poin dari pickup #%s (%.2f kg)", pickupID[:8], totalWeight)
	if err := s.pointLogRepo.Create(ctx, tx, pickup.UserID, &pickupID, domain.PointEarned, totalPoints, desc, newBalance); err != nil {
		return nil, fmt.Errorf("catat point log: %w", err)
	}

	totalPickups, err := s.authRepo.IncrementPickupsCompleted(ctx, tx, pickup.UserID)
	if err != nil {
		return nil, fmt.Errorf("increment pickups: %w", err)
	}

	_, err = tx.ExecContext(ctx, `UPDATE profiles SET is_busy = false WHERE id = $1`, collectorID)
	if err != nil {
		return nil, fmt.Errorf("lepas collector: %w", err)
	}

	if err := s.authRepo.AddWeightCollected(ctx, tx, collectorID, totalWeight); err != nil {
		return nil, fmt.Errorf("update weight collector: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	go func() {
		bgCtx := context.Background()
		if err := s.badgeService.CheckAndAwardBadges(bgCtx, pickup.UserID, totalPickups, newBalance); err != nil {
			logrus.WithError(err).Warn("Gagal cek badge")
		}
	}()

	return s.pickupRepo.GetByID(ctx, pickupID)
}

// validateCollectorPickup validasi bahwa collector berhak mengakses pickup dengan status tertentu
func (s *CollectorService) validateCollectorPickup(ctx context.Context, collectorID, pickupID string, expectedStatus domain.PickupStatus) (*domain.Pickup, error) {
	pickup, err := s.pickupRepo.GetByID(ctx, pickupID)
	if err != nil {
		return nil, err
	}

	if pickup.CollectorID == nil || *pickup.CollectorID != collectorID {
		return nil, domain.ErrForbidden
	}

	if pickup.Status != expectedStatus {
		return nil, domain.ErrInvalidPickupStatus
	}

	return pickup, nil
}