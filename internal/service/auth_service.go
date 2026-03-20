package service

import (
	"context"
	"fmt"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/repository"
	"github.com/ecotracker/backend/internal/utils"
)

type AuthService struct {
	authRepo   *repository.AuthRepository
	jwtManager *utils.JWTManager
	bcryptCost int
}

func NewAuthService(authRepo *repository.AuthRepository, jwtManager *utils.JWTManager, bcryptCost int) *AuthService {
	return &AuthService{authRepo: authRepo, jwtManager: jwtManager, bcryptCost: bcryptCost}
}

// Register mendaftarkan user baru (role = user)
func (s *AuthService) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error) {
	return s.RegisterWithRole(ctx, req, domain.RoleUser)
}

// RegisterWithRole mendaftarkan akun dengan role tertentu (admin/collector/user)
func (s *AuthService) RegisterWithRole(ctx context.Context, req *domain.RegisterRequest, role domain.UserRole) (*domain.AuthResponse, error) {
	// Cek email sudah terdaftar
	_, _, err := s.authRepo.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, domain.ErrEmailAlreadyExists
	}
	if err != domain.ErrInvalidCredentials {
		return nil, fmt.Errorf("cek email: %w", err)
	}

	// Hash password
	hash, err := utils.HashPassword(req.Password, s.bcryptCost)
	if err != nil {
		return nil, err
	}

	// Buat profil
	profile, err := s.authRepo.Create(ctx, req.Name, req.Email, hash, req.Phone, role)
	if err != nil {
		return nil, fmt.Errorf("buat profil: %w", err)
	}

	return s.generateTokenResponse(ctx, profile)
}

// Login memverifikasi kredensial dan mengembalikan token
func (s *AuthService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error) {
	profile, hash, err := s.authRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if !utils.CheckPassword(req.Password, hash) {
		return nil, domain.ErrInvalidCredentials
	}

	return s.generateTokenResponse(ctx, profile)
}

// RefreshToken menghasilkan access token baru dari refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthResponse, error) {
	// Validasi refresh token
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// Cek di database (pastikan belum di-revoke)
	profile, err := s.authRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil || profile.ID != claims.UserID {
		return nil, domain.ErrInvalidToken
	}

	return s.generateTokenResponse(ctx, profile)
}

// GetProfile mengambil profil user berdasarkan ID
func (s *AuthService) GetProfile(ctx context.Context, userID string) (*domain.Profile, error) {
	return s.authRepo.GetByID(ctx, userID)
}

func (s *AuthService) generateTokenResponse(ctx context.Context, profile *domain.Profile) (*domain.AuthResponse, error) {
	// Generate access token
	accessToken, _, err := s.jwtManager.GenerateAccessToken(profile.ID, profile.Email, string(profile.Role))
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, refreshExpires, err := s.jwtManager.GenerateRefreshToken(profile.ID)
	if err != nil {
		return nil, err
	}

	// Simpan refresh token
	if err := s.authRepo.SaveRefreshToken(ctx, profile.ID, refreshToken, refreshExpires); err != nil {
		return nil, err
	}

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.jwtManager.AccessTokenExpirySeconds(),
		User:         profile,
	}, nil
}