package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"microvibe-go/internal/config"
	"microvibe-go/internal/middleware"
	"microvibe-go/pkg/utils"

	"github.com/gin-gonic/gin"
)

func setupGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func newTestCfg() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{Secret: "test-secret-key", Expire: 24},
	}
}

func generateTestToken(userID uint, username string, role int8) string {
	token, _ := utils.GenerateToken(userID, username, role, "test-secret-key", 24)
	return token
}

func TestAuthMiddleware_NoHeader(t *testing.T) {
	r := setupGin()
	r.GET("/test", middleware.AuthMiddleware(newTestCfg(), nil), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_MalformedBearer(t *testing.T) {
	r := setupGin()
	r.GET("/test", middleware.AuthMiddleware(newTestCfg(), nil), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "BearerTokenWithoutSpace")
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_WrongScheme(t *testing.T) {
	r := setupGin()
	r.GET("/test", middleware.AuthMiddleware(newTestCfg(), nil), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 for wrong scheme, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	cfg := newTestCfg()
	token := generateTestToken(1, "testuser", 0)

	r := setupGin()
	r.GET("/test", middleware.AuthMiddleware(cfg, nil), func(c *gin.Context) {
		uid, _ := middleware.GetUserID(c)
		username, _ := c.Get("username")
		c.JSON(200, gin.H{"uid": uid, "username": username})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	cfg := newTestCfg()
	token, _ := utils.GenerateToken(1, "user", 0, "test-secret-key", -1)

	r := setupGin()
	r.GET("/test", middleware.AuthMiddleware(cfg, nil), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	cfg := newTestCfg()
	token, _ := utils.GenerateToken(1, "user", 0, "different-secret", 24)

	r := setupGin()
	r.GET("/test", middleware.AuthMiddleware(cfg, nil), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 for wrong-secret token, got %d", w.Code)
	}
}

func TestAuthMiddleware_EmptyTokenAfterBearer(t *testing.T) {
	cfg := newTestCfg()

	r := setupGin()
	r.GET("/test", middleware.AuthMiddleware(cfg, nil), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ")
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 for empty token, got %d", w.Code)
	}
}

func TestAdminMiddleware_NoRole(t *testing.T) {
	cfg := newTestCfg()

	r := setupGin()
	r.GET("/admin", middleware.AuthMiddleware(cfg, nil), middleware.AdminMiddleware(), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+generateTestToken(1, "normal", 0))
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("expected 403 for non-admin, got %d", w.Code)
	}
}

func TestAdminMiddleware_Admin(t *testing.T) {
	cfg := newTestCfg()

	r := setupGin()
	r.GET("/admin", middleware.AuthMiddleware(cfg, nil), middleware.AdminMiddleware(), func(c *gin.Context) {
		c.JSON(200, gin.H{"admin": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+generateTestToken(1, "adminuser", 1))
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 for admin, got %d", w.Code)
	}
}

func TestOptionalAuthMiddleware_NoHeader(t *testing.T) {
	cfg := newTestCfg()

	r := setupGin()
	r.GET("/public", middleware.OptionalAuthMiddleware(cfg, nil), func(c *gin.Context) {
		_, exists := middleware.GetUserID(c)
		c.JSON(200, gin.H{"logged_in": exists})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/public", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestOptionalAuthMiddleware_ValidToken(t *testing.T) {
	cfg := newTestCfg()

	r := setupGin()
	r.GET("/public", middleware.OptionalAuthMiddleware(cfg, nil), func(c *gin.Context) {
		uid, exists := middleware.GetUserID(c)
		c.JSON(200, gin.H{"logged_in": exists, "uid": uid})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/public", nil)
	req.Header.Set("Authorization", "Bearer "+generateTestToken(42, "optuser", 0))
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestOptionalAuthMiddleware_InvalidToken(t *testing.T) {
	cfg := newTestCfg()

	r := setupGin()
	r.GET("/public", middleware.OptionalAuthMiddleware(cfg, nil), func(c *gin.Context) {
		_, exists := middleware.GetUserID(c)
		c.JSON(200, gin.H{"logged_in": exists})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/public", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 for invalid token, got %d", w.Code)
	}
}

func TestGetUserID_NotExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	_, exists := middleware.GetUserID(c)
	if exists {
		t.Fatal("expected GetUserID to return false when uid not set")
	}
}

func TestAuthMiddleware_GetsClaims(t *testing.T) {
	cfg := newTestCfg()
	token := generateTestToken(7, "claimuser", 1)

	r := setupGin()
	r.GET("/test", middleware.AuthMiddleware(cfg, nil), func(c *gin.Context) {
		claims, exists := c.Get("claims")
		if !exists {
			c.JSON(500, gin.H{"error": "no claims"})
			return
		}
		c.JSON(200, gin.H{"has_claims": true, "claims_type": true, "claims": claims})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}
