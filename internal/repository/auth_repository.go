package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ecotracker/backend/internal/domain"
)

// AuthRepository mengelola operasi database untuk autentikasi
type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

// Create membuat profil user baru
func (r *AuthRepository) Create(ctx context.Context, name, email, passwordHash, phone string, role domain.UserRole) (*domain.Profile, error) {
	query := `
		INSERT INTO profiles (name, email, password_hash, phone, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, email, phone, role,
		          total_points, total_pickups_completed, total_reports_submitted,
		          is_online, is_busy, average_rating, total_ratings, total_weight_collected,
		          last_lat, last_lon, last_location_updated_at,
		          avatar_url, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query, name, email, passwordHash, nullStr(phone), role)
	return scanProfile(row)
}

// GetByEmail mencari profil berdasarkan email (termasuk password_hash untuk login)
func (r *AuthRepository) GetByEmail(ctx context.Context, email string) (*domain.Profile, string, error) {
	query := `
		SELECT id, name, email, phone, role, password_hash,
		       total_points, total_pickups_completed, total_reports_submitted,
		       is_online, is_busy, average_rating, total_ratings, total_weight_collected,
		       last_lat, last_lon, last_location_updated_at,
		       avatar_url, created_at, updated_at
		FROM profiles
		WHERE email = $1 AND deleted_at IS NULL`

	var p domain.Profile
	var passwordHash string
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&p.ID, &p.Name, &p.Email, &p.Phone, &p.Role, &passwordHash,
		&p.TotalPoints, &p.TotalPickupsCompleted, &p.TotalReportsSubmitted,
		&p.IsOnline, &p.IsBusy, &p.AverageRating, &p.TotalRatings, &p.TotalWeightCollected,
		&p.LastLat, &p.LastLon, &p.LastLocationUpdatedAt,
		&p.AvatarURL, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, "", domain.ErrInvalidCredentials
	}
	if err != nil {
		return nil, "", fmt.Errorf("GetByEmail: %w", err)
	}
	return &p, passwordHash, nil
}

// GetByID mencari profil berdasarkan ID
func (r *AuthRepository) GetByID(ctx context.Context, id string) (*domain.Profile, error) {
	query := `
		SELECT id, name, email, phone, role,
		       total_points, total_pickups_completed, total_reports_submitted,
		       is_online, is_busy, average_rating, total_ratings, total_weight_collected,
		       last_lat, last_lon, last_location_updated_at,
		       avatar_url, created_at, updated_at
		FROM profiles
		WHERE id = $1 AND deleted_at IS NULL`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanProfile(row)
}

// SaveRefreshToken menyimpan refresh token ke database
func (r *AuthRepository) SaveRefreshToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE profiles SET refresh_token = $1, refresh_token_expires_at = $2 WHERE id = $3`,
		token, expiresAt, userID,
	)
	return err
}

// GetByRefreshToken mencari user berdasarkan refresh token
func (r *AuthRepository) GetByRefreshToken(ctx context.Context, token string) (*domain.Profile, error) {
	query := `
		SELECT id, name, email, phone, role,
		       total_points, total_pickups_completed, total_reports_submitted,
		       is_online, is_busy, average_rating, total_ratings, total_weight_collected,
		       created_at, updated_at
		FROM profiles
		WHERE refresh_token = $1
		  AND refresh_token_expires_at > NOW()
		  AND deleted_at IS NULL`

	row := r.db.QueryRowContext(ctx, query, token)
	return scanProfile(row)
}

// UpdateLocation memperbarui lokasi GPS collector
func (r *AuthRepository) UpdateLocation(ctx context.Context, collectorID string, lat, lon float64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE profiles SET last_lat=$1, last_lon=$2, last_location_updated_at=NOW() WHERE id=$3`,
		lat, lon, collectorID,
	)
	return err
}

// UpdateOnlineStatus mengubah status online/offline collector
func (r *AuthRepository) UpdateOnlineStatus(ctx context.Context, collectorID string, isOnline bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE profiles SET is_online=$1 WHERE id=$2`,
		isOnline, collectorID,
	)
	return err
}

// UpdatePoints menambah poin user (atomic)
func (r *AuthRepository) UpdatePoints(ctx context.Context, tx *sql.Tx, userID string, points int) (int, error) {
	var newBalance int
	err := tx.QueryRowContext(ctx,
		`UPDATE profiles SET total_points = total_points + $1 WHERE id = $2 RETURNING total_points`,
		points, userID,
	).Scan(&newBalance)
	return newBalance, err
}

// IncrementPickupsCompleted menambah counter pickup selesai user
func (r *AuthRepository) IncrementPickupsCompleted(ctx context.Context, tx *sql.Tx, userID string) (int, error) {
	var total int
	err := tx.QueryRowContext(ctx,
		`UPDATE profiles SET total_pickups_completed = total_pickups_completed + 1 WHERE id = $1 RETURNING total_pickups_completed`,
		userID,
	).Scan(&total)
	return total, err
}

// IncrementReportsSubmitted menambah counter laporan yang dikirim
func (r *AuthRepository) IncrementReportsSubmitted(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE profiles SET total_reports_submitted = total_reports_submitted + 1 WHERE id = $1`,
		userID,
	)
	return err
}

// AddWeightCollected menambah total berat yang dikumpulkan collector
func (r *AuthRepository) AddWeightCollected(ctx context.Context, tx *sql.Tx, collectorID string, weightKg float64) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE profiles SET total_weight_collected = total_weight_collected + $1 WHERE id = $2`,
		weightKg, collectorID,
	)
	return err
}

// scanProfile membaca baris database ke struct Profile
func scanProfile(row *sql.Row) (*domain.Profile, error) {
	var p domain.Profile
	err := row.Scan(
		&p.ID, &p.Name, &p.Email, &p.Phone, &p.Role,
		&p.TotalPoints, &p.TotalPickupsCompleted, &p.TotalReportsSubmitted,
		&p.IsOnline, &p.IsBusy, &p.AverageRating, &p.TotalRatings, &p.TotalWeightCollected,
		&p.LastLat, &p.LastLon, &p.LastLocationUpdatedAt,
		&p.AvatarURL, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan profile: %w", err)
	}
	return &p, nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}