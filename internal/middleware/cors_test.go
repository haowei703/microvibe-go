package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"microvibe-go/internal/config"
	"microvibe-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

func corsCfg(origins ...string) *config.Config {
	if len(origins) == 0 {
		origins = []string{"http://localhost:5173", "http://localhost:3000"}
	}
	return &config.Config{
		CORS: config.CORSConfig{AllowedOrigins: origins},
	}
}

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	r := setupGin()
	cfg := corsCfg("http://example.com", "http://allowed.test")
	r.Use(middleware.CORSMiddleware(cfg))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://allowed.test")
	r.ServeHTTP(w, req)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://allowed.test" {
		t.Errorf("expected origin 'http://allowed.test', got '%s'", origin)
	}

	creds := w.Header().Get("Access-Control-Allow-Credentials")
	if creds != "true" {
		t.Errorf("expected credentials 'true', got '%s'", creds)
	}
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	r := setupGin()
	cfg := corsCfg("http://trusted.local")
	r.Use(middleware.CORSMiddleware(cfg))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	r.ServeHTTP(w, req)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "" {
		t.Errorf("expected empty origin for disallowed origin, got '%s'", origin)
	}
}

func TestCORSMiddleware_NoOrigin(t *testing.T) {
	r := setupGin()
	cfg := corsCfg()
	r.Use(middleware.CORSMiddleware(cfg))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "" {
		t.Errorf("expected no origin header when none sent, got '%s'", origin)
	}
}

func TestCORSMiddleware_OPTIONS(t *testing.T) {
	r := setupGin()
	cfg := corsCfg("http://localhost:5173")
	r.Use(middleware.CORSMiddleware(cfg))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	r.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("expected 204 for OPTIONS, got %d", w.Code)
	}
}

func TestCORSMiddleware_Methods(t *testing.T) {
	r := setupGin()
	cfg := corsCfg()
	r.Use(middleware.CORSMiddleware(cfg))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Fatal("expected Allow-Methods header")
	}
}

func TestCORSMiddleware_MaxAge(t *testing.T) {
	r := setupGin()
	cfg := corsCfg()
	r.Use(middleware.CORSMiddleware(cfg))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	maxAge := w.Header().Get("Access-Control-Max-Age")
	if maxAge != "86400" {
		t.Errorf("expected Max-Age 86400, got '%s'", maxAge)
	}
}

func TestCORSMiddleware_CustomHeaders(t *testing.T) {
	r := setupGin()
	cfg := corsCfg("http://app.local")
	r.Use(middleware.CORSMiddleware(cfg))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://app.local")
	req.Header.Set("Access-Control-Request-Headers", "X-Custom-Header, X-Another")
	r.ServeHTTP(w, req)

	headers := w.Header().Get("Access-Control-Allow-Headers")
	if headers != "X-Custom-Header, X-Another" {
		t.Errorf("expected custom headers reflected, got '%s'", headers)
	}
}
