package service

import (
	"context"
	"fmt"

	"ecotracker/internal/domain"
	"ecotracker/internal/repository"
	"ecotracker/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

type AdminService struct {
	authRepo   *repository.AuthRepository
	pickupRepo *repository.PickupRepository
	jwtUtil    *utils.JWTUtil
}

func NewAdminService(
	authRepo *repository.AuthRepository,
	pickupRepo *repository.PickupRepository,
	jwtUtil *utils.JWTUtil,
) *AdminService {
	return &AdminService{
		authRepo:   authRepo,
		pickupRepo: pickupRepo,
		jwtUtil:    jwtUtil,
	}
}

// CreateCollector creates a new collector account (admin only)
func (s *AdminService) CreateCollector(ctx context.Context, req *domain.RegisterRequest) (*domain.Profile, error) {
	// Check if email already exists
	exists, err := s.authRepo.EmailExists(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil, domain.ErrEmailAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Create collector profile
	profile := &domain.Profile{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Role:         "collector",
		PasswordHash: string(hashedPassword),
	}

	if err := s.authRepo.CreateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("create collector: %w", err)
	}

	// Don't return password hash
	profile.PasswordHash = ""

	return profile, nil
}

// ListCollectors returns all users with role=collector
func (s *AdminService) ListCollectors(ctx context.Context) ([]domain.Profile, error) {
	collectors, err := s.authRepo.ListProfilesByRole(ctx, "collector")
	if err != nil {
		return nil, fmt.Errorf("list collectors: %w", err)
	}

	// Remove password hashes
	for i := range collectors {
		collectors[i].PasswordHash = ""
	}

	return collectors, nil
}

// GetDashboardStats returns admin dashboard statistics
func (s *AdminService) GetDashboardStats(ctx context.Context) (*domain.AdminStats, error) {
	stats := &domain.AdminStats{}

	// Get user count
	userCount, err := s.authRepo.CountProfilesByRole(ctx, "user")
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	stats.TotalUsers = userCount

	// Get collector count
	collectorCount, err := s.authRepo.CountProfilesByRole(ctx, "collector")
	if err != nil {
		return nil, fmt.Errorf("count collectors: %w", err)
	}
	stats.TotalCollectors = collectorCount

	// Get pickup counts by status
	pendingCount, err := s.pickupRepo.CountByStatus(ctx, "pending")
	if err != nil {
		return nil, fmt.Errorf("count pending pickups: %w", err)
	}
	stats.PendingPickups = pendingCount

	completedCount, err := s.pickupRepo.CountByStatus(ctx, "completed")
	if err != nil {
		return nil, fmt.Errorf("count completed pickups: %w", err)
	}
	stats.CompletedPickups = completedCount

	// Get total points awarded
	totalPoints, err := s.authRepo.GetTotalPoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("get total points: %w", err)
	}
	stats.TotalPointsAwarded = totalPoints

	return stats, nil
}

// ListAllPickups returns all pickups (optionally filtered by status)
func (s *AdminService) ListAllPickups(ctx context.Context, status string) ([]domain.Pickup, error) {
	if status != "" {
		pickups, err := s.pickupRepo.ListByStatus(ctx, status)
		if err != nil {
			return nil, fmt.Errorf("list pickups by status: %w", err)
		}
		return pickups, nil
	}

	// Get all pickups
	pickups, err := s.pickupRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all pickups: %w", err)
	}

	return pickups, nil
}

// DeleteCollector soft deletes a collector account
func (s *AdminService) DeleteCollector(ctx context.Context, collectorID string) error {
	// Check if collector exists
	collector, err := s.authRepo.GetProfileByID(ctx, collectorID)
	if err != nil {
		return fmt.Errorf("get collector: %w", err)
	}

	if collector.Role != "collector" {
		return fmt.Errorf("user is not a collector")
	}

	// Delete collector
	if err := s.authRepo.DeleteProfile(ctx, collectorID); err != nil {
		return fmt.Errorf("delete collector: %w", err)
	}

	return nil
}
