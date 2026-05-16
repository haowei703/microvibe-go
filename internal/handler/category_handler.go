package handler

import (
	"github.com/gin-gonic/gin"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
)

// CategoryHandler 分类处理器
type CategoryHandler struct {
	categoryService service.CategoryService
}

func NewCategoryHandler(categoryService service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

// GetCategories 获取分类列表
// @Summary 获取视频分类列表
// @Tags 视频
// @Success 200 {object} response.Response{data=[]model.Category}
// @Router /api/v1/categories [get]
func (h *CategoryHandler) GetCategories(c *gin.Context) {
	categories, err := h.categoryService.GetCategories(c.Request.Context())
	if err != nil {
		response.ServerError(c, "获取分类失败: "+err.Error())
		return
	}
	response.Success(c, categories)
}
