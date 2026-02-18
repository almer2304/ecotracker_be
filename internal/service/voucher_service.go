package service

import (
	"context"
	"fmt"

	"ecotracker/internal/domain"
	"ecotracker/internal/repository"

	"github.com/google/uuid"
)

type VoucherService struct {
	voucherRepo *repository.VoucherRepository
	authRepo    *repository.AuthRepository
}

func NewVoucherService(voucherRepo *repository.VoucherRepository, authRepo *repository.AuthRepository) *VoucherService {
	return &VoucherService{
		voucherRepo: voucherRepo,
		authRepo:    authRepo,
	}
}

// ListAvailable returns all active vouchers the user can claim
func (s *VoucherService) ListAvailable(ctx context.Context) ([]domain.Voucher, error) {
	vouchers, err := s.voucherRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list vouchers: %w", err)
	}
	if vouchers == nil {
		vouchers = []domain.Voucher{}
	}
	return vouchers, nil
}

// ClaimVoucher validates eligibility and runs the atomic claim transaction
func (s *VoucherService) ClaimVoucher(ctx context.Context, userID string, voucherID int) (*domain.UserVoucher, error) {
	// Get voucher details
	voucher, err := s.voucherRepo.GetByID(ctx, voucherID)
	if err != nil {
		return nil, fmt.Errorf("get voucher: %w", err)
	}

	if !voucher.IsActive {
		return nil, domain.ErrVoucherInactive
	}
	if voucher.Stock <= 0 {
		return nil, domain.ErrVoucherOutOfStock
	}

	// Get user's current balance
	profile, err := s.authRepo.GetProfileByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user profile: %w", err)
	}

	if profile.TotalPoints < voucher.PointCost {
		return nil, domain.ErrInsufficientPoints
	}

	// Generate a unique claim code
	claimCode := fmt.Sprintf("ECO-%s", uuid.New().String()[:8])

	// Execute atomic transaction
	uv, err := s.voucherRepo.ClaimVoucherTx(ctx, userID, voucherID, voucher.PointCost, claimCode)
	if err != nil {
		return nil, fmt.Errorf("claim voucher transaction: %w", err)
	}

	uv.Voucher = voucher
	return uv, nil
}

// GetMyVouchers returns all vouchers claimed by the user
func (s *VoucherService) GetMyVouchers(ctx context.Context, userID string) ([]domain.UserVoucher, error) {
	uvs, err := s.voucherRepo.GetUserVouchers(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get my vouchers: %w", err)
	}
	if uvs == nil {
		uvs = []domain.UserVoucher{}
	}
	return uvs, nil
}
