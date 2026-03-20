package service

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/repository"
	"github.com/ecotracker/backend/internal/utils"
)

// PickupService mengelola logika bisnis untuk pickup request
type PickupService struct {
	pickupRepo        *repository.PickupRepository
	assignmentService *AssignmentService
	storageClient     *utils.StorageClient
}

func NewPickupService(
	pickupRepo *repository.PickupRepository,
	assignmentService *AssignmentService,
	storageClient *utils.StorageClient,
) *PickupService {
	return &PickupService{
		pickupRepo:        pickupRepo,
		assignmentService: assignmentService,
		storageClient:     storageClient,
	}
}

// CreatePickup membuat pickup baru, upload foto, lalu auto-assign collector terdekat
func (s *PickupService) CreatePickup(
	ctx context.Context,
	userID string,
	req *domain.CreatePickupRequest,
	photoFile multipart.File,
	photoHeader *multipart.FileHeader,
) (*domain.Pickup, error) {
	var photoURL *string

	// Upload foto jika ada
	if photoFile != nil && photoHeader != nil {
		if s.storageClient == nil {
			return nil, fmt.Errorf("storage belum dikonfigurasi, cek SUPABASE_URL dan SUPABASE_KEY di .env")
		}
		url, err := s.storageClient.UploadPickupPhoto(ctx, photoFile, photoHeader)
		if err != nil {
			return nil, fmt.Errorf("upload foto: %w", err)
		}
		photoURL = &url
	}

	var notesPtr *string
	if req.Notes != "" {
		notesPtr = &req.Notes
	}

	// Simpan pickup ke database
	pickup, err := s.pickupRepo.Create(ctx, userID, req.Address, req.Lat, req.Lon, photoURL, notesPtr)
	if err != nil {
		return nil, fmt.Errorf("buat pickup: %w", err)
	}

	// Auto-assign collector terdekat (async, tidak blokir response)
	go func() {
		bgCtx := context.Background()
		if err := s.assignmentService.AssignClosestCollector(bgCtx, pickup.ID, pickup.Lat, pickup.Lon, nil); err != nil {
			// Log error tapi tidak gagalkan request - pickup tetap tersimpan dengan status pending
			_ = err
		}
	}()

	return pickup, nil
}

// GetMyPickups mengambil daftar pickup milik user dengan pagination
func (s *PickupService) GetMyPickups(ctx context.Context, userID string, page, limit int) ([]*domain.Pickup, int, error) {
	offset := (page - 1) * limit
	return s.pickupRepo.ListByUserID(ctx, userID, limit, offset)
}

// GetPickupDetail mengambil detail pickup (validasi kepemilikan)
func (s *PickupService) GetPickupDetail(ctx context.Context, pickupID, userID string, role domain.UserRole) (*domain.Pickup, error) {
	pickup, err := s.pickupRepo.GetByID(ctx, pickupID)
	if err != nil {
		return nil, err
	}

	// Validasi akses: user hanya bisa lihat pickup milik sendiri
	// Collector hanya bisa lihat pickup yang ditugaskan kepadanya
	// Admin bisa lihat semua
	switch role {
	case domain.RoleUser:
		if pickup.UserID != userID {
			return nil, domain.ErrForbidden
		}
	case domain.RoleCollector:
		if pickup.CollectorID == nil || *pickup.CollectorID != userID {
			return nil, domain.ErrForbidden
		}
	}

	// Ambil juga item-item pickup jika sudah completed
	if pickup.Status == domain.StatusCompleted {
		items, err := s.pickupRepo.GetPickupItems(ctx, pickupID)
		if err == nil {
			pickup.Items = items
		}
	}

	return pickup, nil
}