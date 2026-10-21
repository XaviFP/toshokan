package gate

import (
	"bytes"
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
	"google.golang.org/grpc/status"

	pbDealer "github.com/XaviFP/toshokan/dealer/api/proto/v1"
	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
)

func TestGetReview(t *testing.T) {
	t.Run("success_with_deck_id_keeps_dealt_order", func(t *testing.T) {
		dealerClient := &mockDealerClient{}
		decksClient := &mockDecksClient{}

		dealerClient.On("Deal", mock.Anything, &pbDealer.DealRequest{
			UserId:        "4acc3aa4-7684-4a64-b61c-28a599881922",
			DeckIds:       []string{"fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72"},
			NumberOfCards: 2,
		}).Return(&pbDealer.DealResponse{CardIds: []string{"card-b", "card-a"}}, nil)
		decksClient.On("GetCards", mock.Anything, &pbDeck.GetCardsRequest{CardIds: []string{"card-b", "card-a"}}).
			Return(&pbDeck.GetCardsResponse{Cards: map[string]*pbDeck.Card{
				"card-a": {Id: "card-a", Title: "A"},
				"card-b": {Id: "card-b", Title: "B"},
			}}, nil)

		router := gin.New()
		router.Use(func(ctx *gin.Context) { ctx.Set("userID", "4acc3aa4-7684-4a64-b61c-28a599881922") })
		router.GET("/review", func(ctx *gin.Context) {
			GetReview(ctx, dealerClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/review?max_cards=2&deck_id=fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var actual reviewResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &actual))

		require.Len(t, actual.Cards, 2)
		assert.Equal(t, "card-b", actual.Cards[0].ID)
		assert.Equal(t, "card-a", actual.Cards[1].ID)
	})

	t.Run("success_without_deck_id_deals_from_all_decks_with_default_max_cards", func(t *testing.T) {
		dealerClient := &mockDealerClient{}
		decksClient := &mockDecksClient{}

		dealerClient.On("Deal", mock.Anything, &pbDealer.DealRequest{
			UserId:        "4acc3aa4-7684-4a64-b61c-28a599881922",
			DeckIds:       nil,
			NumberOfCards: DefaultReviewCards,
		}).Return(&pbDealer.DealResponse{CardIds: []string{"card-a"}}, nil)
		decksClient.On("GetCards", mock.Anything, &pbDeck.GetCardsRequest{CardIds: []string{"card-a"}}).
			Return(&pbDeck.GetCardsResponse{Cards: map[string]*pbDeck.Card{
				"card-a": {Id: "card-a", Title: "A"},
			}}, nil)

		router := gin.New()
		router.Use(func(ctx *gin.Context) { ctx.Set("userID", "4acc3aa4-7684-4a64-b61c-28a599881922") })
		router.GET("/review", func(ctx *gin.Context) {
			GetReview(ctx, dealerClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/review", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var actual reviewResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &actual))

		require.Len(t, actual.Cards, 1)
		assert.Equal(t, "card-a", actual.Cards[0].ID)
	})

	t.Run("success_skips_cards_missing_from_deck_service", func(t *testing.T) {
		dealerClient := &mockDealerClient{}
		decksClient := &mockDecksClient{}

		dealerClient.On("Deal", mock.Anything, mock.Anything).
			Return(&pbDealer.DealResponse{CardIds: []string{"card-a", "card-gone"}}, nil)
		decksClient.On("GetCards", mock.Anything, mock.Anything).
			Return(&pbDeck.GetCardsResponse{Cards: map[string]*pbDeck.Card{
				"card-a": {Id: "card-a", Title: "A"},
			}}, nil)

		router := gin.New()
		router.GET("/review", func(ctx *gin.Context) {
			GetReview(ctx, dealerClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/review", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var actual reviewResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &actual))

		require.Len(t, actual.Cards, 1)
		assert.Equal(t, "card-a", actual.Cards[0].ID)
	})

	t.Run("success_nothing_answered_returns_empty_list", func(t *testing.T) {
		dealerClient := &mockDealerClient{}
		decksClient := &mockDecksClient{}

		dealerClient.On("Deal", mock.Anything, mock.Anything).
			Return(&pbDealer.DealResponse{}, nil)

		router := gin.New()
		router.GET("/review", func(ctx *gin.Context) {
			GetReview(ctx, dealerClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/review", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"cards":[]}`, w.Body.String())

		decksClient.AssertNotCalled(t, "GetCards", mock.Anything, mock.Anything)
	})

	t.Run("failure_max_cards_zero", func(t *testing.T) {
		dealerClient := &mockDealerClient{}
		decksClient := &mockDecksClient{}

		router := gin.New()
		router.GET("/review", func(ctx *gin.Context) {
			GetReview(ctx, dealerClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/review?max_cards=0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("failure_max_cards_above_limit", func(t *testing.T) {
		dealerClient := &mockDealerClient{}
		decksClient := &mockDecksClient{}

		router := gin.New()
		router.GET("/review", func(ctx *gin.Context) {
			GetReview(ctx, dealerClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/review?max_cards=101", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("failure_max_cards_negative", func(t *testing.T) {
		dealerClient := &mockDealerClient{}
		decksClient := &mockDecksClient{}

		router := gin.New()
		router.GET("/review", func(ctx *gin.Context) {
			GetReview(ctx, dealerClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/review?max_cards=-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("failure_max_cards_not_a_number", func(t *testing.T) {
		dealerClient := &mockDealerClient{}
		decksClient := &mockDecksClient{}

		router := gin.New()
		router.GET("/review", func(ctx *gin.Context) {
			GetReview(ctx, dealerClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/review?max_cards=abc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("failure_invalid_deck_id", func(t *testing.T) {
		dealerClient := &mockDealerClient{}
		decksClient := &mockDecksClient{}

		router := gin.New()
		router.GET("/review", func(ctx *gin.Context) {
			GetReview(ctx, dealerClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/review?deck_id=invalid-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid deck id format")
	})

	t.Run("failure_empty_deck_id", func(t *testing.T) {
		dealerClient := &mockDealerClient{}
		decksClient := &mockDecksClient{}

		router := gin.New()
		router.GET("/review", func(ctx *gin.Context) {
			GetReview(ctx, dealerClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/review?deck_id=", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid deck id format")
	})

	t.Run("failure_dealer_invalid_argument", func(t *testing.T) {
		dealerClient := &mockDealerClient{}
		decksClient := &mockDecksClient{}

		dealerClient.On("Deal", mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.InvalidArgument, "invalid user_id"))

		router := gin.New()
		router.GET("/review", func(ctx *gin.Context) {
			GetReview(ctx, dealerClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/review", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid user_id")
	})

	t.Run("failure_dealer_internal", func(t *testing.T) {
		dealerClient := &mockDealerClient{}
		decksClient := &mockDecksClient{}

		dealerClient.On("Deal", mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Internal, "deal failed"))

		router := gin.New()
		router.GET("/review", func(ctx *gin.Context) {
			GetReview(ctx, dealerClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/review", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("failure_get_cards", func(t *testing.T) {
		dealerClient := &mockDealerClient{}
		decksClient := &mockDecksClient{}

		dealerClient.On("Deal", mock.Anything, mock.Anything).
			Return(&pbDealer.DealResponse{CardIds: []string{"card-a"}}, nil)
		decksClient.On("GetCards", mock.Anything, mock.Anything).
			Return(nil, errors.New("deck down"))

		router := gin.New()
		router.GET("/review", func(ctx *gin.Context) {
			GetReview(ctx, dealerClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/review", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestReviewAnswers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dealerClient := &mockDealerClient{}

		dealerClient.On("ReviewAnswers", mock.Anything, &pbDealer.ReviewAnswersRequest{
			UserId:    "4acc3aa4-7684-4a64-b61c-28a599881922",
			AnswerIds: []string{"7e6926da-82b2-4ae8-99b4-1b803ebf1877", "72bdff92-5bc8-4e1d-9217-d0b23e22ff33"},
		}).Return(&pbDealer.ReviewAnswersResponse{}, nil)

		router := gin.New()
		router.Use(func(ctx *gin.Context) { ctx.Set("userID", "4acc3aa4-7684-4a64-b61c-28a599881922") })
		router.POST("/review/answers", func(ctx *gin.Context) {
			ReviewAnswers(ctx, dealerClient)
		})

		body, _ := json.Marshal(map[string]interface{}{
			"answer_ids": []string{"7e6926da-82b2-4ae8-99b4-1b803ebf1877", "72bdff92-5bc8-4e1d-9217-d0b23e22ff33"},
		})
		req := httptest.NewRequest(http.MethodPost, "/review/answers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		dealerClient.AssertExpectations(t)
	})

	t.Run("failure_empty_answer_ids", func(t *testing.T) {
		dealerClient := &mockDealerClient{}

		router := gin.New()
		router.POST("/review/answers", func(ctx *gin.Context) {
			ReviewAnswers(ctx, dealerClient)
		})

		body, _ := json.Marshal(map[string]interface{}{"answer_ids": []string{}})
		req := httptest.NewRequest(http.MethodPost, "/review/answers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "answer_ids must not be empty")
	})

	t.Run("failure_invalid_answer_id", func(t *testing.T) {
		dealerClient := &mockDealerClient{}

		router := gin.New()
		router.POST("/review/answers", func(ctx *gin.Context) {
			ReviewAnswers(ctx, dealerClient)
		})

		body, _ := json.Marshal(map[string]interface{}{"answer_ids": []string{"invalid-uuid"}})
		req := httptest.NewRequest(http.MethodPost, "/review/answers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid answer id format")
	})

	t.Run("failure_malformed_body", func(t *testing.T) {
		dealerClient := &mockDealerClient{}

		router := gin.New()
		router.POST("/review/answers", func(ctx *gin.Context) {
			ReviewAnswers(ctx, dealerClient)
		})

		req := httptest.NewRequest(http.MethodPost, "/review/answers", bytes.NewReader([]byte("{")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("failure_dealer_answer_not_in_pool", func(t *testing.T) {
		dealerClient := &mockDealerClient{}

		dealerClient.On("ReviewAnswers", mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.InvalidArgument, "answer_id not in review pool"))

		router := gin.New()
		router.POST("/review/answers", func(ctx *gin.Context) {
			ReviewAnswers(ctx, dealerClient)
		})

		body, _ := json.Marshal(map[string]interface{}{"answer_ids": []string{"7e6926da-82b2-4ae8-99b4-1b803ebf1877"}})
		req := httptest.NewRequest(http.MethodPost, "/review/answers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "answer_id not in review pool")
	})

	t.Run("failure_dealer_internal", func(t *testing.T) {
		dealerClient := &mockDealerClient{}

		dealerClient.On("ReviewAnswers", mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Internal, "store_answers failed"))

		router := gin.New()
		router.POST("/review/answers", func(ctx *gin.Context) {
			ReviewAnswers(ctx, dealerClient)
		})

		body, _ := json.Marshal(map[string]interface{}{"answer_ids": []string{"7e6926da-82b2-4ae8-99b4-1b803ebf1877"}})
		req := httptest.NewRequest(http.MethodPost, "/review/answers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

type mockDealerClient struct {
	mock.Mock
}

func (m *mockDealerClient) Deal(ctx context.Context, req *pbDealer.DealRequest, opts ...grpc.CallOption) (*pbDealer.DealResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDealer.DealResponse), args.Error(1)
}

func (m *mockDealerClient) StoreAnswers(ctx context.Context, req *pbDealer.StoreAnswersRequest, opts ...grpc.CallOption) (*pbDealer.StoreAnswersResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDealer.StoreAnswersResponse), args.Error(1)
}

func (m *mockDealerClient) ReviewAnswers(ctx context.Context, req *pbDealer.ReviewAnswersRequest, opts ...grpc.CallOption) (*pbDealer.ReviewAnswersResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDealer.ReviewAnswersResponse), args.Error(1)
}
