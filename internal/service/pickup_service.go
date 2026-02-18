package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ecotracker/internal/domain"
	"ecotracker/internal/repository"
	"ecotracker/internal/utils"

	"github.com/google/uuid"
)

type PickupService struct {
	pickupRepo   *repository.PickupRepository
	categoryRepo *repository.WasteCategoryRepository
	authRepo     *repository.AuthRepository
	storage      *utils.StorageClient
	bucket       string
}

func NewPickupService(
	pickupRepo *repository.PickupRepository,
	categoryRepo *repository.WasteCategoryRepository,
	authRepo *repository.AuthRepository,
	storage *utils.StorageClient,
	bucket string,
) *PickupService {
	return &PickupService{
		pickupRepo:   pickupRepo,
		categoryRepo: categoryRepo,
		authRepo:     authRepo,
		storage:      storage,
		bucket:       bucket,
	}
}

// CreatePickup handles image processing, upload, and pickup record creation
func (s *PickupService) CreatePickup(
	ctx context.Context,
	userID string,
	req *domain.CreatePickupRequest,
	fileHeader *multipart.FileHeader,
) (*domain.Pickup, error) {
	var photoURL string

	if fileHeader != nil {
		// Process (resize + convert) the image
		imgBytes, contentType, err := utils.ProcessImage(fileHeader)
		if err != nil {
			return nil, fmt.Errorf("process image: %w", err)
		}

		// Generate unique filename
		ext := ".jpg"
		origExt := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if origExt == ".png" {
			ext = ".jpg" // always store as JPEG after processing
		}
		filename := fmt.Sprintf("%s-%d%s", uuid.New().String(), time.Now().Unix(), ext)
		filePath := fmt.Sprintf("pickups/%s", filename)

		// Upload to Supabase Storage
		photoURL, err = s.storage.UploadImage(s.bucket, filePath, imgBytes, contentType)
		if err != nil {
			return nil, fmt.Errorf("upload photo: %w", err)
		}
	}

	pickup := &domain.Pickup{
		UserID:    userID,
		Address:   req.Address,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		PhotoURL:  photoURL,
		Notes:     req.Notes,
	}

	if err := s.pickupRepo.Create(ctx, pickup); err != nil {
		return nil, fmt.Errorf("save pickup: %w", err)
	}
	return pickup, nil
}

// GetPickupDetail returns pickup with its items
func (s *PickupService) GetPickupDetail(ctx context.Context, pickupID, requesterID, requesterRole string) (*domain.PickupDetail, error) {
	pickup, err := s.pickupRepo.GetByID(ctx, pickupID)
	if err != nil {
		return nil, fmt.Errorf("get pickup: %w", err)
	}

	// Authorization: user can only see their own, collector can see all
	if requesterRole == "user" && pickup.UserID != requesterID {
		return nil, domain.ErrForbidden
	}

	items, _ := s.pickupRepo.GetItemsByPickupID(ctx, pickupID)

	return &domain.PickupDetail{Pickup: *pickup, Items: items}, nil
}

// ListMyPickups returns all pickups for the current user
func (s *PickupService) ListMyPickups(ctx context.Context, userID string) ([]domain.Pickup, error) {
	pickups, err := s.pickupRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user pickups: %w", err)
	}
	if pickups == nil {
		pickups = []domain.Pickup{}
	}
	return pickups, nil
}

// ListPendingPickups returns all pending pickups for collector dashboard
func (s *PickupService) ListPendingPickups(ctx context.Context) ([]domain.Pickup, error) {
	pickups, err := s.pickupRepo.ListByStatus(ctx, "pending")
	if err != nil {
		return nil, fmt.Errorf("list pending pickups: %w", err)
	}
	if pickups == nil {
		pickups = []domain.Pickup{}
	}
	return pickups, nil
}

// ListPendingPickupsNearby returns pending pickups sorted by distance from collector
func (s *PickupService) ListPendingPickupsNearby(ctx context.Context, collectorLat, collectorLon float64) ([]domain.PickupWithDistance, error) {
	pickups, err := s.pickupRepo.ListByStatus(ctx, "pending")
	if err != nil {
		return nil, fmt.Errorf("list pending pickups: %w", err)
	}

	var result []domain.PickupWithDistance
	for _, p := range pickups {
		distance := utils.CalculateDistance(collectorLat, collectorLon, p.Latitude, p.Longitude)
		result = append(result, domain.PickupWithDistance{
			ID:          p.ID,
			UserID:      p.UserID,
			CollectorID: p.CollectorID,
			Status:      p.Status,
			Address:     p.Address,
			Latitude:    p.Latitude,
			Longitude:   p.Longitude,
			PhotoURL:    p.PhotoURL,
			Notes:       p.Notes,
			CompletedAt: p.CompletedAt,
			CreatedAt:   p.CreatedAt,
			DistanceKm:  distance,
		})
	}

	// Sort by distance (closest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].DistanceKm < result[j].DistanceKm
	})

	return result, nil
}

// ListPendingPickupsNearbyPostGIS uses database-level geospatial query (PRODUCTION)
// Requires PostGIS extension - see migrations/002_postgis.sql
func (s *PickupService) ListPendingPickupsNearbyPostGIS(ctx context.Context, collectorLat, collectorLon float64, limit int) ([]domain.PickupWithDistance, error) {
	if limit <= 0 {
		limit = 50 // default
	}
	pickups, err := s.pickupRepo.ListByStatusNearLocation(ctx, "pending", collectorLat, collectorLon, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending pickups nearby (PostGIS): %w", err)
	}
	return pickups, nil
}

// ListMyTasks returns pickups assigned to a collector
func (s *PickupService) ListMyTasks(ctx context.Context, collectorID string) ([]domain.Pickup, error) {
	pickups, err := s.pickupRepo.ListByCollectorID(ctx, collectorID)
	if err != nil {
		return nil, fmt.Errorf("list collector tasks: %w", err)
	}
	if pickups == nil {
		pickups = []domain.Pickup{}
	}
	return pickups, nil
}

// TakeTask assigns a collector to a pending pickup
func (s *PickupService) TakeTask(ctx context.Context, pickupID, collectorID string) (*domain.Pickup, error) {
	if err := s.pickupRepo.TakeTask(ctx, pickupID, collectorID); err != nil {
		return nil, fmt.Errorf("take task: %w", err)
	}

	pickup, err := s.pickupRepo.GetByID(ctx, pickupID)
	if err != nil {
		return nil, fmt.Errorf("get updated pickup: %w", err)
	}
	return pickup, nil
}

// CompleteTask runs the atomic pickup completion workflow
func (s *PickupService) CompleteTask(ctx context.Context, pickupID, collectorID string, req *domain.CompletePickupRequest) (*domain.PickupDetail, error) {
	// Verify pickup exists and belongs to this collector
	pickup, err := s.pickupRepo.GetByID(ctx, pickupID)
	if err != nil {
		return nil, fmt.Errorf("get pickup for completion: %w", err)
	}
	if pickup.CollectorID != collectorID {
		return nil, domain.ErrForbidden
	}
	if pickup.Status != "taken" {
		return nil, domain.ErrPickupNotTaken
	}

	// Calculate points per item
	var items []domain.PickupItem
	totalPoints := 0

	for _, input := range req.Items {
		category, err := s.categoryRepo.GetByID(ctx, input.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("get category %d: %w", input.CategoryID, err)
		}
		subtotal := int(input.Weight * float64(category.PointsPerKg))
		totalPoints += subtotal

		items = append(items, domain.PickupItem{
			PickupID:       pickupID,
			CategoryID:     input.CategoryID,
			Weight:         input.Weight,
			SubtotalPoints: subtotal,
		})
	}

	// Run atomic DB transaction
	if err := s.pickupRepo.CompletePickupTx(ctx, pickupID, collectorID, items, totalPoints, pickup.UserID); err != nil {
		return nil, fmt.Errorf("complete pickup transaction: %w", err)
	}

	// Return final state
	updatedPickup, _ := s.pickupRepo.GetByID(ctx, pickupID)
	finalItems, _ := s.pickupRepo.GetItemsByPickupID(ctx, pickupID)

	return &domain.PickupDetail{Pickup: *updatedPickup, Items: finalItems}, nil
}
