package repository

import (
	"context"
	"errors"
	"fmt"

	"ecotracker/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WasteCategoryRepository struct {
	db *pgxpool.Pool
}

func NewWasteCategoryRepository(db *pgxpool.Pool) *WasteCategoryRepository {
	return &WasteCategoryRepository{db: db}
}

func (r *WasteCategoryRepository) GetAll(ctx context.Context) ([]domain.WasteCategory, error) {
	query := `SELECT id, name, points_per_kg, COALESCE(unit,'kg'), COALESCE(icon_url,'') FROM waste_categories ORDER BY name`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get all waste categories: %w", err)
	}
	defer rows.Close()

	var categories []domain.WasteCategory
	for rows.Next() {
		var c domain.WasteCategory
		if err := rows.Scan(&c.ID, &c.Name, &c.PointsPerKg, &c.Unit, &c.IconURL); err != nil {
			return nil, fmt.Errorf("scan waste category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *WasteCategoryRepository) GetByID(ctx context.Context, id int) (*domain.WasteCategory, error) {
	query := `SELECT id, name, points_per_kg, COALESCE(unit,'kg'), COALESCE(icon_url,'') FROM waste_categories WHERE id=$1`

	var c domain.WasteCategory
	err := r.db.QueryRow(ctx, query, id).Scan(&c.ID, &c.Name, &c.PointsPerKg, &c.Unit, &c.IconURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get waste category by id: %w", err)
	}
	return &c, nil
}
