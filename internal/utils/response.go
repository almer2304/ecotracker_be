package utils

import (
	"errors"
	"net/http"

	"ecotracker/internal/domain"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func RespondSuccess(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func RespondError(c *gin.Context, statusCode int, err error) {
	c.JSON(statusCode, Response{
		Success: false,
		Error:   err.Error(),
	})
}

func RespondWithDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		RespondError(c, http.StatusNotFound, err)
	case errors.Is(err, domain.ErrUnauthorized):
		RespondError(c, http.StatusUnauthorized, err)
	case errors.Is(err, domain.ErrForbidden):
		RespondError(c, http.StatusForbidden, err)
	case errors.Is(err, domain.ErrConflict):
		RespondError(c, http.StatusConflict, err)
	case errors.Is(err, domain.ErrInvalidCredentials):
		RespondError(c, http.StatusUnauthorized, err)
	case errors.Is(err, domain.ErrInsufficientPoints):
		RespondError(c, http.StatusBadRequest, err)
	case errors.Is(err, domain.ErrVoucherOutOfStock):
		RespondError(c, http.StatusBadRequest, err)
	case errors.Is(err, domain.ErrVoucherInactive):
		RespondError(c, http.StatusBadRequest, err)
	case errors.Is(err, domain.ErrPickupNotPending):
		RespondError(c, http.StatusBadRequest, err)
	case errors.Is(err, domain.ErrPickupNotTaken):
		RespondError(c, http.StatusBadRequest, err)
	case errors.Is(err, domain.ErrPickupAlreadyTaken):
		RespondError(c, http.StatusConflict, err)
	default:
		RespondError(c, http.StatusInternalServerError, err)
	}
}
