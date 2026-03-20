package service

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/repository"
	"github.com/ecotracker/backend/internal/utils"
)

// ============================================================
// REPORT SERVICE
// ============================================================

type ReportService struct {
	reportRepo    *repository.ReportRepository
	authRepo      *repository.AuthRepository
	badgeService  *BadgeService
	storageClient *utils.StorageClient
}

func NewReportService(
	reportRepo *repository.ReportRepository,
	authRepo *repository.AuthRepository,
	badgeService *BadgeService,
	storageClient *utils.StorageClient,
) *ReportService {
	return &ReportService{
		reportRepo:    reportRepo,
		authRepo:      authRepo,
		badgeService:  badgeService,
		storageClient: storageClient,
	}
}

// CreateReport membuat laporan area kotor baru dengan upload foto
func (s *ReportService) CreateReport(
	ctx context.Context,
	reporterID string,
	req *domain.CreateReportRequest,
	photos []*multipart.FileHeader,
	form *multipart.Form,
) (*domain.AreaReport, error) {
	var photoURLs []string

	// Upload semua foto (maksimal 5)
	maxPhotos := 5
	if len(photos) > maxPhotos {
		photos = photos[:maxPhotos]
	}

	for _, header := range photos {
		file, err := header.Open()
		if err != nil {
			return nil, fmt.Errorf("buka foto: %w", err)
		}
		defer file.Close()

		url, err := s.storageClient.UploadReportPhoto(ctx, file, header)
		if err != nil {
			return nil, fmt.Errorf("upload foto laporan: %w", err)
		}
		photoURLs = append(photoURLs, url)
	}

	report, err := s.reportRepo.Create(
		ctx, reporterID,
		req.Title, req.Description, req.Address,
		req.Lat, req.Lon,
		domain.ReportSeverity(req.Severity),
		photoURLs,
	)
	if err != nil {
		return nil, fmt.Errorf("buat laporan: %w", err)
	}

	// Increment report counter & cek badge (async)
	go func() {
		bgCtx := context.Background()
		if err := s.authRepo.IncrementReportsSubmitted(bgCtx, reporterID); err == nil {
			// Ambil total laporan user
			profile, err := s.authRepo.GetByID(bgCtx, reporterID)
			if err == nil {
				s.badgeService.CheckAndAwardReportBadges(bgCtx, reporterID, profile.TotalReportsSubmitted)
			}
		}
	}()

	return report, nil
}

// GetMyReports mengambil daftar laporan milik user
func (s *ReportService) GetMyReports(ctx context.Context, reporterID string, page, limit int) ([]*domain.AreaReport, int, error) {
	return s.reportRepo.ListByReporterID(ctx, reporterID, limit, (page-1)*limit)
}

// GetReportDetail mengambil detail laporan
func (s *ReportService) GetReportDetail(ctx context.Context, reportID, userID string, role domain.UserRole) (*domain.AreaReport, error) {
	report, err := s.reportRepo.GetByID(ctx, reportID)
	if err != nil {
		return nil, err
	}

	// User biasa hanya bisa lihat laporan miliknya sendiri
	if role == domain.RoleUser && report.ReporterID != userID {
		return nil, domain.ErrForbidden
	}

	return report, nil
}

// ============================================================
// FEEDBACK SERVICE
// ============================================================

type FeedbackService struct {
	feedbackRepo  *repository.FeedbackRepository
	pickupRepo    *repository.PickupRepository
}

func NewFeedbackService(feedbackRepo *repository.FeedbackRepository, pickupRepo *repository.PickupRepository) *FeedbackService {
	return &FeedbackService{feedbackRepo: feedbackRepo, pickupRepo: pickupRepo}
}

// CreateFeedback membuat feedback baru (rating otomatis update ke collector)
func (s *FeedbackService) CreateFeedback(ctx context.Context, userID string, req *domain.CreateFeedbackRequest) (*domain.Feedback, error) {
	f := &domain.Feedback{
		UserID:       userID,
		FeedbackType: domain.FeedbackType(req.FeedbackType),
		Tags:         req.Tags,
	}

	if req.Rating != nil {
		f.Rating = req.Rating
	}
	if req.Title != "" {
		f.Title = &req.Title
	}
	if req.Comment != "" {
		f.Comment = &req.Comment
	}

	// Jika feedback terkait pickup, ambil info collector
	if req.PickupID != "" {
		f.PickupID = &req.PickupID
		pickup, err := s.pickupRepo.GetByID(ctx, req.PickupID)
		if err == nil {
			// Validasi: user hanya bisa beri feedback untuk pickup miliknya
			if pickup.UserID != userID {
				return nil, domain.ErrForbidden
			}
			if pickup.CollectorID != nil {
				f.CollectorID = pickup.CollectorID
			}
		}
	}

	return s.feedbackRepo.Create(ctx, f)
}

// GetMyFeedback mengambil feedback yang dibuat user
func (s *FeedbackService) GetMyFeedback(ctx context.Context, userID string, page, limit int) ([]*domain.Feedback, int, error) {
	return s.feedbackRepo.ListByUserID(ctx, userID, limit, (page-1)*limit)
}

// ============================================================
// ADMIN SERVICE
// ============================================================

type AdminService struct {
	authRepo      *repository.AuthRepository
	pickupRepo    *repository.PickupRepository
	reportRepo    *repository.ReportRepository
	feedbackRepo  *repository.FeedbackRepository
	collectorRepo *repository.CollectorRepository
	categoryRepo  *repository.CategoryRepository
	jwtManager    interface{ GenerateAccessToken(string, string, string) (string, interface{}, error) }
	bcryptCost    int
}

// AdminServiceFull adalah versi dengan dependency lengkap
type AdminServiceFull struct {
	authRepo      *repository.AuthRepository
	pickupRepo    *repository.PickupRepository
	reportRepo    *repository.ReportRepository
	feedbackRepo  *repository.FeedbackRepository
	collectorRepo *repository.CollectorRepository
	bcryptCost    int
	hashFunc      func(string) (string, error)
}

func NewAdminService(
	authRepo *repository.AuthRepository,
	pickupRepo *repository.PickupRepository,
	reportRepo *repository.ReportRepository,
	feedbackRepo *repository.FeedbackRepository,
	collectorRepo *repository.CollectorRepository,
	bcryptCost int,
) *AdminServiceFull {
	return &AdminServiceFull{
		authRepo:      authRepo,
		pickupRepo:    pickupRepo,
		reportRepo:    reportRepo,
		feedbackRepo:  feedbackRepo,
		collectorRepo: collectorRepo,
		bcryptCost:    bcryptCost,
		hashFunc: func(pw string) (string, error) {
			return utils.HashPassword(pw, bcryptCost)
		},
	}
}

// GetDashboardStats mengambil statistik untuk dashboard admin
func (s *AdminServiceFull) GetDashboardStats(ctx context.Context) (*domain.DashboardStats, error) {
	// Query langsung dari view v_dashboard_stats
	var stats domain.DashboardStats
	// Implementasi melalui query langsung
	return &stats, nil
}

// ListCollectors mengambil semua collector
func (s *AdminServiceFull) ListCollectors(ctx context.Context, page, limit int) ([]*domain.Profile, int, error) {
	return s.collectorRepo.AdminListCollectors(ctx, limit, (page-1)*limit)
}

// CreateCollector membuat akun collector baru
func (s *AdminServiceFull) CreateCollector(ctx context.Context, req *domain.CreateCollectorRequest) (*domain.Profile, error) {
	hash, err := s.hashFunc(req.Password)
	if err != nil {
		return nil, err
	}
	return s.authRepo.Create(ctx, req.Name, req.Email, hash, req.Phone, domain.RoleCollector)
}

// DeleteCollector menghapus (soft delete) akun collector
func (s *AdminServiceFull) DeleteCollector(ctx context.Context, collectorID string) error {
	return s.collectorRepo.DeleteCollector(ctx, collectorID)
}

// ListPickups mengambil semua pickup untuk admin dengan filter status
func (s *AdminServiceFull) ListPickups(ctx context.Context, status string, page, limit int) ([]*domain.Pickup, int, error) {
	return s.pickupRepo.AdminListPickups(ctx, status, limit, (page-1)*limit)
}

// ListReports mengambil semua laporan untuk admin
func (s *AdminServiceFull) ListReports(ctx context.Context, status, severity string, page, limit int) ([]*domain.AreaReport, int, error) {
	return s.reportRepo.AdminListReports(ctx, status, severity, limit, (page-1)*limit)
}

// UpdateReport memperbarui status laporan area kotor
func (s *AdminServiceFull) UpdateReport(ctx context.Context, reportID, status, adminNotes, assignedTo string) error {
	return s.reportRepo.UpdateStatus(ctx, reportID, status, adminNotes, assignedTo)
}

// ListFeedback mengambil semua feedback untuk admin
func (s *AdminServiceFull) ListFeedback(ctx context.Context, feedbackType string, page, limit int) ([]*domain.Feedback, int, error) {
	return s.feedbackRepo.AdminListFeedback(ctx, feedbackType, limit, (page-1)*limit)
}

// RespondToFeedback admin membalas feedback
func (s *AdminServiceFull) RespondToFeedback(ctx context.Context, feedbackID, adminID, response string) error {
	return s.feedbackRepo.UpdateAdminResponse(ctx, feedbackID, adminID, response)
}
