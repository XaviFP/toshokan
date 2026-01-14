package gate

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminConfig holds admin access configuration
type AdminConfig struct {
	HeaderName   string
	HeaderSecret string

	// Per-endpoint admin requirements (true = admin-only)
	SignupAdminOnly       bool
	EnrollAdminOnly       bool
	CreateCourseAdminOnly bool
	CreateLessonAdminOnly bool
	CreateDeckAdminOnly   bool
	UpdateCourseAdminOnly bool
	UpdateLessonAdminOnly bool
}

// IsConfigured returns true if admin auth is properly configured
func (c AdminConfig) IsConfigured() bool {
	return c.HeaderName != "" && c.HeaderSecret != ""
}

// RequireAdmin returns a middleware that checks for admin header when required.
// If adminRequired is false, the middleware passes through.
// If adminRequired is true but admin is not configured, returns 500.
// If adminRequired is true and admin is configured, validates the header.
func RequireAdmin(cfg AdminConfig, adminRequired bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !adminRequired {
			ctx.Next()
			return
		}

		if !cfg.IsConfigured() {
			slog.Error("RequireAdmin: admin auth required but not configured")
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "admin auth not configured"})
			ctx.Abort()
			return
		}

		headerValue := ctx.GetHeader(cfg.HeaderName)
		if headerValue == "" {
			slog.Warn("RequireAdmin: missing admin header",
				"path", ctx.Request.URL.Path,
				"method", ctx.Request.Method,
				"remote_addr", ctx.ClientIP(),
			)
			ctx.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			ctx.Abort()
			return
		}

		if headerValue != cfg.HeaderSecret {
			slog.Warn("RequireAdmin: invalid admin header",
				"path", ctx.Request.URL.Path,
				"method", ctx.Request.Method,
				"remote_addr", ctx.ClientIP(),
			)
			ctx.JSON(http.StatusForbidden, gin.H{"error": "invalid admin credentials"})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
