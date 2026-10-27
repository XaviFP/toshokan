// Package health answers the standard gRPC health check by pinging the databases and caches
// a service depends on, so a service that can't reach them reports NOT_SERVING.
package health

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/mediocregopher/radix/v4"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// Check pings a database or cache the service depends on, such as Postgres or Redis.
type Check struct {
	Name string
	Ping func(ctx context.Context) error
}

func Postgres(db *sql.DB) Check {
	return Check{Name: "postgres", Ping: db.PingContext}
}

func Redis(client radix.Client) Check {
	return Check{Name: "redis", Ping: func(ctx context.Context) error {
		return client.Do(ctx, radix.Cmd(nil, "PING"))
	}}
}

// Server only answers for the whole service, the empty service name. Watch and List
// answer Unimplemented.
type Server struct {
	healthpb.UnimplementedHealthServer

	checks []Check
}

func NewServer(checks ...Check) *Server {
	return &Server{checks: checks}
}

func (s *Server) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	if req.GetService() != "" {
		return nil, status.Errorf(codes.NotFound, "unknown service %q", req.GetService())
	}

	for _, check := range s.checks {
		if err := check.Ping(ctx); err != nil {
			slog.Error("Health check failed", "dependency", check.Name, "error", err)
			return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_NOT_SERVING}, nil
		}
	}

	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}
