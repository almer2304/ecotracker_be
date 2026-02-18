package handler

import (
	"net/http"

	"ecotracker/internal/domain"
	"ecotracker/internal/middleware"
	"ecotracker/internal/service"
	"ecotracker/internal/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register godoc
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}

	utils.RespondSuccess(c, http.StatusCreated, "Registration successful", resp)
}

// Login godoc
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}

	utils.RespondSuccess(c, http.StatusOK, "Login successful", resp)
}

// GetProfile godoc
// GET /api/v1/auth/profile
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	profile, err := h.authService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}

	utils.RespondSuccess(c, http.StatusOK, "Profile retrieved", profile)
}
