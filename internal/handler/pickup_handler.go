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
// Role: user or collector (depending on ownership/assignment)
func (h *PickupHandler) GetPickupDetail(c *gin.Context) {
	pickupID := c.Param("id")
	
	detail, err := h.pickupService.GetPickupDetail(c.Request.Context(), pickupID)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}
	utils.RespondSuccess(c, http.StatusOK, "Pickup detail retrieved", detail)
}

// GetPendingPickups godoc
// GET /api/v1/collector/pickups/pending
// Role: collector
// Query params: ?lat=<latitude>&lon=<longitude> (optional)
func (h *PickupHandler) GetPendingPickups(c *gin.Context) {
	// Check for GPS coordinates in query params
	latStr := c.Query("lat")
	lonStr := c.Query("lon")

	// If GPS coordinates provided, return sorted by distance
	if latStr != "" && lonStr != "" {
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid latitude: %w", err))
			return
		}
		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid longitude: %w", err))
			return
		}

		// Use Haversine-based sorting
		pickups, err := h.pickupService.ListPendingPickupsNearby(c.Request.Context(), lat, lon)
		if err != nil {
			utils.RespondWithDomainError(c, err)
			return
		}
		utils.RespondSuccess(c, http.StatusOK, "Pending pickups near you", pickups)
		return
	}

	// Otherwise, return all pending pickups without sorting
	pickups, err := h.pickupService.ListPendingPickups(c.Request.Context())
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}
	utils.RespondSuccess(c, http.StatusOK, "All pending pickups", pickups)
}

// GetMyTasks godoc
// GET /api/v1/collector/pickups/my-tasks
// Role: collector
func (h *PickupHandler) GetMyTasks(c *gin.Context) {
	collectorID := middleware.GetUserID(c)
	pickups, err := h.pickupService.ListCollectorTasks(c.Request.Context(), collectorID)
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

	pickup, err := h.pickupService.TakePickup(c.Request.Context(), pickupID, collectorID)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}

	utils.RespondSuccess(c, http.StatusOK, "Pickup task taken successfully", pickup)
}

// CompleteTask godoc
// POST /api/v1/collector/pickups/:id/complete
// Role: collector
// Body: { "items": [{"category_id": 1, "weight": 5.5}, ...] }
func (h *PickupHandler) CompleteTask(c *gin.Context) {
	pickupID := c.Param("id")
	collectorID := middleware.GetUserID(c)

	var req domain.CompletePickupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	pickup, totalPoints, err := h.pickupService.CompletePickup(c.Request.Context(), pickupID, collectorID, req.Items)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}

	result := map[string]interface{}{
		"pickup":       pickup,
		"total_points": totalPoints,
	}
	utils.RespondSuccess(c, http.StatusOK, "Pickup completed and points awarded", result)
}