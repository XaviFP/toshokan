package health

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func TestCheck(t *testing.T) {
	t.Run("success_every_dependency_answers", func(t *testing.T) {
		server := NewServer(
			Check{Name: "postgres", Ping: func(ctx context.Context) error { return nil }},
			Check{Name: "redis", Ping: func(ctx context.Context) error { return nil }},
		)

		actual, err := server.Check(context.Background(), &healthpb.HealthCheckRequest{})
		require.NoError(t, err)

		assert.Equal(t, healthpb.HealthCheckResponse_SERVING, actual.GetStatus())
	})

	t.Run("success_no_dependencies", func(t *testing.T) {
		server := NewServer()

		actual, err := server.Check(context.Background(), &healthpb.HealthCheckRequest{})
		require.NoError(t, err)

		assert.Equal(t, healthpb.HealthCheckResponse_SERVING, actual.GetStatus())
	})

	t.Run("success_ping_gets_the_request_context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		server := NewServer(
			Check{Name: "postgres", Ping: func(ctx context.Context) error { return ctx.Err() }},
		)

		actual, err := server.Check(ctx, &healthpb.HealthCheckRequest{})
		require.NoError(t, err)

		assert.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, actual.GetStatus())
	})

	t.Run("failure_first_dependency_unreachable", func(t *testing.T) {
		server := NewServer(
			Check{Name: "postgres", Ping: func(ctx context.Context) error { return errors.New("connection refused") }},
			Check{Name: "redis", Ping: func(ctx context.Context) error { return nil }},
		)

		actual, err := server.Check(context.Background(), &healthpb.HealthCheckRequest{})
		require.NoError(t, err)

		assert.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, actual.GetStatus())
	})

	t.Run("failure_last_dependency_unreachable", func(t *testing.T) {
		server := NewServer(
			Check{Name: "postgres", Ping: func(ctx context.Context) error { return nil }},
			Check{Name: "redis", Ping: func(ctx context.Context) error { return errors.New("connection refused") }},
		)

		actual, err := server.Check(context.Background(), &healthpb.HealthCheckRequest{})
		require.NoError(t, err)

		assert.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, actual.GetStatus())
	})

	t.Run("failure_unknown_service", func(t *testing.T) {
		server := NewServer()

		_, err := server.Check(context.Background(), &healthpb.HealthCheckRequest{Service: "deck.v1.DecksAPI"})

		assert.Equal(t, codes.NotFound, status.Code(err))
	})
}
