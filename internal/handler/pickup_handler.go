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

// PickupHandler mengelola endpoint pickup
type PickupHandler struct {
	pickupService *service.PickupService
}

func NewPickupHandler(pickupService *service.PickupService) *PickupHandler {
	return &PickupHandler{pickupService: pickupService}
}

// CreatePickup godoc
// POST /api/v1/pickups
// Content-Type: multipart/form-data
func (h *PickupHandler) CreatePickup(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req domain.CreatePickupRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	// Ambil file foto (opsional)
	file, header, err := c.Request.FormFile("photo")
	if err != nil && err != http.ErrMissingFile {
		utils.Error(c, http.StatusBadRequest, "Gagal membaca foto: "+err.Error())
		return
	}
	// Jika err == http.ErrMissingFile maka file = nil, pickup tetap dibuat tanpa foto

	pickup, err := h.pickupService.CreatePickup(c.Request.Context(), userID, &req, file, header)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "Pickup berhasil dibuat, mencari collector terdekat...", pickup)
}

// GetMyPickups godoc
// GET /api/v1/pickups/my?page=1&limit=20
func (h *PickupHandler) GetMyPickups(c *gin.Context) {
	userID := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	pickups, total, err := h.pickupService.GetMyPickups(c.Request.Context(), userID, page, limit)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Berhasil", utils.BuildListResponse(pickups, total, page, limit))
}

// GetPickupDetail godoc
// GET /api/v1/pickups/:id
func (h *PickupHandler) GetPickupDetail(c *gin.Context) {
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)
	pickupID := c.Param("id")

	pickup, err := h.pickupService.GetPickupDetail(c.Request.Context(), pickupID, userID, role)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Berhasil", pickup)
}