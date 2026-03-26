package handler

import (
	"net/http"
	"strconv"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/middleware"
	"github.com/ecotracker/backend/internal/service"
	"github.com/ecotracker/backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// CollectorHandler mengelola endpoint khusus collector
type CollectorHandler struct {
	collectorService *service.CollectorService
}

func NewCollectorHandler(collectorService *service.CollectorService) *CollectorHandler {
	return &CollectorHandler{collectorService: collectorService}
}

// UpdateStatus godoc
// PUT /api/v1/collector/status
// Body: {"is_online": true/false}
func (h *CollectorHandler) UpdateStatus(c *gin.Context) {
	collectorID := middleware.GetUserID(c)

	var req domain.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	if err := h.collectorService.UpdateStatus(c.Request.Context(), collectorID, req.IsOnline); err != nil {
		utils.HandleError(c, err)
		return
	}

	status := "offline"
	if req.IsOnline {
		status = "online"
	}
	utils.Success(c, http.StatusOK, "Status berhasil diubah menjadi "+status, gin.H{"is_online": req.IsOnline})
}

// UpdateLocation godoc
// PUT /api/v1/collector/location
// Body: {"lat": -6.2088, "lon": 106.8456}
func (h *CollectorHandler) UpdateLocation(c *gin.Context) {
	collectorID := middleware.GetUserID(c)

	var req domain.UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	if err := h.collectorService.UpdateLocation(c.Request.Context(), collectorID, req.Lat, req.Lon); err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Lokasi diperbarui", gin.H{"lat": req.Lat, "lon": req.Lon})
}

// GetAssignedPickup godoc
// GET /api/v1/collector/assigned
func (h *CollectorHandler) GetAssignedPickup(c *gin.Context) {
	collectorID := middleware.GetUserID(c)

	pickup, err := h.collectorService.GetAssignedPickup(c.Request.Context(), collectorID)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	if pickup == nil {
		utils.Success(c, http.StatusOK, "Tidak ada pickup aktif", nil)
		return
	}

	utils.Success(c, http.StatusOK, "Berhasil", pickup)
}

// AcceptPickup godoc
// POST /api/v1/collector/pickups/:id/accept
func (h *CollectorHandler) AcceptPickup(c *gin.Context) {
	collectorID := middleware.GetUserID(c)
	pickupID := c.Param("id")

	pickup, err := h.collectorService.AcceptPickup(c.Request.Context(), collectorID, pickupID)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Pickup berhasil diterima", pickup)
}

// StartPickup godoc
// POST /api/v1/collector/pickups/:id/start
func (h *CollectorHandler) StartPickup(c *gin.Context) {
	collectorID := middleware.GetUserID(c)
	pickupID := c.Param("id")

	pickup, err := h.collectorService.StartPickup(c.Request.Context(), collectorID, pickupID)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Pickup dimulai, menuju ke lokasi user", pickup)
}

// ArriveAtPickup godoc
// POST /api/v1/collector/pickups/:id/arrive
func (h *CollectorHandler) ArriveAtPickup(c *gin.Context) {
	collectorID := middleware.GetUserID(c)
	pickupID := c.Param("id")

	pickup, err := h.collectorService.ArriveAtPickup(c.Request.Context(), collectorID, pickupID)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Berhasil dicatat, collector telah tiba di lokasi", pickup)
}

// CompletePickup godoc
// POST /api/v1/collector/pickups/:id/complete
// Body: {"items": [{"category_id": "uuid", "weight_kg": 1.5}]}
func (h *CollectorHandler) CompletePickup(c *gin.Context) {
	collectorID := middleware.GetUserID(c)
	pickupID := c.Param("id")

	var req domain.CompletePickupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	pickup, err := h.collectorService.CompletePickup(c.Request.Context(), collectorID, pickupID, &req)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Pickup selesai! Poin telah diberikan ke user", pickup)
}

func (h *CollectorHandler) GetHistory(c *gin.Context) {
	collectorID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
 
	pickups, total, err := h.collectorService.GetHistory(c.Request.Context(), collectorID, page, limit)
	if err != nil {
		utils.HandleError(c, err)
		return
	}
 
	utils.Success(c, http.StatusOK, "Berhasil", utils.BuildListResponse(pickups, total, page, limit))
}