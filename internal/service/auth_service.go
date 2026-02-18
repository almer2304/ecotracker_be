package service

import (
	"context"
	"fmt"

	"ecotracker/internal/domain"
	"ecotracker/internal/repository"
	"ecotracker/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	authRepo *repository.AuthRepository
	jwtUtil  *utils.JWTUtil
}

func NewAuthService(authRepo *repository.AuthRepository, jwtUtil *utils.JWTUtil) *AuthService {
	return &AuthService{
		authRepo: authRepo,
		jwtUtil:  jwtUtil,
	}
}

// Register creates a new profile with hashed password
func (s *AuthService) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error) {
	// Check for duplicate email
	exists, err := s.authRepo.EmailExists(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("register check email: %w", err)
	}
	if exists {
		return nil, domain.ErrConflict
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	profile := &domain.Profile{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Role:         req.Role,
		PasswordHash: string(hashed),
	}

	if err := s.authRepo.CreateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}

	token, err := s.jwtUtil.GenerateToken(profile.ID, profile.Email, profile.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &domain.AuthResponse{Token: token, Profile: profile}, nil
}

// Login validates credentials and returns a JWT token
func (s *AuthService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error) {
	profile, err := s.authRepo.GetProfileByEmail(ctx, req.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(profile.PasswordHash), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	token, err := s.jwtUtil.GenerateToken(profile.ID, profile.Email, profile.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &domain.AuthResponse{Token: token, Profile: profile}, nil
}

// GetProfile returns the profile for the given user ID
func (s *AuthService) GetProfile(ctx context.Context, userID string) (*domain.Profile, error) {
	profile, err := s.authRepo.GetProfileByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	return profile, nil
}
