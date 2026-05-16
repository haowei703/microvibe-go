package handler

import (
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/model"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	reportService service.ReportService
}

func NewReportHandler(reportService service.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

// CreateReport POST /api/v1/reports
func (h *ReportHandler) CreateReport(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		TargetID    uint   `json:"target_id" binding:"required"`
		TargetType  int8   `json:"target_type" binding:"required"`
		Reason      string `json:"reason" binding:"required,max=255"`
		Description string `json:"description" binding:"max=1000"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	report := &model.Report{
		ReporterID: userID,

		TargetID:    req.TargetID,
		TargetType:  req.TargetType,
		Reason:      req.Reason,
		Description: req.Description,
	}

	if err := h.reportService.CreateReport(c.Request.Context(), report); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "举报提交成功，我们会尽快处理", nil)
}

// GetMyReports GET /api/v1/reports
func (h *ReportHandler) GetMyReports(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	list, total, err := h.reportService.GetReportsByReporter(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取举报记录失败")
		return
	}

	response.Success(c, gin.H{
		"items": list,
		"total": total,
	})
}
