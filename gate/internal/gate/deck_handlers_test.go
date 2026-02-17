package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
	pbUser "github.com/XaviFP/toshokan/user/api/proto/v1"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestRouter creates a router with real route registration
func setupTestRouter(usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) *gin.Engine {
	router := gin.New()
	adminCfg := AdminConfig{} // No admin required for tests
	RegisterDeckRoutes(router.Group("/"), usersClient, decksClient, adminCfg)
	return router
}

func TestRouteNotFound(t *testing.T) {
	decksClient := &mockDecksClient{}
	usersClient := &mockUsersClient{}
	router := setupTestRouter(usersClient, decksClient)

	// Trailing empty segments return 404 (route not matched)
	// This covers "missing" last segment params (cardId in UpdateCard, answerId in UpdateAnswer)
	t.Run("missing_card_id_trailing_slash", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{"title": "Test"})
		req := httptest.NewRequest(http.MethodPatch, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("missing_answer_id_trailing_slash", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{"text": "Test"})
		req := httptest.NewRequest(http.MethodPatch, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33/answers/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// Empty segments in middle (double slash) match route with empty param, handler validates
	t.Run("missing_deck_id_update_card", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{"title": "Test"})
		req := httptest.NewRequest(http.MethodPatch, "/decks//cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing deck id")
	})

	t.Run("missing_deck_id_update_answer", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{"text": "Test"})
		req := httptest.NewRequest(http.MethodPatch, "/decks//cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33/answers/7e6926da-82b2-4ae8-99b4-1b803ebf1877", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing deck id")
	})

	t.Run("missing_card_id_update_answer", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{"text": "Test"})
		req := httptest.NewRequest(http.MethodPatch, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards//answers/7e6926da-82b2-4ae8-99b4-1b803ebf1877", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing card id")
	})
}

func TestUpdateDeck(t *testing.T) {
	t.Run("success_update_title", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		title := "Updated Title"
		decksClient.On("UpdateDeck", mock.Anything, mock.MatchedBy(func(req *pbDeck.UpdateDeckRequest) bool {
			return req.Id == "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72" && *req.Title == title
		})).Return(&pbDeck.UpdateDeckResponse{
			Deck: &pbDeck.Deck{
				Id:          "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72",
				Title:       title,
				Description: "Original description",
			},
		}, nil)

		router := setupTestRouter(usersClient, decksClient)

		body, _ := json.Marshal(map[string]interface{}{"title": title})
		req := httptest.NewRequest(http.MethodPatch, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp deckResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, title, resp.Title)
	})

	t.Run("failure_invalid_uuid", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := setupTestRouter(usersClient, decksClient)

		body, _ := json.Marshal(map[string]interface{}{"title": "New Title"})
		req := httptest.NewRequest(http.MethodPatch, "/decks/invalid-uuid", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid deck id format")
	})

	t.Run("failure_no_fields_provided", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := setupTestRouter(usersClient, decksClient)

		body, _ := json.Marshal(map[string]interface{}{})
		req := httptest.NewRequest(http.MethodPatch, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "at least one field must be provided")
	})

	t.Run("failure_deck_not_found", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		title := "New Title"
		decksClient.On("UpdateDeck", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		router := setupTestRouter(usersClient, decksClient)

		body, _ := json.Marshal(map[string]interface{}{"title": title})
		req := httptest.NewRequest(http.MethodPatch, "/decks/00000000-0000-0000-0000-000000000000", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUpdateCard(t *testing.T) {
	t.Run("success_update_title", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		title := "Updated Card Title"
		decksClient.On("UpdateCard", mock.Anything, mock.MatchedBy(func(req *pbDeck.UpdateCardRequest) bool {
			return req.DeckId == "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72" &&
				req.CardId == "72bdff92-5bc8-4e1d-9217-d0b23e22ff33" &&
				*req.Title == title
		})).Return(&pbDeck.UpdateCardResponse{
			Card: &pbDeck.Card{
				Id:    "72bdff92-5bc8-4e1d-9217-d0b23e22ff33",
				Title: title,
				Kind:  "single_choice",
			},
		}, nil)

		router := setupTestRouter(usersClient, decksClient)

		body, _ := json.Marshal(map[string]interface{}{"title": title})
		req := httptest.NewRequest(http.MethodPatch, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp cardResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, title, resp.Title)
	})

	t.Run("failure_invalid_deck_uuid", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := setupTestRouter(usersClient, decksClient)

		body, _ := json.Marshal(map[string]interface{}{"title": "New Title"})
		req := httptest.NewRequest(http.MethodPatch, "/decks/invalid-uuid/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid deck id format")
	})

	t.Run("failure_invalid_card_uuid", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := setupTestRouter(usersClient, decksClient)

		body, _ := json.Marshal(map[string]interface{}{"title": "New Title"})
		req := httptest.NewRequest(http.MethodPatch, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/invalid-uuid", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid card id format")
	})

	t.Run("failure_no_fields_provided", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := setupTestRouter(usersClient, decksClient)

		body, _ := json.Marshal(map[string]interface{}{})
		req := httptest.NewRequest(http.MethodPatch, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "at least one field must be provided")
	})
}

func TestUpdateAnswer(t *testing.T) {
	t.Run("success_update_text", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		text := "Updated Answer Text"
		decksClient.On("UpdateAnswer", mock.Anything, mock.MatchedBy(func(req *pbDeck.UpdateAnswerRequest) bool {
			return req.DeckId == "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72" &&
				req.CardId == "72bdff92-5bc8-4e1d-9217-d0b23e22ff33" &&
				req.AnswerId == "7e6926da-82b2-4ae8-99b4-1b803ebf1877" &&
				*req.Text == text
		})).Return(&pbDeck.UpdateAnswerResponse{
			Answer: &pbDeck.Answer{
				Id:        "7e6926da-82b2-4ae8-99b4-1b803ebf1877",
				Text:      text,
				IsCorrect: true,
			},
		}, nil)

		router := setupTestRouter(usersClient, decksClient)

		body, _ := json.Marshal(map[string]interface{}{"text": text})
		req := httptest.NewRequest(http.MethodPatch, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33/answers/7e6926da-82b2-4ae8-99b4-1b803ebf1877", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp answerResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, text, resp.Text)
	})

	t.Run("success_update_is_correct", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		isCorrect := false
		decksClient.On("UpdateAnswer", mock.Anything, mock.MatchedBy(func(req *pbDeck.UpdateAnswerRequest) bool {
			return req.IsCorrect != nil && *req.IsCorrect == isCorrect
		})).Return(&pbDeck.UpdateAnswerResponse{
			Answer: &pbDeck.Answer{
				Id:        "7e6926da-82b2-4ae8-99b4-1b803ebf1877",
				Text:      "Original text",
				IsCorrect: isCorrect,
			},
		}, nil)

		router := setupTestRouter(usersClient, decksClient)

		body, _ := json.Marshal(map[string]interface{}{"is_correct": isCorrect})
		req := httptest.NewRequest(http.MethodPatch, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33/answers/7e6926da-82b2-4ae8-99b4-1b803ebf1877", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp answerResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.False(t, resp.IsCorrect)
	})

	t.Run("failure_invalid_answer_uuid", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := setupTestRouter(usersClient, decksClient)

		body, _ := json.Marshal(map[string]interface{}{"text": "New Text"})
		req := httptest.NewRequest(http.MethodPatch, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33/answers/invalid-uuid", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid answer id format")
	})

	t.Run("failure_no_fields_provided", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := setupTestRouter(usersClient, decksClient)

		body, _ := json.Marshal(map[string]interface{}{})
		req := httptest.NewRequest(http.MethodPatch, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33/answers/7e6926da-82b2-4ae8-99b4-1b803ebf1877", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "at least one field must be provided")
	})
}

type mockDecksClient struct {
	mock.Mock
}

func (m *mockDecksClient) GetDeck(ctx context.Context, req *pbDeck.GetDeckRequest, opts ...grpc.CallOption) (*pbDeck.GetDeckResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.GetDeckResponse), args.Error(1)
}

func (m *mockDecksClient) GetDecks(ctx context.Context, req *pbDeck.GetDecksRequest, opts ...grpc.CallOption) (*pbDeck.GetDecksResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.GetDecksResponse), args.Error(1)
}

func (m *mockDecksClient) CreateDeck(ctx context.Context, req *pbDeck.CreateDeckRequest, opts ...grpc.CallOption) (*pbDeck.CreateDeckResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.CreateDeckResponse), args.Error(1)
}

func (m *mockDecksClient) DeleteDeck(ctx context.Context, req *pbDeck.DeleteDeckRequest, opts ...grpc.CallOption) (*pbDeck.DeleteDeckResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.DeleteDeckResponse), args.Error(1)
}

func (m *mockDecksClient) GetPopularDecks(ctx context.Context, req *pbDeck.GetPopularDecksRequest, opts ...grpc.CallOption) (*pbDeck.GetPopularDecksResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.GetPopularDecksResponse), args.Error(1)
}

func (m *mockDecksClient) CreateCard(ctx context.Context, req *pbDeck.CreateCardRequest, opts ...grpc.CallOption) (*pbDeck.CreateCardResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.CreateCardResponse), args.Error(1)
}

func (m *mockDecksClient) GetCards(ctx context.Context, req *pbDeck.GetCardsRequest, opts ...grpc.CallOption) (*pbDeck.GetCardsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.GetCardsResponse), args.Error(1)
}

func (m *mockDecksClient) UpdateDeck(ctx context.Context, req *pbDeck.UpdateDeckRequest, opts ...grpc.CallOption) (*pbDeck.UpdateDeckResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.UpdateDeckResponse), args.Error(1)
}

func (m *mockDecksClient) UpdateCard(ctx context.Context, req *pbDeck.UpdateCardRequest, opts ...grpc.CallOption) (*pbDeck.UpdateCardResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.UpdateCardResponse), args.Error(1)
}

func (m *mockDecksClient) UpdateAnswer(ctx context.Context, req *pbDeck.UpdateAnswerRequest, opts ...grpc.CallOption) (*pbDeck.UpdateAnswerResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.UpdateAnswerResponse), args.Error(1)
}

type mockUsersClient struct {
	mock.Mock
}

func (m *mockUsersClient) SignUp(ctx context.Context, req *pbUser.SignUpRequest, opts ...grpc.CallOption) (*pbUser.SignUpResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbUser.SignUpResponse), args.Error(1)
}

func (m *mockUsersClient) LogIn(ctx context.Context, req *pbUser.LogInRequest, opts ...grpc.CallOption) (*pbUser.LogInResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbUser.LogInResponse), args.Error(1)
}

func (m *mockUsersClient) GetUserID(ctx context.Context, req *pbUser.GetUserIDRequest, opts ...grpc.CallOption) (*pbUser.GetUserIDResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbUser.GetUserIDResponse), args.Error(1)
}
