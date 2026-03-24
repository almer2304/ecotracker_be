package handler

import (
	"net/http"

	"github.com/ecotracker/backend/internal/repository"
	"github.com/ecotracker/backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// CategoryHandler mengelola endpoint waste categories
type CategoryHandler struct {
	categoryRepo *repository.CategoryRepository
}

func NewCategoryHandler(categoryRepo *repository.CategoryRepository) *CategoryHandler {
	return &CategoryHandler{categoryRepo: categoryRepo}
}

// GetAllCategories godoc
// GET /api/v1/categories
func (h *CategoryHandler) GetAllCategories(c *gin.Context) {
	categories, err := h.categoryRepo.GetAll(c.Request.Context())
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Berhasil", categories)
}