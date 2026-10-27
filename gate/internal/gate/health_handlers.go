package gate

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const healthCheckTimeout = 2 * time.Second

// HealthCheck asks a service gate depends on whether it's serving, through the standard gRPC
// health service.
type HealthCheck struct {
	Service string
	Client  healthpb.HealthClient
}

// The response doesn't say which service failed, since that's internal; gate's log does.
type healthResponse struct {
	Status string `json:"status"`
}

func GetHealth(ctx *gin.Context, checks []HealthCheck) {
	serving := make([]bool, len(checks))

	var wg sync.WaitGroup
	for i, check := range checks {
		wg.Go(func() {
			checkCtx, cancel := context.WithTimeout(ctx.Request.Context(), healthCheckTimeout)
			defer cancel()

			res, err := check.Client.Check(checkCtx, &healthpb.HealthCheckRequest{})
			if err != nil {
				slog.Error("GetHealth: service unreachable", "dependency", check.Service, "error", err)
				return
			}
			if res.GetStatus() != healthpb.HealthCheckResponse_SERVING {
				slog.Error("GetHealth: service not serving", "dependency", check.Service, "status", res.GetStatus().String())
				return
			}
			serving[i] = true
		})
	}
	wg.Wait()

	if slices.Contains(serving, false) {
		ctx.JSON(http.StatusServiceUnavailable, healthResponse{Status: "unavailable"})
		return
	}

	ctx.JSON(http.StatusOK, healthResponse{Status: "ok"})
}

func RegisterHealthRoutes(r *gin.Engine, checks []HealthCheck) {
	r.GET("/health", func(ctx *gin.Context) {
		GetHealth(ctx, checks)
	})
}
