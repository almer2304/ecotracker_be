package handler

import (
	"net/http"

	"ecotracker/internal/domain"
	"ecotracker/internal/service"
	"ecotracker/internal/utils"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	adminService *service.AdminService
}

func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

// CreateCollector godoc
// POST /api/v1/admin/collectors
// Role: admin
func (h *AdminHandler) CreateCollector(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	// Force role to collector
	req.Role = "collector"

	collector, err := h.adminService.CreateCollector(c.Request.Context(), &req)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}

	utils.RespondSuccess(c, http.StatusCreated, "Collector account created successfully", collector)
}

// ListCollectors godoc
// GET /api/v1/admin/collectors
// Role: admin
func (h *AdminHandler) ListCollectors(c *gin.Context) {
	collectors, err := h.adminService.ListCollectors(c.Request.Context())
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}

	utils.RespondSuccess(c, http.StatusOK, "Collectors retrieved", collectors)
}

// GetDashboardStats godoc
// GET /api/v1/admin/stats
// Role: admin
func (h *AdminHandler) GetDashboardStats(c *gin.Context) {
	stats, err := h.adminService.GetDashboardStats(c.Request.Context())
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}

	utils.RespondSuccess(c, http.StatusOK, "Statistics retrieved", stats)
}

// ListAllPickups godoc
// GET /api/v1/admin/pickups
// Role: admin
// Query params: ?status=pending&limit=50
func (h *AdminHandler) ListAllPickups(c *gin.Context) {
	status := c.Query("status")
	
	pickups, err := h.adminService.ListAllPickups(c.Request.Context(), status)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}

	utils.RespondSuccess(c, http.StatusOK, "Pickups retrieved", pickups)
}

// DeleteCollector godoc
// DELETE /api/v1/admin/collectors/:id
// Role: admin
func (h *AdminHandler) DeleteCollector(c *gin.Context) {
	collectorID := c.Param("id")

	if err := h.adminService.DeleteCollector(c.Request.Context(), collectorID); err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}

	utils.RespondSuccess(c, http.StatusOK, "Collector deleted successfully", nil)
}
