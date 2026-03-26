package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ecotracker/backend/internal/domain"
)

// PickupRepository mengelola operasi database untuk pickup
type PickupRepository struct {
	db *sql.DB
}

func NewPickupRepository(db *sql.DB) *PickupRepository {
	return &PickupRepository{db: db}
}

// Create membuat pickup baru
func (r *PickupRepository) Create(ctx context.Context, userID, address string, lat, lon float64, photoURL, notes *string) (*domain.Pickup, error) {
	query := `
		INSERT INTO pickups (user_id, address, lat, lon, photo_url, notes, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		RETURNING id, user_id, collector_id, address, lat, lon, photo_url, notes, status,
		          assigned_at, assignment_timeout, reassignment_count,
		          accepted_at, started_at, arrived_at, completed_at,
		          total_weight, total_points_awarded, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query, userID, address, lat, lon, photoURL, notes)
	return scanPickup(row)
}

// GetByID mencari pickup berdasarkan ID dengan data user dan collector
func (r *PickupRepository) GetByID(ctx context.Context, id string) (*domain.Pickup, error) {
	query := `
		SELECT
			p.id, p.user_id, p.collector_id, p.address, p.lat, p.lon,
			p.photo_url, p.notes, p.status,
			p.assigned_at, p.assignment_timeout, p.reassignment_count,
			p.accepted_at, p.started_at, p.arrived_at, p.completed_at,
			p.total_weight, p.total_points_awarded, p.created_at, p.updated_at,
			u.name AS user_name, u.phone AS user_phone,
			c.name AS collector_name, c.phone AS collector_phone, c.average_rating AS collector_rating
		FROM pickups p
		JOIN profiles u ON u.id = p.user_id
		LEFT JOIN profiles c ON c.id = p.collector_id
		WHERE p.id = $1 AND p.deleted_at IS NULL`

	var p domain.Pickup
	var userName, userPhone sql.NullString
	var collectorName, collectorPhone sql.NullString
	var collectorRating sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.UserID, &p.CollectorID, &p.Address, &p.Lat, &p.Lon,
		&p.PhotoURL, &p.Notes, &p.Status,
		&p.AssignedAt, &p.AssignmentTimeout, &p.ReassignmentCount,
		&p.AcceptedAt, &p.StartedAt, &p.ArrivedAt, &p.CompletedAt,
		&p.TotalWeight, &p.TotalPointsAwarded, &p.CreatedAt, &p.UpdatedAt,
		&userName, &userPhone,
		&collectorName, &collectorPhone, &collectorRating,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrPickupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("GetByID: %w", err)
	}

	// Set user info
	p.User = &domain.Profile{ID: p.UserID}
	if userName.Valid {
		p.User.Name = userName.String
	}
	if userPhone.Valid {
		phone := userPhone.String
		p.User.Phone = &phone
	}

	// Set collector info jika ada
	if p.CollectorID != nil {
		p.Collector = &domain.Profile{ID: *p.CollectorID}
		if collectorName.Valid {
			p.Collector.Name = collectorName.String
		}
		if collectorPhone.Valid {
			phone := collectorPhone.String
			p.Collector.Phone = &phone
		}
		if collectorRating.Valid {
			p.Collector.AverageRating = collectorRating.Float64
		}
	}

	return &p, nil
}

// ListByUserID mengambil daftar pickup milik user
func (r *PickupRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Pickup, int, error) {
	// Count total
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pickups WHERE user_id = $1 AND deleted_at IS NULL`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT p.id, p.user_id, p.collector_id, p.address, p.lat, p.lon,
		       p.photo_url, p.notes, p.status,
		       p.assigned_at, p.assignment_timeout, p.reassignment_count,
		       p.accepted_at, p.started_at, p.arrived_at, p.completed_at,
		       p.total_weight, p.total_points_awarded, p.created_at, p.updated_at,
		       c.name AS collector_name
		FROM pickups p
		LEFT JOIN profiles c ON c.id = p.collector_id
		WHERE p.user_id = $1 AND p.deleted_at IS NULL
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var pickups []*domain.Pickup
	for rows.Next() {
		var p domain.Pickup
		var collectorName sql.NullString
		err := rows.Scan(
			&p.ID, &p.UserID, &p.CollectorID, &p.Address, &p.Lat, &p.Lon,
			&p.PhotoURL, &p.Notes, &p.Status,
			&p.AssignedAt, &p.AssignmentTimeout, &p.ReassignmentCount,
			&p.AcceptedAt, &p.StartedAt, &p.ArrivedAt, &p.CompletedAt,
			&p.TotalWeight, &p.TotalPointsAwarded, &p.CreatedAt, &p.UpdatedAt,
			&collectorName,
		)
		if err != nil {
			return nil, 0, err
		}
		if p.CollectorID != nil && collectorName.Valid {
			p.Collector = &domain.Profile{ID: *p.CollectorID, Name: collectorName.String}
		}
		pickups = append(pickups, &p)
	}

	return pickups, total, rows.Err()
}

// AssignToCollector menugaskan pickup ke collector (dalam transaksi)
func (r *PickupRepository) AssignToCollector(ctx context.Context, tx *sql.Tx, pickupID, collectorID string, timeout time.Time, distanceKm float64) error {
	// Update pickup
	_, err := tx.ExecContext(ctx, `
		UPDATE pickups
		SET collector_id = $1, status = 'assigned',
		    assigned_at = NOW(), assignment_timeout = $2
		WHERE id = $3`,
		collectorID, timeout, pickupID,
	)
	if err != nil {
		return fmt.Errorf("update pickup: %w", err)
	}

	// Set collector sebagai busy
	_, err = tx.ExecContext(ctx,
		`UPDATE profiles SET is_busy = true WHERE id = $1`,
		collectorID,
	)
	if err != nil {
		return fmt.Errorf("update collector busy: %w", err)
	}

	// Catat di assignment_history
	_, err = tx.ExecContext(ctx, `
		INSERT INTO assignment_history (pickup_id, collector_id, assigned_at, timeout_at, distance_km)
		VALUES ($1, $2, NOW(), $3, $4)`,
		pickupID, collectorID, timeout, distanceKm,
	)
	return err
}

// ReleaseCollector melepaskan collector dari pickup (saat timeout / cancel)
func (r *PickupRepository) ReleaseCollector(ctx context.Context, tx *sql.Tx, pickupID, collectorID, reason string) error {
	// Release collector
	_, err := tx.ExecContext(ctx,
		`UPDATE profiles SET is_busy = false WHERE id = $1`,
		collectorID,
	)
	if err != nil {
		return fmt.Errorf("release collector: %w", err)
	}

	// Update pickup status
	_, err = tx.ExecContext(ctx, `
		UPDATE pickups
		SET status = 'reassigned', collector_id = NULL,
		    reassignment_count = reassignment_count + 1
		WHERE id = $1`,
		pickupID,
	)
	if err != nil {
		return fmt.Errorf("update pickup reassigned: %w", err)
	}

	// Update assignment_history
	_, err = tx.ExecContext(ctx, `
		UPDATE assignment_history
		SET released_at = NOW(), release_reason = $1
		WHERE pickup_id = $2 AND collector_id = $3 AND released_at IS NULL`,
		reason, pickupID, collectorID,
	)
	return err
}

// UpdateStatus mengubah status pickup
func (r *PickupRepository) UpdateStatus(ctx context.Context, pickupID string, status domain.PickupStatus, timestampField *string) error {
	query := `UPDATE pickups SET status = $1`
	args := []interface{}{status}

	if timestampField != nil {
		query += fmt.Sprintf(", %s = NOW()", *timestampField)
	}

	query += " WHERE id = $2"
	args = append(args, pickupID)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// CompletePickup menyelesaikan pickup secara atomik (dalam transaksi)
func (r *PickupRepository) CompletePickup(ctx context.Context, tx *sql.Tx, pickupID string, totalWeight float64, totalPoints int) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE pickups
		SET status = 'completed', completed_at = NOW(),
		    total_weight = $1, total_points_awarded = $2
		WHERE id = $3`,
		totalWeight, totalPoints, pickupID,
	)
	return err
}

// AddPickupItems menyimpan detail item sampah
func (r *PickupRepository) AddPickupItems(ctx context.Context, tx *sql.Tx, items []struct {
	PickupID   string
	CategoryID string
	WeightKg   float64
	Points     int
}) error {
	for _, item := range items {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO pickup_items (pickup_id, category_id, weight_kg, points_awarded)
			VALUES ($1, $2, $3, $4)`,
			item.PickupID, item.CategoryID, item.WeightKg, item.Points,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPickupItems mengambil item-item dalam satu pickup
func (r *PickupRepository) GetPickupItems(ctx context.Context, pickupID string) ([]domain.PickupItem, error) {
	query := `
		SELECT pi.id, pi.pickup_id, pi.category_id, pi.weight_kg, pi.points_awarded, pi.created_at,
		       wc.name, wc.points_per_kg, wc.color_hex
		FROM pickup_items pi
		JOIN waste_categories wc ON wc.id = pi.category_id
		WHERE pi.pickup_id = $1`

	rows, err := r.db.QueryContext(ctx, query, pickupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.PickupItem
	for rows.Next() {
		var item domain.PickupItem
		var cat domain.WasteCategory
		err := rows.Scan(
			&item.ID, &item.PickupID, &item.CategoryID, &item.WeightKg, &item.PointsAwarded, &item.CreatedAt,
			&cat.Name, &cat.PointsPerKg, &cat.ColorHex,
		)
		if err != nil {
			return nil, err
		}
		cat.ID = item.CategoryID
		item.Category = &cat
		items = append(items, item)
	}
	return items, rows.Err()
}

// FindExpiredAssignments mencari pickup yang sudah melewati batas waktu assignment
func (r *PickupRepository) FindExpiredAssignments(ctx context.Context) ([]domain.Pickup, error) {
	query := `
		SELECT id, user_id, collector_id, lat, lon, reassignment_count
		FROM pickups
		WHERE status = 'assigned'
		  AND assignment_timeout < NOW()
		  AND deleted_at IS NULL`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pickups []domain.Pickup
	for rows.Next() {
		var p domain.Pickup
		err := rows.Scan(&p.ID, &p.UserID, &p.CollectorID, &p.Lat, &p.Lon, &p.ReassignmentCount)
		if err != nil {
			return nil, err
		}
		pickups = append(pickups, p)
	}
	return pickups, rows.Err()
}

// FindPendingPickups mencari pickup yang masih menunggu collector
func (r *PickupRepository) FindPendingPickups(ctx context.Context) ([]domain.Pickup, error) {
	query := `
		SELECT id, user_id, collector_id, lat, lon, reassignment_count
		FROM pickups
		WHERE status = 'pending'
		  AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT 10`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pickups []domain.Pickup
	for rows.Next() {
		var p domain.Pickup
		err := rows.Scan(&p.ID, &p.UserID, &p.CollectorID, &p.Lat, &p.Lon, &p.ReassignmentCount)
		if err != nil {
			return nil, err
		}
		pickups = append(pickups, p)
	}
	return pickups, rows.Err()
}

// GetAssignedPickupByCollector mencari pickup yang sedang ditangani collector
func (r *PickupRepository) GetAssignedPickupByCollector(ctx context.Context, collectorID string) (*domain.Pickup, error) {
	query := `
		SELECT p.id, p.user_id, p.collector_id, p.address, p.lat, p.lon,
		       p.photo_url, p.notes, p.status,
		       p.assigned_at, p.assignment_timeout, p.reassignment_count,
		       p.accepted_at, p.started_at, p.arrived_at, p.completed_at,
		       p.total_weight, p.total_points_awarded, p.created_at, p.updated_at,
		       u.name AS user_name, u.phone AS user_phone
		FROM pickups p
		JOIN profiles u ON u.id = p.user_id
		WHERE p.collector_id = $1
		  AND p.status IN ('assigned', 'accepted', 'in_progress', 'arrived')
		  AND p.deleted_at IS NULL
		LIMIT 1`

	var p domain.Pickup
	var userName, userPhone sql.NullString
	err := r.db.QueryRowContext(ctx, query, collectorID).Scan(
		&p.ID, &p.UserID, &p.CollectorID, &p.Address, &p.Lat, &p.Lon,
		&p.PhotoURL, &p.Notes, &p.Status,
		&p.AssignedAt, &p.AssignmentTimeout, &p.ReassignmentCount,
		&p.AcceptedAt, &p.StartedAt, &p.ArrivedAt, &p.CompletedAt,
		&p.TotalWeight, &p.TotalPointsAwarded, &p.CreatedAt, &p.UpdatedAt,
		&userName, &userPhone,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetAssignedPickupByCollector: %w", err)
	}

	p.User = &domain.Profile{ID: p.UserID}
	if userName.Valid {
		p.User.Name = userName.String
	}
	if userPhone.Valid {
		phone := userPhone.String
		p.User.Phone = &phone
	}

	return &p, nil
}

// BeginTx memulai database transaction
func (r *PickupRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

// AdminListPickups mengambil semua pickup untuk admin
func (r *PickupRepository) AdminListPickups(ctx context.Context, status string, limit, offset int) ([]*domain.Pickup, int, error) {
	whereClause := "p.deleted_at IS NULL"
	args := []interface{}{}
	argIdx := 1

	if status != "" {
		whereClause += fmt.Sprintf(" AND p.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	var total int
	err := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM pickups p WHERE %s`, whereClause),
		args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT p.id, p.user_id, p.collector_id, p.address, p.lat, p.lon,
		       p.photo_url, p.notes, p.status,
		       p.assigned_at, p.assignment_timeout, p.reassignment_count,
		       p.accepted_at, p.started_at, p.arrived_at, p.completed_at,
		       p.total_weight, p.total_points_awarded, p.created_at, p.updated_at,
		       u.name AS user_name, c.name AS collector_name
		FROM pickups p
		JOIN profiles u ON u.id = p.user_id
		LEFT JOIN profiles c ON c.id = p.collector_id
		WHERE %s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var pickups []*domain.Pickup
	for rows.Next() {
		var p domain.Pickup
		var userName, collectorName sql.NullString
		err := rows.Scan(
			&p.ID, &p.UserID, &p.CollectorID, &p.Address, &p.Lat, &p.Lon,
			&p.PhotoURL, &p.Notes, &p.Status,
			&p.AssignedAt, &p.AssignmentTimeout, &p.ReassignmentCount,
			&p.AcceptedAt, &p.StartedAt, &p.ArrivedAt, &p.CompletedAt,
			&p.TotalWeight, &p.TotalPointsAwarded, &p.CreatedAt, &p.UpdatedAt,
			&userName, &collectorName,
		)
		if err != nil {
			return nil, 0, err
		}
		if userName.Valid {
			p.User = &domain.Profile{ID: p.UserID, Name: userName.String}
		}
		if p.CollectorID != nil && collectorName.Valid {
			p.Collector = &domain.Profile{ID: *p.CollectorID, Name: collectorName.String}
		}
		pickups = append(pickups, &p)
	}
	return pickups, total, rows.Err()
}

func scanPickup(row *sql.Row) (*domain.Pickup, error) {
	var p domain.Pickup
	err := row.Scan(
		&p.ID, &p.UserID, &p.CollectorID, &p.Address, &p.Lat, &p.Lon,
		&p.PhotoURL, &p.Notes, &p.Status,
		&p.AssignedAt, &p.AssignmentTimeout, &p.ReassignmentCount,
		&p.AcceptedAt, &p.StartedAt, &p.ArrivedAt, &p.CompletedAt,
		&p.TotalWeight, &p.TotalPointsAwarded, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrPickupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan pickup: %w", err)
	}
	return &p, nil
}
