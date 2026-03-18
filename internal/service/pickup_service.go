package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"ecotracker/internal/domain"
	"ecotracker/internal/repository"
	"ecotracker/internal/utils"

	"github.com/google/uuid"
)

type PickupService struct {
	pickupRepo *repository.PickupRepository
	storage    *utils.StorageClient
	bucket     string
}

func NewPickupService(pickupRepo *repository.PickupRepository, storage *utils.StorageClient, bucket string) *PickupService {
	return &PickupService{
		pickupRepo: pickupRepo,
		storage:    storage,
		bucket:     bucket,
	}
}

// CreatePickup - FIXED VERSION with direct upload (no image processing)
func (s *PickupService) CreatePickup(
	ctx context.Context,
	userID string,
	req *domain.CreatePickupRequest,
	fileHeader *multipart.FileHeader,
) (*domain.Pickup, error) {
	var photoURL string

	// Photo upload - SIMPLIFIED (no processing)
	if fileHeader != nil {
		fmt.Printf("[DEBUG] Processing photo: %s, size: %d bytes\n", fileHeader.Filename, fileHeader.Size)
		
		// Open file
		file, err := fileHeader.Open()
		if err != nil {
			fmt.Printf("[ERROR] Failed to open file: %v\n", err)
			return nil, fmt.Errorf("open file: %w", err)
		}
		defer file.Close()

		// Read all bytes
		imgBytes, err := io.ReadAll(file)
		if err != nil {
			fmt.Printf("[ERROR] Failed to read file: %v\n", err)
			return nil, fmt.Errorf("read file: %w", err)
		}
		fmt.Printf("[DEBUG] File read successfully: %d bytes\n", len(imgBytes))

		// Generate unique filename
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if ext == "" {
			ext = ".jpg"
		}
		filename := fmt.Sprintf("%s-%d%s", uuid.New().String(), time.Now().Unix(), ext)
		filePath := fmt.Sprintf("pickups/%s", filename)

		// Determine content type
		contentType := fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			// Guess from extension
			switch ext {
			case ".jpg", ".jpeg":
				contentType = "image/jpeg"
			case ".png":
				contentType = "image/png"
			case ".webp":
				contentType = "image/webp"
			default:
				contentType = "image/jpeg"
			}
		}
		fmt.Printf("[DEBUG] Uploading to Supabase: %s (type: %s)\n", filePath, contentType)

		// Upload to Supabase Storage
		photoURL, err = s.storage.UploadImage(s.bucket, filePath, imgBytes, contentType)
		if err != nil {
			fmt.Printf("[ERROR] Upload failed: %v\n", err)
			return nil, fmt.Errorf("upload photo to storage: %w", err)
		}
		fmt.Printf("[SUCCESS] Photo uploaded: %s\n", photoURL)
	}

	// Create pickup record
	pickup := &domain.Pickup{
		UserID:    userID,
		Address:   req.Address,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		PhotoURL:  photoURL,
		Notes:     req.Notes,
	}

	fmt.Printf("[DEBUG] Creating pickup record in database...\n")
	if err := s.pickupRepo.Create(ctx, pickup); err != nil {
		fmt.Printf("[ERROR] Database insert failed: %v\n", err)
		return nil, fmt.Errorf("save pickup to database: %w", err)
	}
	
	fmt.Printf("[SUCCESS] Pickup created with ID: %s\n", pickup.ID)
	return pickup, nil
}

// GetPickupDetail returns pickup with its items
func (s *PickupService) GetPickupDetail(ctx context.Context, pickupID string) (*domain.PickupDetail, error) {
	pickup, err := s.pickupRepo.GetByID(ctx, pickupID)
	if err != nil {
		return nil, fmt.Errorf("get pickup: %w", err)
	}

	items, err := s.pickupRepo.GetItemsByPickupID(ctx, pickupID)
	if err != nil {
		return nil, fmt.Errorf("get items: %w", err)
	}

	detail := &domain.PickupDetail{
		Pickup: *pickup,
		Items:  items,
	}
	return detail, nil
}

// ListMyPickups returns all pickups created by the user
func (s *PickupService) ListMyPickups(ctx context.Context, userID string) ([]domain.Pickup, error) {
	pickups, err := s.pickupRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list my pickups: %w", err)
	}
	return pickups, nil
}

// ListPendingPickups returns all pending pickups (for collectors)
func (s *PickupService) ListPendingPickups(ctx context.Context) ([]domain.Pickup, error) {
	pickups, err := s.pickupRepo.ListByStatus(ctx, "pending")
	if err != nil {
		return nil, fmt.Errorf("list pending pickups: %w", err)
	}
	return pickups, nil
}

// ListPendingPickupsNearby returns pending pickups sorted by distance from collector's location
func (s *PickupService) ListPendingPickupsNearby(ctx context.Context, lat, lon float64) ([]domain.PickupWithDistance, error) {
	// Get all pending pickups
	pickups, err := s.pickupRepo.ListByStatus(ctx, "pending")
	if err != nil {
		return nil, fmt.Errorf("list pending pickups: %w", err)
	}

	// Calculate distances and create result
	results := make([]domain.PickupWithDistance, 0)
	for _, p := range pickups {
		// Skip invalid coordinates (0,0)
		if p.Latitude == 0 && p.Longitude == 0 {
			continue
		}

		// Calculate distance using Haversine formula
		distance := utils.CalculateDistance(lat, lon, p.Latitude, p.Longitude)

		result := domain.PickupWithDistance{
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
		}
		results = append(results, result)
	}

	// Sort by distance (bubble sort for simplicity)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].DistanceKm > results[j].DistanceKm {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results, nil
}

// ListPendingPickupsNearbyPostGIS uses PostGIS for distance calculation (faster for large datasets)
func (s *PickupService) ListPendingPickupsNearbyPostGIS(ctx context.Context, lat, lon float64) ([]domain.PickupWithDistance, error) {
	pickups, err := s.pickupRepo.ListByStatusNearLocation(ctx, "pending", lat, lon, 50)
	if err != nil {
		return nil, fmt.Errorf("list pending pickups near location (PostGIS): %w", err)
	}
	return pickups, nil
}

// TakePickup allows collector to take a pending pickup
func (s *PickupService) TakePickup(ctx context.Context, pickupID, collectorID string) (*domain.Pickup, error) {
	// Use repository's TakeTask method
	if err := s.pickupRepo.TakeTask(ctx, pickupID, collectorID); err != nil {
		return nil, fmt.Errorf("take pickup: %w", err)
	}

	// Get updated pickup
	pickup, err := s.pickupRepo.GetByID(ctx, pickupID)
	if err != nil {
		return nil, fmt.Errorf("get pickup: %w", err)
	}

	return pickup, nil
}

// CompletePickup marks pickup as completed and awards points
func (s *PickupService) CompletePickup(
	ctx context.Context,
	pickupID, collectorID string,
	items []domain.PickupItemInput,
) (*domain.Pickup, int, error) {
	fmt.Printf("[COMPLETE] Starting completion for pickup: %s by collector: %s\n", pickupID, collectorID)
	fmt.Printf("[COMPLETE] Items: %+v\n", items)
	
	// Get pickup first to get userID
	pickup, err := s.pickupRepo.GetByID(ctx, pickupID)
	if err != nil {
		fmt.Printf("[COMPLETE ERROR] Get pickup failed: %v\n", err)
		return nil, 0, fmt.Errorf("get pickup: %w", err)
	}

	fmt.Printf("[COMPLETE] Pickup found: %+v\n", pickup)

	// Validate ownership
	if pickup.CollectorID != collectorID {
		fmt.Printf("[COMPLETE ERROR] Collector mismatch. Expected: %s, Got: %s\n", pickup.CollectorID, collectorID)
		return nil, 0, fmt.Errorf("pickup not assigned to this collector")
	}

	// Validate status
	if pickup.Status != "taken" {
		fmt.Printf("[COMPLETE ERROR] Invalid status: %s (expected: taken)\n", pickup.Status)
		return nil, 0, fmt.Errorf("pickup is not taken (current status: %s)", pickup.Status)
	}

	// Convert input items to domain.PickupItem and calculate points
	totalPoints := 0
	pickupItems := make([]domain.PickupItem, 0, len(items))
	
	for i, item := range items {
		fmt.Printf("[COMPLETE] Processing item %d: category_id=%d, weight=%.2f\n", i+1, item.CategoryID, item.Weight)
		
		// Validate weight
		if item.Weight <= 0 {
			fmt.Printf("[COMPLETE ERROR] Invalid weight for item %d: %.2f\n", i+1, item.Weight)
			return nil, 0, fmt.Errorf("item %d: weight must be greater than 0", i+1)
		}
		
		// Calculate points: weight * 10 (simplified)
		subtotalPoints := int(item.Weight * 10)
		totalPoints += subtotalPoints

		pickupItems = append(pickupItems, domain.PickupItem{
			PickupID:       pickupID,
			CategoryID:     item.CategoryID,
			Weight:         item.Weight,
			SubtotalPoints: subtotalPoints,
		})
	}

	fmt.Printf("[COMPLETE] Total points calculated: %d\n", totalPoints)

	// Use repository's atomic transaction to complete pickup
	fmt.Printf("[COMPLETE] Calling CompletePickupTx...\n")
	if err := s.pickupRepo.CompletePickupTx(ctx, pickupID, collectorID, pickupItems, totalPoints, pickup.UserID); err != nil {
		fmt.Printf("[COMPLETE ERROR] Transaction failed: %v\n", err)
		return nil, 0, fmt.Errorf("complete pickup transaction: %w", err)
	}

	fmt.Printf("[COMPLETE] Transaction successful!\n")

	// Get updated pickup
	pickup, err = s.pickupRepo.GetByID(ctx, pickupID)
	if err != nil {
		fmt.Printf("[COMPLETE ERROR] Get updated pickup failed: %v\n", err)
		return nil, 0, fmt.Errorf("get updated pickup: %w", err)
	}

	fmt.Printf("[COMPLETE SUCCESS] Pickup %s completed with %d points\n", pickupID, totalPoints)
	return pickup, totalPoints, nil
}

// ListCollectorTasks returns all pickups assigned to a collector
func (s *PickupService) ListCollectorTasks(ctx context.Context, collectorID string) ([]domain.Pickup, error) {
	pickups, err := s.pickupRepo.ListByCollectorID(ctx, collectorID)
	if err != nil {
		return nil, fmt.Errorf("list collector tasks: %w", err)
	}
	return pickups, nil
}