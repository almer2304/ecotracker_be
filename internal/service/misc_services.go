package service

import (
	"context"
	"fmt"

	"ecotracker/internal/domain"
	"ecotracker/internal/repository"
)

// ─── Point Log Service ────────────────────────────────────────────────────────

type PointLogService struct {
	pointLogRepo *repository.PointLogRepository
}

func NewPointLogService(repo *repository.PointLogRepository) *PointLogService {
	return &PointLogService{pointLogRepo: repo}
}

func (s *PointLogService) GetMyLogs(ctx context.Context, userID string) ([]domain.PointLog, error) {
	logs, err := s.pointLogRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get point logs: %w", err)
	}
	if logs == nil {
		logs = []domain.PointLog{}
	}
	return logs, nil
}

// ─── Waste Category Service ───────────────────────────────────────────────────

type WasteCategoryService struct {
	categoryRepo *repository.WasteCategoryRepository
}

func NewWasteCategoryService(repo *repository.WasteCategoryRepository) *WasteCategoryService {
	return &WasteCategoryService{categoryRepo: repo}
}

func (s *WasteCategoryService) GetAll(ctx context.Context) ([]domain.WasteCategory, error) {
	categories, err := s.categoryRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get waste categories: %w", err)
	}
	if categories == nil {
		categories = []domain.WasteCategory{}
	}
	return categories, nil
}
