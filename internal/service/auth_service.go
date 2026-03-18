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

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error) {
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

	// Create profile
	profile := &domain.Profile{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Role:         req.Role,
		PasswordHash: string(hashedPassword),
	}

	if err := s.authRepo.CreateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Generate JWT token
	token, err := s.jwtUtil.GenerateToken(profile.ID, profile.Email, profile.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Return response without password hash
	profile.PasswordHash = ""

	return &domain.AuthResponse{
		Token:   token,
		Profile: profile,
	}, nil
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.AuthResponse, error) {
	fmt.Printf("[LOGIN DEBUG] Email: %s\n", email)
	
	// Get user by email
	user, err := s.authRepo.GetProfileByEmail(ctx, email)
	if err != nil {
		fmt.Printf("[LOGIN ERROR] User not found: %v\n", err)
		return nil, domain.ErrInvalidCredentials
	}
	
	fmt.Printf("[LOGIN DEBUG] User found: %s (ID: %s)\n", user.Email, user.ID)
	
	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		fmt.Printf("[LOGIN ERROR] Password mismatch: %v\n", err)
		return nil, domain.ErrInvalidCredentials
	}
	
	fmt.Printf("[LOGIN SUCCESS] Password verified!\n")
	
	// Generate token
	token, err := s.jwtUtil.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	
	// Return response without password hash
	user.PasswordHash = ""
	
	return &domain.AuthResponse{
		Token:   token,
		Profile: user,
	}, nil
}

// GetProfile fetches user profile by ID
func (s *AuthService) GetProfile(ctx context.Context, userID string) (*domain.Profile, error) {
	profile, err := s.authRepo.GetProfileByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	
	// Don't return password hash
	profile.PasswordHash = ""
	
	return profile, nil
}