package handler

import (
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// LiveProductHandler 直播商品Handler
type LiveProductHandler struct {
	productService service.LiveProductService
}

// NewLiveProductHandler 创建商品Handler
func NewLiveProductHandler(productService service.LiveProductService) *LiveProductHandler {
	return &LiveProductHandler{
		productService: productService,
	}
}

// AddProduct 添加商品
// @Summary 添加商品到直播间
// @Tags 直播商品
// @Accept json
// @Param request body service.AddProductRequest true "添加商品请求"
// @Success 200 {object} response.Response
// @Router /api/v1/live/products [post]
func (h *LiveProductHandler) AddProduct(c *gin.Context) {
	// 获取当前登录用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var req service.AddProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	product, err := h.productService.AddProduct(c.Request.Context(), userID.(uint), &req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "添加商品成功", product)
}

// UpdateProduct 更新商品
// @Summary 更新商品
// @Tags 直播商品
// @Accept json
// @Param id path int true "商品ID"
// @Param request body service.UpdateProductRequest true "更新商品请求"
// @Success 200 {object} response.Response
// @Router /api/v1/live/products/:id [put]
func (h *LiveProductHandler) UpdateProduct(c *gin.Context) {
	// 获取当前登录用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的商品ID")
		return
	}

	var req service.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	if err := h.productService.UpdateProduct(c.Request.Context(), userID.(uint), uint(id), &req); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新商品成功", nil)
}

// DeleteProduct 删除商品
// @Summary 删除商品
// @Tags 直播商品
// @Param id path int true "商品ID"
// @Success 200 {object} response.Response
// @Router /api/v1/live/products/:id [delete]
func (h *LiveProductHandler) DeleteProduct(c *gin.Context) {
	// 获取当前登录用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的商品ID")
		return
	}

	if err := h.productService.DeleteProduct(c.Request.Context(), userID.(uint), uint(id)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除商品成功", nil)
}

// GetProduct 获取商品详情
// @Summary 获取商品详情
// @Tags 直播商品
// @Param id path int true "商品ID"
// @Success 200 {object} response.Response
// @Router /api/v1/live/products/:id [get]
func (h *LiveProductHandler) GetProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的商品ID")
		return
	}

	product, err := h.productService.GetProduct(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, product)
}

// ListProducts 获取直播间商品列表
// @Summary 获取直播间商品列表
// @Tags 直播商品
// @Param live_id query int true "直播间ID"
// @Param status query int false "状态"
// @Success 200 {object} response.Response
// @Router /api/v1/live/products [get]
func (h *LiveProductHandler) ListProducts(c *gin.Context) {
	liveIDStr := c.Query("live_id")
	status := c.DefaultQuery("status", "-1")

	liveID, err := strconv.ParseUint(liveIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	statusInt, _ := strconv.Atoi(status)

	products, err := h.productService.ListProducts(c.Request.Context(), uint(liveID), int8(statusInt))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, products)
}

// GetHotProducts 获取热卖商品
// @Summary 获取热卖商品
// @Tags 直播商品
// @Param live_id query int true "直播间ID"
// @Param limit query int false "数量限制"
// @Success 200 {object} response.Response
// @Router /api/v1/live/products/hot [get]
func (h *LiveProductHandler) GetHotProducts(c *gin.Context) {
	liveIDStr := c.Query("live_id")
	limit := c.DefaultQuery("limit", "10")

	liveID, err := strconv.ParseUint(liveIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	limitInt, _ := strconv.Atoi(limit)

	products, err := h.productService.GetHotProducts(c.Request.Context(), uint(liveID), limitInt)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, products)
}

// ExplainProduct 讲解商品
// @Summary 讲解商品
// @Tags 直播商品
// @Param id path int true "商品ID"
// @Success 200 {object} response.Response
// @Router /api/v1/live/products/:id/explain [post]
func (h *LiveProductHandler) ExplainProduct(c *gin.Context) {
	// 获取当前登录用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的商品ID")
		return
	}

	if err := h.productService.ExplainProduct(c.Request.Context(), userID.(uint), uint(id)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "讲解商品成功", nil)
}

// UpdateStock 更新库存
// @Summary 更新库存
// @Tags 直播商品
// @Param id path int true "商品ID"
// @Param quantity query int true "变更数量（正数增加，负数减少）"
// @Success 200 {object} response.Response
// @Router /api/v1/live/products/:id/stock [put]
func (h *LiveProductHandler) UpdateStock(c *gin.Context) {
	// 获取当前登录用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的商品ID")
		return
	}

	quantityStr := c.Query("quantity")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil {
		response.InvalidParam(c, "无效的数量")
		return
	}

	if err := h.productService.UpdateStock(c.Request.Context(), userID.(uint), uint(id), quantity); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新库存成功", nil)
}
