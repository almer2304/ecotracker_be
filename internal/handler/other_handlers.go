package handler

import (
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/middleware"
	"github.com/ecotracker/backend/internal/service"
	"github.com/ecotracker/backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// ============================================================
// BADGE HANDLER
// ============================================================

type BadgeHandler struct {
	badgeService *service.BadgeService
}

func NewBadgeHandler(badgeService *service.BadgeService) *BadgeHandler {
	return &BadgeHandler{badgeService: badgeService}
}

// GetAllBadges godoc
// GET /api/v1/badges
func (h *BadgeHandler) GetAllBadges(c *gin.Context) {
	badges, err := h.badgeService.GetAllBadges(c.Request.Context())
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Berhasil", badges)
}

// GetMyBadges godoc
// GET /api/v1/badges/my
func (h *BadgeHandler) GetMyBadges(c *gin.Context) {
	userID := middleware.GetUserID(c)
	badges, err := h.badgeService.GetUserBadges(c.Request.Context(), userID)
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Berhasil", badges)
}

// ============================================================
// REPORT HANDLER
// ============================================================

type ReportHandler struct {
	reportService *service.ReportService
}

func NewReportHandler(reportService *service.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

// CreateReport godoc
// POST /api/v1/reports
// Content-Type: multipart/form-data
func (h *ReportHandler) CreateReport(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req domain.CreateReportRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	// Ambil foto-foto (bisa lebih dari satu, field name: "photos")
	form, _ := c.MultipartForm()
	var photos []*multipart.FileHeader
	if form != nil {
		photos = form.File["photos"]
	}

	report, err := h.reportService.CreateReport(c.Request.Context(), userID, &req, photos, form)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "Laporan berhasil dikirim", report)
}

// GetMyReports godoc
// GET /api/v1/reports/my
func (h *ReportHandler) GetMyReports(c *gin.Context) {
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	reports, total, err := h.reportService.GetMyReports(c.Request.Context(), userID, page, limit)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Berhasil", utils.BuildListResponse(reports, total, page, limit))
}

// GetReportDetail godoc
// GET /api/v1/reports/:id
func (h *ReportHandler) GetReportDetail(c *gin.Context) {
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)
	reportID := c.Param("id")

	report, err := h.reportService.GetReportDetail(c.Request.Context(), reportID, userID, role)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Berhasil", report)
}

// ============================================================
// FEEDBACK HANDLER
// ============================================================

type FeedbackHandler struct {
	feedbackService *service.FeedbackService
}

func NewFeedbackHandler(feedbackService *service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{feedbackService: feedbackService}
}

// CreateFeedback godoc
// POST /api/v1/feedback
func (h *FeedbackHandler) CreateFeedback(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req domain.CreateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	feedback, err := h.feedbackService.CreateFeedback(c.Request.Context(), userID, &req)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "Feedback berhasil dikirim, terima kasih!", feedback)
}

// GetMyFeedback godoc
// GET /api/v1/feedback/my
func (h *FeedbackHandler) GetMyFeedback(c *gin.Context) {
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	feedbacks, total, err := h.feedbackService.GetMyFeedback(c.Request.Context(), userID, page, limit)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Berhasil", utils.BuildListResponse(feedbacks, total, page, limit))
}

// ============================================================
// ADMIN HANDLER
// ============================================================

type AdminHandler struct {
	adminService *service.AdminServiceFull
}

func NewAdminHandler(adminService *service.AdminServiceFull) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

// GetDashboard godoc
// GET /api/v1/admin/dashboard
func (h *AdminHandler) GetDashboard(c *gin.Context) {
	stats, err := h.adminService.GetDashboardStats(c.Request.Context())
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Berhasil", stats)
}

// ListCollectors godoc
// GET /api/v1/admin/collectors
func (h *AdminHandler) ListCollectors(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	collectors, total, err := h.adminService.ListCollectors(c.Request.Context(), page, limit)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Berhasil", utils.BuildListResponse(collectors, total, page, limit))
}

// CreateCollector godoc
// POST /api/v1/admin/collectors
func (h *AdminHandler) CreateCollector(c *gin.Context) {
	var req domain.CreateCollectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	collector, err := h.adminService.CreateCollector(c.Request.Context(), &req)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "Collector berhasil dibuat", collector)
}

// DeleteCollector godoc
// DELETE /api/v1/admin/collectors/:id
func (h *AdminHandler) DeleteCollector(c *gin.Context) {
	collectorID := c.Param("id")

	if err := h.adminService.DeleteCollector(c.Request.Context(), collectorID); err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Collector berhasil dihapus", nil)
}

// ListPickups godoc
// GET /api/v1/admin/pickups?status=pending
func (h *AdminHandler) ListPickups(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	pickups, total, err := h.adminService.ListPickups(c.Request.Context(), status, page, limit)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Berhasil", utils.BuildListResponse(pickups, total, page, limit))
}

// ListReports godoc
// GET /api/v1/admin/reports?status=new&severity=high
func (h *AdminHandler) ListReports(c *gin.Context) {
	status := c.Query("status")
	severity := c.Query("severity")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	reports, total, err := h.adminService.ListReports(c.Request.Context(), status, severity, page, limit)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Berhasil", utils.BuildListResponse(reports, total, page, limit))
}

// UpdateReport godoc
// PUT /api/v1/admin/reports/:id
func (h *AdminHandler) UpdateReport(c *gin.Context) {
	reportID := c.Param("id")

	var req domain.UpdateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	if err := h.adminService.UpdateReport(c.Request.Context(), reportID, req.Status, req.AdminNotes, req.AssignedTo); err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Laporan berhasil diperbarui", nil)
}

// ListFeedback godoc
// GET /api/v1/admin/feedback?type=collector
func (h *AdminHandler) ListFeedback(c *gin.Context) {
	feedbackType := c.Query("type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	feedbacks, total, err := h.adminService.ListFeedback(c.Request.Context(), feedbackType, page, limit)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Berhasil", utils.BuildListResponse(feedbacks, total, page, limit))
}

// RespondToFeedback godoc
// PUT /api/v1/admin/feedback/:id/respond
func (h *AdminHandler) RespondToFeedback(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	feedbackID := c.Param("id")

	var req domain.AdminFeedbackResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	if err := h.adminService.RespondToFeedback(c.Request.Context(), feedbackID, adminID, req.Response); err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Respons berhasil disimpan", nil)
}


