package repository

import (
	"context"
	"fmt"

	"ecotracker/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PointLogRepository struct {
	db *pgxpool.Pool
}

func NewPointLogRepository(db *pgxpool.Pool) *PointLogRepository {
	return &PointLogRepository{db: db}
}

func (r *PointLogRepository) GetByUserID(ctx context.Context, userID string) ([]domain.PointLog, error) {
	query := `
		SELECT id, user_id, amount, transaction_type,
		       COALESCE(reference_id::text,''), COALESCE(description,''), created_at
		FROM point_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 100`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get point logs: %w", err)
	}
	defer rows.Close()

	var logs []domain.PointLog
	for rows.Next() {
		var l domain.PointLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.Amount, &l.TransactionType, &l.ReferenceID, &l.Description, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan point log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}
