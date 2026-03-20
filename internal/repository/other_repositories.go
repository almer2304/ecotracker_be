package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/lib/pq"
)

// ============================================================
// COLLECTOR REPOSITORY
// ============================================================

type CollectorRepository struct {
	db *sql.DB
}

func NewCollectorRepository(db *sql.DB) *CollectorRepository {
	return &CollectorRepository{db: db}
}

// FindAvailable mencari semua collector online yang tidak sedang busy
// dan memiliki lokasi terbaru (dalam 30 menit terakhir)
func (r *CollectorRepository) FindAvailable(ctx context.Context, excludeIDs []string) ([]domain.Profile, error) {
	query := `
		SELECT id, last_lat, last_lon, name, average_rating
		FROM v_available_collectors`

	args := []interface{}{}
	if len(excludeIDs) > 0 {
		query += ` WHERE id != ALL($1)`
		args = append(args, pq.Array(excludeIDs))
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("FindAvailable: %w", err)
	}
	defer rows.Close()

	var collectors []domain.Profile
	for rows.Next() {
		var c domain.Profile
		err := rows.Scan(&c.ID, &c.LastLat, &c.LastLon, &c.Name, &c.AverageRating)
		if err != nil {
			return nil, err
		}
		collectors = append(collectors, c)
	}
	return collectors, rows.Err()
}

// UpdateBusyStatus mengubah status busy collector
func (r *CollectorRepository) UpdateBusyStatus(ctx context.Context, collectorID string, isBusy bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE profiles SET is_busy = $1 WHERE id = $2`,
		isBusy, collectorID,
	)
	return err
}

// AdminListCollectors mengambil daftar collector untuk admin
func (r *CollectorRepository) AdminListCollectors(ctx context.Context, limit, offset int) ([]*domain.Profile, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM profiles WHERE role = 'collector' AND deleted_at IS NULL`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, email, phone, is_online, is_busy,
		       average_rating, total_ratings, total_weight_collected,
		       total_pickups_completed, created_at, updated_at
		FROM profiles
		WHERE role = 'collector' AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var collectors []*domain.Profile
	for rows.Next() {
		var c domain.Profile
		err := rows.Scan(
			&c.ID, &c.Name, &c.Email, &c.Phone, &c.IsOnline, &c.IsBusy,
			&c.AverageRating, &c.TotalRatings, &c.TotalWeightCollected,
			&c.TotalPickupsCompleted, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		collectors = append(collectors, &c)
	}
	return collectors, total, rows.Err()
}

// DeleteCollector soft delete collector
func (r *CollectorRepository) DeleteCollector(ctx context.Context, collectorID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE profiles SET deleted_at = NOW() WHERE id = $1 AND role = 'collector'`,
		collectorID,
	)
	return err
}

// ============================================================
// BADGE REPOSITORY
// ============================================================

type BadgeRepository struct {
	db *sql.DB
}

func NewBadgeRepository(db *sql.DB) *BadgeRepository {
	return &BadgeRepository{db: db}
}

func (r *BadgeRepository) GetAll(ctx context.Context) ([]domain.Badge, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, code, name, description, icon_url, color_hex,
		       criteria_type, criteria_value, display_order, created_at
		FROM badges ORDER BY display_order ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var badges []domain.Badge
	for rows.Next() {
		var b domain.Badge
		err := rows.Scan(
			&b.ID, &b.Code, &b.Name, &b.Description, &b.IconURL, &b.ColorHex,
			&b.CriteriaType, &b.CriteriaValue, &b.DisplayOrder, &b.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		badges = append(badges, b)
	}
	return badges, rows.Err()
}

// GetUserBadges mengambil semua badge dengan status locked/unlocked untuk user tertentu
func (r *BadgeRepository) GetUserBadges(ctx context.Context, userID string) ([]domain.Badge, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT b.id, b.code, b.name, b.description, b.icon_url, b.color_hex,
		       b.criteria_type, b.criteria_value, b.display_order, b.created_at,
		       ub.awarded_at
		FROM badges b
		LEFT JOIN user_badges ub ON ub.badge_id = b.id AND ub.user_id = $1
		ORDER BY b.display_order ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var badges []domain.Badge
	for rows.Next() {
		var b domain.Badge
		var awardedAt sql.NullTime
		err := rows.Scan(
			&b.ID, &b.Code, &b.Name, &b.Description, &b.IconURL, &b.ColorHex,
			&b.CriteriaType, &b.CriteriaValue, &b.DisplayOrder, &b.CreatedAt,
			&awardedAt,
		)
		if err != nil {
			return nil, err
		}

		unlocked := awardedAt.Valid
		b.IsUnlocked = &unlocked
		if awardedAt.Valid {
			b.AwardedAt = &awardedAt.Time
		}
		badges = append(badges, b)
	}
	return badges, rows.Err()
}

// HasBadge mengecek apakah user sudah memiliki badge tertentu
func (r *BadgeRepository) HasBadge(ctx context.Context, userID, badgeID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_badges WHERE user_id = $1 AND badge_id = $2`,
		userID, badgeID,
	).Scan(&count)
	return count > 0, err
}

// AwardBadge memberikan badge kepada user
func (r *BadgeRepository) AwardBadge(ctx context.Context, userID, badgeID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_badges (user_id, badge_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, badgeID,
	)
	return err
}

// ============================================================
// REPORT REPOSITORY
// ============================================================

type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Create(ctx context.Context, reporterID, title, description, address string, lat, lon float64, severity domain.ReportSeverity, photoURLs []string) (*domain.AreaReport, error) {
	var report domain.AreaReport
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO area_reports (reporter_id, title, description, address, lat, lon, severity, photo_urls)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, reporter_id, assigned_to, title, description, address, lat, lon,
		          severity, status, photo_urls, admin_notes, resolved_at, created_at, updated_at`,
		reporterID, title, description, address, lat, lon, severity, pq.Array(photoURLs),
	).Scan(
		&report.ID, &report.ReporterID, &report.AssignedTo, &report.Title, &report.Description,
		&report.Address, &report.Lat, &report.Lon, &report.Severity, &report.Status,
		pq.Array(&report.PhotoURLs), &report.AdminNotes, &report.ResolvedAt,
		&report.CreatedAt, &report.UpdatedAt,
	)
	return &report, err
}

func (r *ReportRepository) GetByID(ctx context.Context, id string) (*domain.AreaReport, error) {
	var report domain.AreaReport
	err := r.db.QueryRowContext(ctx, `
		SELECT ar.id, ar.reporter_id, ar.assigned_to, ar.title, ar.description,
		       ar.address, ar.lat, ar.lon, ar.severity, ar.status,
		       ar.photo_urls, ar.admin_notes, ar.resolved_at, ar.created_at, ar.updated_at,
		       p.name AS reporter_name
		FROM area_reports ar
		JOIN profiles p ON p.id = ar.reporter_id
		WHERE ar.id = $1 AND ar.deleted_at IS NULL`, id,
	).Scan(
		&report.ID, &report.ReporterID, &report.AssignedTo, &report.Title, &report.Description,
		&report.Address, &report.Lat, &report.Lon, &report.Severity, &report.Status,
		pq.Array(&report.PhotoURLs), &report.AdminNotes, &report.ResolvedAt,
		&report.CreatedAt, &report.UpdatedAt,
		new(string), // reporter_name - populate below
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrReportNotFound
	}
	return &report, err
}

func (r *ReportRepository) ListByReporterID(ctx context.Context, reporterID string, limit, offset int) ([]*domain.AreaReport, int, error) {
	var total int
	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM area_reports WHERE reporter_id = $1 AND deleted_at IS NULL`, reporterID,
	).Scan(&total)

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, reporter_id, assigned_to, title, description, address, lat, lon,
		       severity, status, photo_urls, admin_notes, resolved_at, created_at, updated_at
		FROM area_reports
		WHERE reporter_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, reporterID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanReports(rows, total)
}

func (r *ReportRepository) AdminListReports(ctx context.Context, status, severity string, limit, offset int) ([]*domain.AreaReport, int, error) {
	where := "deleted_at IS NULL"
	args := []interface{}{}
	i := 1

	if status != "" {
		where += fmt.Sprintf(" AND status=$%d", i)
		args = append(args, status)
		i++
	}
	if severity != "" {
		where += fmt.Sprintf(" AND severity=$%d", i)
		args = append(args, severity)
		i++
	}

	var total int
	r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM area_reports WHERE %s`, where), args...,
	).Scan(&total)

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, reporter_id, assigned_to, title, description, address, lat, lon,
		       severity, status, photo_urls, admin_notes, resolved_at, created_at, updated_at
		FROM area_reports WHERE %s
		ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, i, i+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanReports(rows, total)
}

func (r *ReportRepository) UpdateStatus(ctx context.Context, reportID, status, adminNotes, assignedTo string) error {
	query := `UPDATE area_reports SET status = $1, admin_notes = $2, updated_at = NOW()`
	args := []interface{}{status, nullStr(adminNotes)}
	i := 3

	if assignedTo != "" {
		query += fmt.Sprintf(", assigned_to = $%d", i)
		args = append(args, assignedTo)
		i++
	}
	if status == "resolved" {
		query += ", resolved_at = NOW()"
	}
	query += fmt.Sprintf(" WHERE id = $%d", i)
	args = append(args, reportID)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func scanReports(rows *sql.Rows, total int) ([]*domain.AreaReport, int, error) {
	var reports []*domain.AreaReport
	for rows.Next() {
		var ar domain.AreaReport
		err := rows.Scan(
			&ar.ID, &ar.ReporterID, &ar.AssignedTo, &ar.Title, &ar.Description,
			&ar.Address, &ar.Lat, &ar.Lon, &ar.Severity, &ar.Status,
			pq.Array(&ar.PhotoURLs), &ar.AdminNotes, &ar.ResolvedAt,
			&ar.CreatedAt, &ar.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		reports = append(reports, &ar)
	}
	return reports, total, rows.Err()
}

// ============================================================
// FEEDBACK REPOSITORY
// ============================================================

type FeedbackRepository struct {
	db *sql.DB
}

func NewFeedbackRepository(db *sql.DB) *FeedbackRepository {
	return &FeedbackRepository{db: db}
}

func (r *FeedbackRepository) Create(ctx context.Context, f *domain.Feedback) (*domain.Feedback, error) {
	var created domain.Feedback
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO feedback (user_id, pickup_id, collector_id, feedback_type, rating, title, comment, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, pickup_id, collector_id, feedback_type, rating,
		          title, comment, tags, created_at, updated_at`,
		f.UserID, f.PickupID, f.CollectorID, f.FeedbackType, f.Rating,
		f.Title, f.Comment, pq.Array(f.Tags),
	).Scan(
		&created.ID, &created.UserID, &created.PickupID, &created.CollectorID,
		&created.FeedbackType, &created.Rating, &created.Title, &created.Comment,
		pq.Array(&created.Tags), &created.CreatedAt, &created.UpdatedAt,
	)
	return &created, err
}

func (r *FeedbackRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Feedback, int, error) {
	var total int
	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM feedback WHERE user_id = $1`, userID,
	).Scan(&total)

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, pickup_id, collector_id, feedback_type, rating,
		       title, comment, tags, admin_response, responded_at, created_at, updated_at
		FROM feedback WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanFeedbacks(rows, total)
}

func (r *FeedbackRepository) AdminListFeedback(ctx context.Context, feedbackType string, limit, offset int) ([]*domain.Feedback, int, error) {
	where := "1=1"
	args := []interface{}{}
	i := 1

	if feedbackType != "" {
		where += fmt.Sprintf(" AND feedback_type=$%d", i)
		args = append(args, feedbackType)
		i++
	}

	var total int
	r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM feedback WHERE %s`, where), args...,
	).Scan(&total)

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, user_id, pickup_id, collector_id, feedback_type, rating,
		       title, comment, tags, admin_response, responded_at, created_at, updated_at
		FROM feedback WHERE %s
		ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, i, i+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanFeedbacks(rows, total)
}

func (r *FeedbackRepository) UpdateAdminResponse(ctx context.Context, feedbackID, adminID, response string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE feedback SET admin_response=$1, responded_by=$2, responded_at=NOW()
		WHERE id=$3`, response, adminID, feedbackID)
	return err
}

func scanFeedbacks(rows *sql.Rows, total int) ([]*domain.Feedback, int, error) {
	var feedbacks []*domain.Feedback
	for rows.Next() {
		var f domain.Feedback
		err := rows.Scan(
			&f.ID, &f.UserID, &f.PickupID, &f.CollectorID, &f.FeedbackType,
			&f.Rating, &f.Title, &f.Comment, pq.Array(&f.Tags),
			&f.AdminResponse, &f.RespondedAt, &f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		feedbacks = append(feedbacks, &f)
	}
	return feedbacks, total, rows.Err()
}

// ============================================================
// CATEGORY REPOSITORY
// ============================================================

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) GetAll(ctx context.Context) ([]domain.WasteCategory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, points_per_kg, icon_url, color_hex, is_active, created_at
		FROM waste_categories WHERE is_active = true ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []domain.WasteCategory
	for rows.Next() {
		var c domain.WasteCategory
		err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.PointsPerKg, &c.IconURL, &c.ColorHex, &c.IsActive, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (r *CategoryRepository) GetByID(ctx context.Context, id string) (*domain.WasteCategory, error) {
	var c domain.WasteCategory
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, points_per_kg, icon_url, color_hex, is_active, created_at
		FROM waste_categories WHERE id = $1 AND is_active = true`, id,
	).Scan(&c.ID, &c.Name, &c.Description, &c.PointsPerKg, &c.IconURL, &c.ColorHex, &c.IsActive, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrCategoryNotFound
	}
	return &c, err
}

// ============================================================
// POINT LOG REPOSITORY
// ============================================================

type PointLogRepository struct {
	db *sql.DB
}

func NewPointLogRepository(db *sql.DB) *PointLogRepository {
	return &PointLogRepository{db: db}
}

func (r *PointLogRepository) Create(ctx context.Context, tx *sql.Tx, userID string, pickupID *string, logType domain.PointLogType, points int, description string, balanceAfter int) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO point_logs (user_id, pickup_id, log_type, points, description, balance_after)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, pickupID, logType, points, nullStr(description), balanceAfter,
	)
	return err
}
