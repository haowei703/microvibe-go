package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// DeviceInfo 设备信息
type DeviceInfo struct {
	Platform    string // web, android, ios, windows, macos, linux
	AppVersion  string
	OSVersion   string
	DeviceModel string
	Browser     string
}

const DeviceContextKey = "device_info"

// DeviceMiddleware 用于识别设备信息
func DeviceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		device := DeviceInfo{
			Platform:    c.GetHeader("X-Platform"),
			AppVersion:  c.GetHeader("X-App-Version"),
			OSVersion:   c.GetHeader("X-OS-Version"),
			DeviceModel: c.GetHeader("X-Device-Model"),
		}

		ua := c.GetHeader("User-Agent")
		device.Browser = detectBrowser(ua)

		// 如果头部没有 Platform，尝试从 User-Agent 解析
		if device.Platform == "" {
			if device.Browser != "unknown" {
				device.Platform = "web"
			} else {
				device.Platform = detectPlatform(ua)
			}
		}

		// 默认设置为 web
		if device.Platform == "" {
			device.Platform = "web"
		}

		// 转换成小写
		device.Platform = strings.ToLower(device.Platform)

		c.Set(DeviceContextKey, device)
		c.Next()
	}
}

// GetDeviceInfo 从 Context 中获取设备信息
func GetDeviceInfo(c *gin.Context) DeviceInfo {
	if val, ok := c.Get(DeviceContextKey); ok {
		if info, ok := val.(DeviceInfo); ok {
			return info
		}
	}
	return DeviceInfo{Platform: "web"}
}

// IsNative 判断是否是原生平台 (Mobile/Desktop)
func (d DeviceInfo) IsNative() bool {
	p := strings.ToLower(d.Platform)
	return p == "android" || p == "ios" || p == "windows" || p == "macos" || p == "linux"
}

func detectPlatform(ua string) string {
	ua = strings.ToLower(ua)
	if strings.Contains(ua, "android") {
		return "android"
	}
	if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
		return "ios"
	}
	if strings.Contains(ua, "windows") {
		return "windows"
	}
	if strings.Contains(ua, "macintosh") {
		return "macos"
	}
	if strings.Contains(ua, "linux") {
		return "linux"
	}
	return ""
}

func detectBrowser(ua string) string {
	ua = strings.ToLower(ua)
	if strings.Contains(ua, "edg/") {
		return "edge"
	}
	if strings.Contains(ua, "chrome/") {
		return "chrome"
	}
	if strings.Contains(ua, "safari/") {
		return "safari"
	}
	if strings.Contains(ua, "firefox/") {
		return "firefox"
	}
	return "unknown"
}
