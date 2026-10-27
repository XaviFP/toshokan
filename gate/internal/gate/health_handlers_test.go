package gate

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func TestGetHealth(t *testing.T) {
	t.Run("success_all_serving", func(t *testing.T) {
		userClient := &mockHealthClient{}
		dealerClient := &mockHealthClient{}

		userClient.On("Check", mock.Anything, &healthpb.HealthCheckRequest{}).
			Return(&healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil)
		dealerClient.On("Check", mock.Anything, &healthpb.HealthCheckRequest{}).
			Return(&healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil)

		router := gin.New()
		RegisterHealthRoutes(router, []HealthCheck{
			{Service: "user", Client: userClient},
			{Service: "dealer", Client: dealerClient},
		})

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var actual healthResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &actual))

		assert.Equal(t, healthResponse{Status: "ok"}, actual)
		userClient.AssertExpectations(t)
		dealerClient.AssertExpectations(t)
	})

	t.Run("success_checks_have_a_deadline", func(t *testing.T) {
		userClient := &mockHealthClient{}

		userClient.On("Check", mock.MatchedBy(func(ctx context.Context) bool {
			_, ok := ctx.Deadline()
			return ok
		}), &healthpb.HealthCheckRequest{}).
			Return(&healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil)

		router := gin.New()
		RegisterHealthRoutes(router, []HealthCheck{
			{Service: "user", Client: userClient},
		})

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		userClient.AssertExpectations(t)
	})

	t.Run("failure_service_unreachable", func(t *testing.T) {
		userClient := &mockHealthClient{}
		dealerClient := &mockHealthClient{}

		userClient.On("Check", mock.Anything, &healthpb.HealthCheckRequest{}).
			Return(&healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil)
		dealerClient.On("Check", mock.Anything, &healthpb.HealthCheckRequest{}).
			Return(nil, status.Error(codes.Unavailable, "connection refused"))

		router := gin.New()
		RegisterHealthRoutes(router, []HealthCheck{
			{Service: "user", Client: userClient},
			{Service: "dealer", Client: dealerClient},
		})

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusServiceUnavailable, w.Code)

		var actual healthResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &actual))

		assert.Equal(t, healthResponse{Status: "unavailable"}, actual)
	})

	t.Run("failure_service_not_serving", func(t *testing.T) {
		userClient := &mockHealthClient{}
		dealerClient := &mockHealthClient{}

		userClient.On("Check", mock.Anything, &healthpb.HealthCheckRequest{}).
			Return(&healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_NOT_SERVING}, nil)
		dealerClient.On("Check", mock.Anything, &healthpb.HealthCheckRequest{}).
			Return(&healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil)

		router := gin.New()
		RegisterHealthRoutes(router, []HealthCheck{
			{Service: "user", Client: userClient},
			{Service: "dealer", Client: dealerClient},
		})

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusServiceUnavailable, w.Code)

		var actual healthResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &actual))

		assert.Equal(t, healthResponse{Status: "unavailable"}, actual)
	})

	t.Run("failure_several_unavailable", func(t *testing.T) {
		userClient := &mockHealthClient{}
		deckClient := &mockHealthClient{}
		dealerClient := &mockHealthClient{}

		userClient.On("Check", mock.Anything, &healthpb.HealthCheckRequest{}).
			Return(nil, errors.New("deadline exceeded"))
		deckClient.On("Check", mock.Anything, &healthpb.HealthCheckRequest{}).
			Return(&healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil)
		dealerClient.On("Check", mock.Anything, &healthpb.HealthCheckRequest{}).
			Return(&healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_UNKNOWN}, nil)

		router := gin.New()
		RegisterHealthRoutes(router, []HealthCheck{
			{Service: "user", Client: userClient},
			{Service: "deck", Client: deckClient},
			{Service: "dealer", Client: dealerClient},
		})

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusServiceUnavailable, w.Code)

		var actual healthResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &actual))

		assert.Equal(t, healthResponse{Status: "unavailable"}, actual)
	})
}

type mockHealthClient struct {
	mock.Mock
}

func (m *mockHealthClient) Check(ctx context.Context, req *healthpb.HealthCheckRequest, opts ...grpc.CallOption) (*healthpb.HealthCheckResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*healthpb.HealthCheckResponse), args.Error(1)
}

func (m *mockHealthClient) List(ctx context.Context, req *healthpb.HealthListRequest, opts ...grpc.CallOption) (*healthpb.HealthListResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*healthpb.HealthListResponse), args.Error(1)
}

func (m *mockHealthClient) Watch(ctx context.Context, req *healthpb.HealthCheckRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[healthpb.HealthCheckResponse], error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(grpc.ServerStreamingClient[healthpb.HealthCheckResponse]), args.Error(1)
}
