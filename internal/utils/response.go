package utils

import (
	"math"
	"net/http"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

// APIResponse format response standar
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Success mengirimkan response sukses
func Success(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error mengirimkan response error
func Error(c *gin.Context, code int, message string) {
	c.JSON(code, APIResponse{
		Success: false,
		Error:   message,
	})
}

// HandleError menangani error domain dan menghasilkan response yang sesuai
func HandleError(c *gin.Context, err error) {
	// Cek apakah AppError
	if appErr, ok := err.(*domain.AppError); ok {
		Error(c, appErr.Code, appErr.Message)
		return
	}

	// Map domain errors ke HTTP status
	switch err {
	case domain.ErrEmailAlreadyExists:
		Error(c, http.StatusConflict, err.Error())
	case domain.ErrInvalidCredentials:
		Error(c, http.StatusUnauthorized, err.Error())
	case domain.ErrInvalidToken, domain.ErrTokenExpired:
		Error(c, http.StatusUnauthorized, err.Error())
	case domain.ErrUnauthorized:
		Error(c, http.StatusUnauthorized, err.Error())
	case domain.ErrForbidden:
		Error(c, http.StatusForbidden, err.Error())
	case domain.ErrNotFound,
		domain.ErrPickupNotFound,
		domain.ErrCollectorNotFound,
		domain.ErrBadgeNotFound,
		domain.ErrReportNotFound,
		domain.ErrFeedbackNotFound,
		domain.ErrCategoryNotFound:
		Error(c, http.StatusNotFound, err.Error())
	case domain.ErrNoCollectorAvailable:
		Error(c, http.StatusServiceUnavailable, err.Error())
	case domain.ErrPickupAlreadyAssigned,
		domain.ErrCollectorBusy,
		domain.ErrInvalidPickupStatus,
		domain.ErrBadgeAlreadyAwarded:
		Error(c, http.StatusConflict, err.Error())
	case domain.ErrCollectorOffline:
		Error(c, http.StatusBadRequest, err.Error())
	case domain.ErrInvalidInput,
		domain.ErrFileTooLarge,
		domain.ErrInvalidFileType:
		Error(c, http.StatusBadRequest, err.Error())
	default:
		Error(c, http.StatusInternalServerError, "Terjadi kesalahan internal, coba lagi nanti")
	}
}

// BuildListResponse membuat response list dengan pagination
func BuildListResponse(data interface{}, total, page, limit int) domain.ListResponse {
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	return domain.ListResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}
}
