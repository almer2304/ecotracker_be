package handler

import (
	"net/http"

	"ecotracker/internal/middleware"
	"ecotracker/internal/service"
	"ecotracker/internal/utils"

	"github.com/gin-gonic/gin"
)

// ─── Point Log Handler ────────────────────────────────────────────────────────

type PointLogHandler struct {
	pointLogService *service.PointLogService
}

func NewPointLogHandler(s *service.PointLogService) *PointLogHandler {
	return &PointLogHandler{pointLogService: s}
}

// GetMyPointLogs godoc
// GET /api/v1/points/logs
// Role: authenticated
func (h *PointLogHandler) GetMyPointLogs(c *gin.Context) {
	userID := middleware.GetUserID(c)
	logs, err := h.pointLogService.GetMyLogs(c.Request.Context(), userID)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}
	utils.RespondSuccess(c, http.StatusOK, "Point logs retrieved", logs)
}

// ─── Waste Category Handler ───────────────────────────────────────────────────

type WasteCategoryHandler struct {
	categoryService *service.WasteCategoryService
}

func NewWasteCategoryHandler(s *service.WasteCategoryService) *WasteCategoryHandler {
	return &WasteCategoryHandler{categoryService: s}
}

// GetCategories godoc
// GET /api/v1/categories
// Role: authenticated
func (h *WasteCategoryHandler) GetCategories(c *gin.Context) {
	categories, err := h.categoryService.GetAll(c.Request.Context())
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}
	utils.RespondSuccess(c, http.StatusOK, "Waste categories retrieved", categories)
}
