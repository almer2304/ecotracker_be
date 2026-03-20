package handler

import (
	"net/http"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/middleware"
	"github.com/ecotracker/backend/internal/service"
	"github.com/ecotracker/backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// AuthHandler mengelola endpoint autentikasi
type AuthHandler struct {
	authService *service.AuthService
	adminSecret string
}

func NewAuthHandler(authService *service.AuthService, adminSecret string) *AuthHandler {
	return &AuthHandler{authService: authService, adminSecret: adminSecret}
}

// Register godoc
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "Registrasi berhasil", resp)
}

// Login godoc
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Login berhasil", resp)
}

// RefreshToken godoc
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req domain.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	resp, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Token diperbarui", resp)
}

// GetProfile godoc
// GET /api/v1/auth/profile
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	profile, err := h.authService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "Profil berhasil diambil", profile)
}

// RegisterAdmin godoc
// POST /api/v1/auth/register-admin
// Header: X-Admin-Secret: <secret>
func (h *AuthHandler) RegisterAdmin(c *gin.Context) {
	// Validasi secret key
	secret := c.GetHeader("X-Admin-Secret")
	if secret != h.adminSecret {
		utils.Error(c, http.StatusForbidden, "Secret key tidak valid")
		return
	}

	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	resp, err := h.authService.RegisterWithRole(c.Request.Context(), &req, domain.RoleAdmin)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "Akun admin berhasil dibuat", resp)
}

// RegisterCollector godoc
// POST /api/v1/auth/register-collector
// Header: X-Admin-Secret: <secret>
func (h *AuthHandler) RegisterCollector(c *gin.Context) {
	// Validasi secret key
	secret := c.GetHeader("X-Admin-Secret")
	if secret != h.adminSecret {
		utils.Error(c, http.StatusForbidden, "Secret key tidak valid")
		return
	}

	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
		return
	}

	resp, err := h.authService.RegisterWithRole(c.Request.Context(), &req, domain.RoleCollector)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "Akun collector berhasil dibuat", resp)
}