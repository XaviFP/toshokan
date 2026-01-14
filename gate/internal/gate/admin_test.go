package gate

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAdminConfig_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		cfg      AdminConfig
		expected bool
	}{
		{
			name:     "fully_configured",
			cfg:      AdminConfig{HeaderName: "X-Admin-Token", HeaderSecret: "secret123"},
			expected: true,
		},
		{
			name:     "missing_header_name",
			cfg:      AdminConfig{HeaderName: "", HeaderSecret: "secret123"},
			expected: false,
		},
		{
			name:     "missing_header_secret",
			cfg:      AdminConfig{HeaderName: "X-Admin-Token", HeaderSecret: ""},
			expected: false,
		},
		{
			name:     "both_empty",
			cfg:      AdminConfig{HeaderName: "", HeaderSecret: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cfg.IsConfigured())
		})
	}
}

func TestRequireAdmin_NotRequired(t *testing.T) {
	cfg := AdminConfig{
		HeaderName:   "X-Admin-Token",
		HeaderSecret: "secret123",
	}

	router := gin.New()
	router.GET("/test", RequireAdmin(cfg, false), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAdmin_RequiredButNotConfigured(t *testing.T) {
	cfg := AdminConfig{
		HeaderName:   "",
		HeaderSecret: "",
	}

	router := gin.New()
	router.GET("/test", RequireAdmin(cfg, true), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "admin auth not configured")
}

func TestRequireAdmin_MissingHeader(t *testing.T) {
	cfg := AdminConfig{
		HeaderName:   "X-Admin-Token",
		HeaderSecret: "secret123",
	}

	router := gin.New()
	router.GET("/test", RequireAdmin(cfg, true), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "admin access required")
}

func TestRequireAdmin_InvalidHeader(t *testing.T) {
	cfg := AdminConfig{
		HeaderName:   "X-Admin-Token",
		HeaderSecret: "secret123",
	}

	router := gin.New()
	router.GET("/test", RequireAdmin(cfg, true), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Admin-Token", "wrong-secret")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "invalid admin credentials")
}

func TestRequireAdmin_ValidHeader(t *testing.T) {
	cfg := AdminConfig{
		HeaderName:   "X-Admin-Token",
		HeaderSecret: "secret123",
	}

	router := gin.New()
	router.GET("/test", RequireAdmin(cfg, true), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Admin-Token", "secret123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestRequireAdmin_CaseSensitiveSecret(t *testing.T) {
	cfg := AdminConfig{
		HeaderName:   "X-Admin-Token",
		HeaderSecret: "Secret123",
	}

	router := gin.New()
	router.GET("/test", RequireAdmin(cfg, true), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test with wrong case
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Admin-Token", "secret123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	// Test with correct case
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Admin-Token", "Secret123")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
