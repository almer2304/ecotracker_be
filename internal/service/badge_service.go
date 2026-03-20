package service

import (
	"context"
	"fmt"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/repository"
	"github.com/sirupsen/logrus"
)

// BadgeService mengelola sistem badge/achievement
type BadgeService struct {
	badgeRepo *repository.BadgeRepository
}

func NewBadgeService(badgeRepo *repository.BadgeRepository) *BadgeService {
	return &BadgeService{badgeRepo: badgeRepo}
}

// GetAllBadges mengambil semua definisi badge
func (s *BadgeService) GetAllBadges(ctx context.Context) ([]domain.Badge, error) {
	return s.badgeRepo.GetAll(ctx)
}

// GetUserBadges mengambil badge dengan status locked/unlocked untuk user tertentu
func (s *BadgeService) GetUserBadges(ctx context.Context, userID string) ([]domain.Badge, error) {
	badges, err := s.badgeRepo.GetUserBadges(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Tambahkan info progress untuk badge yang belum unlock
	// (progress diisi berdasarkan data yang sudah ada di struct user)
	return badges, nil
}

// CheckAndAwardBadges mengecek kriteria dan memberikan badge yang memenuhi syarat
// Dipanggil setelah pickup selesai
func (s *BadgeService) CheckAndAwardBadges(ctx context.Context, userID string, totalPickups, totalPoints int) error {
	// Ambil semua badge
	badges, err := s.badgeRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("ambil badge: %w", err)
	}

	for _, badge := range badges {
		// Cek apakah kriteria terpenuhi
		met := false
		switch badge.CriteriaType {
		case "pickups":
			met = totalPickups >= badge.CriteriaValue
		case "points":
			met = totalPoints >= badge.CriteriaValue
		// "reports" dihandle terpisah saat submit report
		}

		if !met {
			continue
		}

		// Cek apakah sudah pernah diberikan
		has, err := s.badgeRepo.HasBadge(ctx, userID, badge.ID)
		if err != nil {
			logrus.WithError(err).Warnf("Gagal cek badge %s untuk user %s", badge.Code, userID)
			continue
		}

		if has {
			continue
		}

		// Berikan badge
		if err := s.badgeRepo.AwardBadge(ctx, userID, badge.ID); err != nil {
			logrus.WithError(err).Warnf("Gagal berikan badge %s ke user %s", badge.Code, userID)
			continue
		}

		logrus.WithFields(logrus.Fields{
			"user_id":    userID,
			"badge_code": badge.Code,
		}).Info("Badge berhasil diberikan")
	}

	return nil
}

// CheckAndAwardReportBadges mengecek badge terkait pelaporan area kotor
func (s *BadgeService) CheckAndAwardReportBadges(ctx context.Context, userID string, totalReports int) error {
	badges, err := s.badgeRepo.GetAll(ctx)
	if err != nil {
		return err
	}

	for _, badge := range badges {
		if badge.CriteriaType != "reports" {
			continue
		}

		if totalReports < badge.CriteriaValue {
			continue
		}

		has, _ := s.badgeRepo.HasBadge(ctx, userID, badge.ID)
		if has {
			continue
		}

		if err := s.badgeRepo.AwardBadge(ctx, userID, badge.ID); err != nil {
			logrus.WithError(err).Warnf("Gagal berikan badge %s", badge.Code)
		} else {
			logrus.Infof("Badge %s diberikan ke user %s (total reports: %d)", badge.Code, userID, totalReports)
		}
	}

	return nil
}
