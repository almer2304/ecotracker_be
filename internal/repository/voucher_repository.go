package repository

import (
	"context"
	"errors"
	"fmt"

	"ecotracker/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VoucherRepository struct {
	db *pgxpool.Pool
}

func NewVoucherRepository(db *pgxpool.Pool) *VoucherRepository {
	return &VoucherRepository{db: db}
}

// ListActive returns all active vouchers with stock > 0
func (r *VoucherRepository) ListActive(ctx context.Context) ([]domain.Voucher, error) {
	query := `
		SELECT id, title, description, point_cost, stock, COALESCE(image_url,''), is_active
		FROM vouchers
		WHERE is_active = true AND stock > 0
		ORDER BY point_cost ASC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list active vouchers: %w", err)
	}
	defer rows.Close()

	var vouchers []domain.Voucher
	for rows.Next() {
		var v domain.Voucher
		if err := rows.Scan(&v.ID, &v.Title, &v.Description, &v.PointCost, &v.Stock, &v.ImageURL, &v.IsActive); err != nil {
			return nil, fmt.Errorf("scan voucher: %w", err)
		}
		vouchers = append(vouchers, v)
	}
	return vouchers, rows.Err()
}

// GetByID fetches a single voucher
func (r *VoucherRepository) GetByID(ctx context.Context, id int) (*domain.Voucher, error) {
	query := `
		SELECT id, title, description, point_cost, stock, COALESCE(image_url,''), is_active
		FROM vouchers WHERE id=$1`

	var v domain.Voucher
	err := r.db.QueryRow(ctx, query, id).Scan(&v.ID, &v.Title, &v.Description, &v.PointCost, &v.Stock, &v.ImageURL, &v.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get voucher by id: %w", err)
	}
	return &v, nil
}

// ClaimVoucherTx atomically: deducts user points, decrements stock, creates user_voucher
func (r *VoucherRepository) ClaimVoucherTx(ctx context.Context, userID string, voucherID, pointCost int, claimCode string) (*domain.UserVoucher, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin claim transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Step 1: Deduct points from user
	var newBalance int
	err = tx.QueryRow(ctx,
		`UPDATE profiles SET total_points = total_points - $1 WHERE id=$2 AND total_points >= $1 RETURNING total_points`,
		pointCost, userID,
	).Scan(&newBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInsufficientPoints
		}
		return nil, fmt.Errorf("deduct points: %w", err)
	}

	// Step 2: Decrement voucher stock
	result, err := tx.Exec(ctx,
		`UPDATE vouchers SET stock = stock - 1 WHERE id=$1 AND stock > 0 AND is_active = true`,
		voucherID,
	)
	if err != nil {
		return nil, fmt.Errorf("decrement voucher stock: %w", err)
	}
	if result.RowsAffected() == 0 {
		return nil, domain.ErrVoucherOutOfStock
	}

	// Step 3: Create user_voucher record
	uv := &domain.UserVoucher{}
	err = tx.QueryRow(ctx,
		`INSERT INTO user_vouchers (user_id, voucher_id, claim_code, is_used, claimed_at)
		 VALUES ($1, $2, $3, false, NOW())
		 RETURNING id, user_id, voucher_id, claim_code, is_used, claimed_at`,
		userID, voucherID, claimCode,
	).Scan(&uv.ID, &uv.UserID, &uv.VoucherID, &uv.ClaimCode, &uv.IsUsed, &uv.ClaimedAt)
	if err != nil {
		return nil, fmt.Errorf("insert user voucher: %w", err)
	}

	// Step 4: Log the spend transaction
	_, err = tx.Exec(ctx,
		`INSERT INTO point_logs (user_id, amount, transaction_type, description)
		 VALUES ($1, $2, 'spend', 'Points spent on voucher redemption')`,
		userID, pointCost,
	)
	if err != nil {
		return nil, fmt.Errorf("log spend transaction: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim transaction: %w", err)
	}
	return uv, nil
}

// GetUserVouchers returns all vouchers claimed by a user
func (r *VoucherRepository) GetUserVouchers(ctx context.Context, userID string) ([]domain.UserVoucher, error) {
	query := `
		SELECT uv.id, uv.user_id, uv.voucher_id, uv.claim_code, uv.is_used, uv.claimed_at,
		       v.id, v.title, v.description, v.point_cost, v.stock, COALESCE(v.image_url,''), v.is_active
		FROM user_vouchers uv
		JOIN vouchers v ON v.id = uv.voucher_id
		WHERE uv.user_id = $1
		ORDER BY uv.claimed_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get user vouchers: %w", err)
	}
	defer rows.Close()

	var userVouchers []domain.UserVoucher
	for rows.Next() {
		var uv domain.UserVoucher
		v := &domain.Voucher{}
		err := rows.Scan(
			&uv.ID, &uv.UserID, &uv.VoucherID, &uv.ClaimCode, &uv.IsUsed, &uv.ClaimedAt,
			&v.ID, &v.Title, &v.Description, &v.PointCost, &v.Stock, &v.ImageURL, &v.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user voucher: %w", err)
		}
		uv.Voucher = v
		userVouchers = append(userVouchers, uv)
	}
	return userVouchers, rows.Err()
}
