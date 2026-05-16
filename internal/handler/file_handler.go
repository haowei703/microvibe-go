package handler

import (
	"fmt"
	"microvibe-go/internal/config"
	"microvibe-go/internal/middleware"
	"microvibe-go/pkg/response"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// FileHandler 文件处理器
type FileHandler struct {
	cfg *config.Config
}

// NewFileHandler 创建文件处理器实例
func NewFileHandler(cfg *config.Config) *FileHandler {
	return &FileHandler{
		cfg: cfg,
	}
}

// UploadImage 上传图片资源
func (h *FileHandler) UploadImage(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		response.InvalidParam(c, "无法获取上传的图片: "+err.Error())
		return
	}

	// 校验文件扩展名
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	if !allowedExts[ext] {
		response.InvalidParam(c, "不支持的文件格式: "+ext)
		return
	}

	// 校验文件大小 (5MB)
	if file.Size > 5*1024*1024 {
		response.InvalidParam(c, "图片文件太大，不能超过5MB")
		return
	}

	// 创建保存目录
	timestamp := time.Now().Unix()
	baseDir := "./uploads/images"
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		response.ServerError(c, "创建目录失败: "+err.Error())
		return
	}

	// 生成唯一文件名
	filename := fmt.Sprintf("%d_%d_%s", userID, timestamp, file.Filename)
	// 去除文件名中的特殊字符，避免路径问题
	filename = strings.ReplaceAll(filename, " ", "_")

	savePath := filepath.Join(baseDir, filename)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		response.ServerError(c, "保存图片失败: "+err.Error())
		return
	}

	// 返回访问URL
	// 基础URL可以从配置中获取，或者根据请求动态构建
	url := fmt.Sprintf("/uploads/images/%s", filename)

	response.Success(c, gin.H{
		"url": url,
	})
}

// UploadVideo 上传视频资源
func (h *FileHandler) UploadVideo(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	file, err := c.FormFile("video")
	if err != nil {
		response.InvalidParam(c, "无法获取上传的视频: "+err.Error())
		return
	}

	// 校验文件扩展名
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".mp4":  true,
		".mov":  true,
		".avi":  true,
		".wmv":  true,
		".flv":  true,
		".mkv":  true,
		".webm": true,
	}

	if !allowedExts[ext] {
		response.InvalidParam(c, "不支持的视频格式: "+ext)
		return
	}

	// 校验文件大小 (50MB)
	if file.Size > 50*1024*1024 {
		response.InvalidParam(c, "视频文件太大，不能超过50MB")
		return
	}

	// 创建保存目录
	timestamp := time.Now().Unix()
	baseDir := "./uploads/videos"
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		response.ServerError(c, "创建目录失败: "+err.Error())
		return
	}

	// 生成唯一文件名
	filename := fmt.Sprintf("%d_%d_%s", userID, timestamp, file.Filename)
	filename = strings.ReplaceAll(filename, " ", "_")
	savePath := filepath.Join(baseDir, filename)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		response.ServerError(c, "保存视频失败: "+err.Error())
		return
	}

	url := fmt.Sprintf("/uploads/videos/%s", filename)
	response.Success(c, gin.H{
		"url": url,
	})
}

// UploadAudio 上传语音/音频资源
func (h *FileHandler) UploadAudio(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	file, err := c.FormFile("audio")
	if err != nil {
		response.InvalidParam(c, "无法获取上传的音频: "+err.Error())
		return
	}

	// 校验文件扩展名
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".mp3": true,
		".wav": true,
		".aac": true,
		".m4a": true,
		".ogg": true,
		".amr": true,
	}

	if !allowedExts[ext] {
		response.InvalidParam(c, "不支持的音频格式: "+ext)
		return
	}

	// 校验文件大小 (10MB)
	if file.Size > 10*1024*1024 {
		response.InvalidParam(c, "音频文件太大，不能超过10MB")
		return
	}

	// 创建保存目录
	timestamp := time.Now().Unix()
	baseDir := "./uploads/audio"
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		response.ServerError(c, "创建目录失败: "+err.Error())
		return
	}

	// 生成唯一文件名
	filename := fmt.Sprintf("%d_%d_%s", userID, timestamp, file.Filename)
	filename = strings.ReplaceAll(filename, " ", "_")
	savePath := filepath.Join(baseDir, filename)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		response.ServerError(c, "保存音频失败: "+err.Error())
		return
	}

	url := fmt.Sprintf("/uploads/audio/%s", filename)
	response.Success(c, gin.H{
		"url": url,
	})
}
