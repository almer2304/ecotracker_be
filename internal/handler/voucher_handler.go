package handler

import (
	"net/http"
	"strconv"

	"ecotracker/internal/middleware"
	"ecotracker/internal/service"
	"ecotracker/internal/utils"

	"github.com/gin-gonic/gin"
)

type VoucherHandler struct {
	voucherService *service.VoucherService
}

func NewVoucherHandler(voucherService *service.VoucherService) *VoucherHandler {
	return &VoucherHandler{voucherService: voucherService}
}

// ListVouchers godoc
// GET /api/v1/vouchers
// Role: authenticated
func (h *VoucherHandler) ListVouchers(c *gin.Context) {
	vouchers, err := h.voucherService.ListAvailable(c.Request.Context())
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}
	utils.RespondSuccess(c, http.StatusOK, "Available vouchers retrieved", vouchers)
}

// ClaimVoucher godoc
// POST /api/v1/vouchers/:id/claim
// Role: user
func (h *VoucherHandler) ClaimVoucher(c *gin.Context) {
	idStr := c.Param("id")
	voucherID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	userID := middleware.GetUserID(c)
	uv, err := h.voucherService.ClaimVoucher(c.Request.Context(), userID, voucherID)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}

	utils.RespondSuccess(c, http.StatusCreated, "Voucher claimed successfully", uv)
}

// GetMyVouchers godoc
// GET /api/v1/vouchers/my
// Role: user
func (h *VoucherHandler) GetMyVouchers(c *gin.Context) {
	userID := middleware.GetUserID(c)
	uvs, err := h.voucherService.GetMyVouchers(c.Request.Context(), userID)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}
	utils.RespondSuccess(c, http.StatusOK, "My vouchers retrieved", uvs)
}
