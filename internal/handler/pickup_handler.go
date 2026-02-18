package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"ecotracker/internal/domain"
	"ecotracker/internal/middleware"
	"ecotracker/internal/service"
	"ecotracker/internal/utils"

	"github.com/gin-gonic/gin"
)

type PickupHandler struct {
	pickupService *service.PickupService
}

func NewPickupHandler(pickupService *service.PickupService) *PickupHandler {
	return &PickupHandler{pickupService: pickupService}
}

// CreatePickup godoc
// POST /api/v1/pickups  (multipart/form-data)
// Role: user
func (h *PickupHandler) CreatePickup(c *gin.Context) {
	var req domain.CreatePickupRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	// Photo is optional
	fileHeader, _ := c.FormFile("photo")

	userID := middleware.GetUserID(c)
	pickup, err := h.pickupService.CreatePickup(c.Request.Context(), userID, &req, fileHeader)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}

	utils.RespondSuccess(c, http.StatusCreated, "Pickup request created successfully", pickup)
}

// GetMyPickups godoc
// GET /api/v1/pickups/my
// Role: user
func (h *PickupHandler) GetMyPickups(c *gin.Context) {
	userID := middleware.GetUserID(c)
	pickups, err := h.pickupService.ListMyPickups(c.Request.Context(), userID)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}
	utils.RespondSuccess(c, http.StatusOK, "My pickups retrieved", pickups)
}

// GetPickupDetail godoc
// GET /api/v1/pickups/:id
// Role: user (own only), collector (all)
func (h *PickupHandler) GetPickupDetail(c *gin.Context) {
	pickupID := c.Param("id")
	userID := middleware.GetUserID(c)
	userRole := middleware.GetUserRole(c)

	detail, err := h.pickupService.GetPickupDetail(c.Request.Context(), pickupID, userID, userRole)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}
	utils.RespondSuccess(c, http.StatusOK, "Pickup detail retrieved", detail)
}

// ─── Collector Endpoints ──────────────────────────────────────────────────────

// GetPendingPickups godoc
// GET /api/v1/collector/pickups/pending
// Query params: ?lat=xxx&lon=xxx (optional, for distance sorting)
// Role: collector
func (h *PickupHandler) GetPendingPickups(c *gin.Context) {
	latStr := c.Query("lat")
	lonStr := c.Query("lon")

	// If location provided, return sorted by distance
	if latStr != "" && lonStr != "" {
		lat, errLat := strconv.ParseFloat(latStr, 64)
		lon, errLon := strconv.ParseFloat(lonStr, 64)

		if errLat != nil || errLon != nil {
			utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid latitude or longitude"))
			return
		}

		pickups, err := h.pickupService.ListPendingPickupsNearby(c.Request.Context(), lat, lon)
		if err != nil {
			utils.RespondWithDomainError(c, err)
			return
		}
		utils.RespondSuccess(c, http.StatusOK, "Nearby pending pickups retrieved (sorted by distance)", pickups)
		return
	}

	// Default: return all without distance
	pickups, err := h.pickupService.ListPendingPickups(c.Request.Context())
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}
	utils.RespondSuccess(c, http.StatusOK, "Pending pickups retrieved", pickups)
}

// GetMyTasks godoc
// GET /api/v1/collector/pickups/my-tasks
// Role: collector
func (h *PickupHandler) GetMyTasks(c *gin.Context) {
	collectorID := middleware.GetUserID(c)
	pickups, err := h.pickupService.ListMyTasks(c.Request.Context(), collectorID)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}
	utils.RespondSuccess(c, http.StatusOK, "My tasks retrieved", pickups)
}

// TakeTask godoc
// POST /api/v1/collector/pickups/:id/take
// Role: collector
func (h *PickupHandler) TakeTask(c *gin.Context) {
	pickupID := c.Param("id")
	collectorID := middleware.GetUserID(c)

	pickup, err := h.pickupService.TakeTask(c.Request.Context(), pickupID, collectorID)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}
	utils.RespondSuccess(c, http.StatusOK, "Task taken successfully", pickup)
}

// CompleteTask godoc
// POST /api/v1/collector/pickups/:id/complete
// Role: collector
func (h *PickupHandler) CompleteTask(c *gin.Context) {
	pickupID := c.Param("id")
	collectorID := middleware.GetUserID(c)

	var req domain.CompletePickupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	detail, err := h.pickupService.CompleteTask(c.Request.Context(), pickupID, collectorID, &req)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}
	utils.RespondSuccess(c, http.StatusOK, "Pickup completed. Points awarded successfully", detail)
}
