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
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
	pbUser "github.com/XaviFP/toshokan/user/api/proto/v1"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCreateCard(t *testing.T) {
	card := map[string]interface{}{
		"title": "What does CSP stand for?",
		"kind":  "single_choice",
		"possible_answers": []map[string]interface{}{
			{"text": "Communicating Sequential Processes", "is_correct": true},
		},
	}

	t.Run("success", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		decksClient.On("CreateCard", mock.Anything, mock.MatchedBy(func(req *pbDeck.CreateCardRequest) bool {
			return req.Card.DeckId == "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72" &&
				req.Card.Title == "What does CSP stand for?" &&
				req.Card.Kind == "single_choice" &&
				len(req.Card.PossibleAnswers) == 1 &&
				req.Card.PossibleAnswers[0].Text == "Communicating Sequential Processes" &&
				req.Card.PossibleAnswers[0].IsCorrect
		})).Return(&pbDeck.CreateCardResponse{
			Card: &pbDeck.Card{
				Id:     "72bdff92-5bc8-4e1d-9217-d0b23e22ff33",
				DeckId: "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72",
				Title:  "What does CSP stand for?",
				Kind:   "single_choice",
				PossibleAnswers: []*pbDeck.Answer{
					{
						Id:        "7e6926da-82b2-4ae8-99b4-1b803ebf1877",
						CardId:    "72bdff92-5bc8-4e1d-9217-d0b23e22ff33",
						Text:      "Communicating Sequential Processes",
						IsCorrect: true,
					},
				},
			},
		}, nil)

		router := gin.New()
		router.POST("/decks/:deckId/cards", func(ctx *gin.Context) {
			CreateCard(ctx, usersClient, decksClient)
		})

		body, _ := json.Marshal(card)
		req := httptest.NewRequest(http.MethodPost, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var actual cardResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &actual))

		expected := cardResponse{
			ID:     "72bdff92-5bc8-4e1d-9217-d0b23e22ff33",
			DeckID: "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72",
			Title:  "What does CSP stand for?",
			Kind:   "single_choice",
			PossibleAnswers: []answerResponse{
				{
					ID:        "7e6926da-82b2-4ae8-99b4-1b803ebf1877",
					CardID:    "72bdff92-5bc8-4e1d-9217-d0b23e22ff33",
					Text:      "Communicating Sequential Processes",
					IsCorrect: true,
				},
			},
		}
		assert.Equal(t, expected, actual)
	})

	t.Run("failure_invalid_deck_uuid", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		router.POST("/decks/:deckId/cards", func(ctx *gin.Context) {
			CreateCard(ctx, usersClient, decksClient)
		})

		body, _ := json.Marshal(card)
		req := httptest.NewRequest(http.MethodPost, "/decks/invalid-uuid/cards", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid deck id format")
	})

	t.Run("failure_malformed_body", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		router.POST("/decks/:deckId/cards", func(ctx *gin.Context) {
			CreateCard(ctx, usersClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodPost, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards", bytes.NewReader([]byte("{")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("failure_invalid_card", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		decksClient.On("CreateCard", mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.InvalidArgument, "deck: invalid card"))

		router := gin.New()
		router.POST("/decks/:deckId/cards", func(ctx *gin.Context) {
			CreateCard(ctx, usersClient, decksClient)
		})

		body, _ := json.Marshal(card)
		req := httptest.NewRequest(http.MethodPost, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid card")
	})

	t.Run("failure_deck_not_found", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		decksClient.On("CreateCard", mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.NotFound, "deck: deck not found"))

		router := gin.New()
		router.POST("/decks/:deckId/cards", func(ctx *gin.Context) {
			CreateCard(ctx, usersClient, decksClient)
		})

		body, _ := json.Marshal(card)
		req := httptest.NewRequest(http.MethodPost, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "deck not found")
	})

	t.Run("failure_missing_admin_header", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		RegisterDeckRoutes(router.Group("/"), usersClient, decksClient, AdminConfig{
			HeaderName:          "X-Admin-Token",
			HeaderSecret:        "secret123",
			CreateCardAdminOnly: true,
		})

		body, _ := json.Marshal(card)
		req := httptest.NewRequest(http.MethodPost, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		decksClient.AssertNotCalled(t, "CreateCard", mock.Anything, mock.Anything)
	})
}

func TestDeleteDeck(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		decksClient.On("DeleteDeck", mock.Anything, mock.MatchedBy(func(req *pbDeck.DeleteDeckRequest) bool {
			return req.Id == "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72"
		})).Return(&pbDeck.DeleteDeckResponse{}, nil)

		router := gin.New()
		router.DELETE("/decks/:deckId", func(ctx *gin.Context) {
			DeleteDeck(ctx, usersClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodDelete, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("success_with_admin_header", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		decksClient.On("DeleteDeck", mock.Anything, &pbDeck.DeleteDeckRequest{Id: "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72"}).Return(&pbDeck.DeleteDeckResponse{}, nil)

		router := gin.New()
		RegisterDeckRoutes(router.Group("/"), usersClient, decksClient, AdminConfig{
			HeaderName:   "X-Admin-Token",
			HeaderSecret: "secret123",
		})

		req := httptest.NewRequest(http.MethodDelete, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72", nil)
		req.Header.Set("X-Admin-Token", "secret123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		decksClient.AssertExpectations(t)
	})

	t.Run("failure_missing_admin_header", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		RegisterDeckRoutes(router.Group("/"), usersClient, decksClient, AdminConfig{
			HeaderName:   "X-Admin-Token",
			HeaderSecret: "secret123",
		})

		req := httptest.NewRequest(http.MethodDelete, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		decksClient.AssertNotCalled(t, "DeleteDeck", mock.Anything, mock.Anything)
	})

	t.Run("failure_missing_id", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		router.DELETE("/decks/:deckId", func(ctx *gin.Context) {
			ctx.Params = gin.Params{} // Clear params to simulate missing id
			DeleteDeck(ctx, usersClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodDelete, "/decks/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing deck id")
	})

	t.Run("failure_invalid_uuid", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		router.DELETE("/decks/:deckId", func(ctx *gin.Context) {
			DeleteDeck(ctx, usersClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodDelete, "/decks/invalid-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid deck id format")
	})

	t.Run("failure_grpc_error", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		decksClient.On("DeleteDeck", mock.Anything, mock.MatchedBy(func(req *pbDeck.DeleteDeckRequest) bool {
			return req.Id == "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72"
		})).Return(nil, assert.AnError)

		router := gin.New()
		router.DELETE("/decks/:deckId", func(ctx *gin.Context) {
			DeleteDeck(ctx, usersClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodDelete, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("failure_deck_not_found", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		notFoundErr := status.Error(codes.NotFound, "deck: deck not found")
		decksClient.On("DeleteDeck", mock.Anything, mock.MatchedBy(func(req *pbDeck.DeleteDeckRequest) bool {
			return req.Id == "00000000-0000-0000-0000-000000000000"
		})).Return(nil, notFoundErr)

		router := gin.New()
		router.DELETE("/decks/:deckId", func(ctx *gin.Context) {
			DeleteDeck(ctx, usersClient, decksClient)
		})

		req := httptest.NewRequest(http.MethodDelete, "/decks/00000000-0000-0000-0000-000000000000", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestDeleteCard(t *testing.T) {
	adminCfg := AdminConfig{
		HeaderName:   "X-Admin-Token",
		HeaderSecret: "secret123",
	}

	t.Run("success", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		decksClient.On("DeleteCard", mock.Anything, &pbDeck.DeleteCardRequest{
			DeckId: "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72",
			CardId: "72bdff92-5bc8-4e1d-9217-d0b23e22ff33",
		}).Return(&pbDeck.DeleteCardResponse{}, nil)

		router := gin.New()
		RegisterDeckRoutes(router.Group("/"), usersClient, decksClient, adminCfg)

		req := httptest.NewRequest(http.MethodDelete, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33", nil)
		req.Header.Set("X-Admin-Token", "secret123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())
		decksClient.AssertExpectations(t)
	})

	t.Run("failure_missing_admin_header", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		RegisterDeckRoutes(router.Group("/"), usersClient, decksClient, adminCfg)

		req := httptest.NewRequest(http.MethodDelete, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		decksClient.AssertNotCalled(t, "DeleteCard", mock.Anything, mock.Anything)
	})

	t.Run("failure_invalid_deck_id", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		RegisterDeckRoutes(router.Group("/"), usersClient, decksClient, adminCfg)

		req := httptest.NewRequest(http.MethodDelete, "/decks/invalid-uuid/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33", nil)
		req.Header.Set("X-Admin-Token", "secret123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid deck id format")
		decksClient.AssertNotCalled(t, "DeleteCard", mock.Anything, mock.Anything)
	})

	t.Run("failure_invalid_card_id", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		RegisterDeckRoutes(router.Group("/"), usersClient, decksClient, adminCfg)

		req := httptest.NewRequest(http.MethodDelete, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/invalid-uuid", nil)
		req.Header.Set("X-Admin-Token", "secret123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid card id format")
		decksClient.AssertNotCalled(t, "DeleteCard", mock.Anything, mock.Anything)
	})

	t.Run("failure_card_not_found", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		decksClient.On("DeleteCard", mock.Anything, mock.Anything).Return(nil, status.Error(codes.NotFound, "deck: card not found"))

		router := gin.New()
		RegisterDeckRoutes(router.Group("/"), usersClient, decksClient, adminCfg)

		req := httptest.NewRequest(http.MethodDelete, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/00000000-0000-0000-0000-000000000000", nil)
		req.Header.Set("X-Admin-Token", "secret123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "card not found")
	})

	t.Run("failure_grpc_error", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		decksClient.On("DeleteCard", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		router := gin.New()
		RegisterDeckRoutes(router.Group("/"), usersClient, decksClient, adminCfg)

		req := httptest.NewRequest(http.MethodDelete, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33", nil)
		req.Header.Set("X-Admin-Token", "secret123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
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

		router := gin.New()
		router.PATCH("/decks/:deckId", func(ctx *gin.Context) {
			UpdateDeck(ctx, usersClient, decksClient)
		})

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

	t.Run("failure_missing_id", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		router.PATCH("/decks/:deckId", func(ctx *gin.Context) {
			ctx.Params = gin.Params{} // Clear params to simulate missing id
			UpdateDeck(ctx, usersClient, decksClient)
		})

		body, _ := json.Marshal(map[string]interface{}{"title": "New Title"})
		req := httptest.NewRequest(http.MethodPatch, "/decks/test", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing deck id")
	})

	t.Run("failure_invalid_uuid", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		router.PATCH("/decks/:deckId", func(ctx *gin.Context) {
			UpdateDeck(ctx, usersClient, decksClient)
		})

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

		router := gin.New()
		router.PATCH("/decks/:deckId", func(ctx *gin.Context) {
			UpdateDeck(ctx, usersClient, decksClient)
		})

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
		notFoundErr := status.Error(codes.NotFound, "deck: deck not found")
		decksClient.On("UpdateDeck", mock.Anything, mock.MatchedBy(func(req *pbDeck.UpdateDeckRequest) bool {
			return req.Id == "00000000-0000-0000-0000-000000000000" && *req.Title == title
		})).Return(nil, notFoundErr)

		router := gin.New()
		router.PATCH("/decks/:deckId", func(ctx *gin.Context) {
			UpdateDeck(ctx, usersClient, decksClient)
		})

		body, _ := json.Marshal(map[string]interface{}{"title": title})
		req := httptest.NewRequest(http.MethodPatch, "/decks/00000000-0000-0000-0000-000000000000", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
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

		router := gin.New()
		router.PATCH("/decks/:deckId/cards/:cardId", func(ctx *gin.Context) {
			UpdateCard(ctx, usersClient, decksClient)
		})

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

	t.Run("failure_missing_deck_id", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		router.PATCH("/decks/:deckId/cards/:cardId", func(ctx *gin.Context) {
			ctx.Params = gin.Params{{Key: "cardId", Value: "72bdff92-5bc8-4e1d-9217-d0b23e22ff33"}}
			UpdateCard(ctx, usersClient, decksClient)
		})

		body, _ := json.Marshal(map[string]interface{}{"title": "New Title"})
		req := httptest.NewRequest(http.MethodPatch, "/decks/test/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing deck id")
	})

	t.Run("failure_missing_card_id", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		router.PATCH("/decks/:deckId/cards/:cardId", func(ctx *gin.Context) {
			ctx.Params = gin.Params{{Key: "deckId", Value: "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72"}}
			UpdateCard(ctx, usersClient, decksClient)
		})

		body, _ := json.Marshal(map[string]interface{}{"title": "New Title"})
		req := httptest.NewRequest(http.MethodPatch, "/decks/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/cards/test", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing card id")
	})

	t.Run("failure_invalid_deck_uuid", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		router.PATCH("/decks/:deckId/cards/:cardId", func(ctx *gin.Context) {
			UpdateCard(ctx, usersClient, decksClient)
		})

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

		router := gin.New()
		router.PATCH("/decks/:deckId/cards/:cardId", func(ctx *gin.Context) {
			UpdateCard(ctx, usersClient, decksClient)
		})

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

		router := gin.New()
		router.PATCH("/decks/:deckId/cards/:cardId", func(ctx *gin.Context) {
			UpdateCard(ctx, usersClient, decksClient)
		})

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

		router := gin.New()
		router.PATCH("/decks/:deckId/cards/:cardId/answers/:answerId", func(ctx *gin.Context) {
			UpdateAnswer(ctx, usersClient, decksClient)
		})

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

		router := gin.New()
		router.PATCH("/decks/:deckId/cards/:cardId/answers/:answerId", func(ctx *gin.Context) {
			UpdateAnswer(ctx, usersClient, decksClient)
		})

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

	t.Run("failure_missing_deck_id", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		router.PATCH("/decks/:deckId/cards/:cardId/answers/:answerId", func(ctx *gin.Context) {
			ctx.Params = gin.Params{
				{Key: "cardId", Value: "72bdff92-5bc8-4e1d-9217-d0b23e22ff33"},
				{Key: "answerId", Value: "7e6926da-82b2-4ae8-99b4-1b803ebf1877"},
			}
			UpdateAnswer(ctx, usersClient, decksClient)
		})

		body, _ := json.Marshal(map[string]interface{}{"text": "New Text"})
		req := httptest.NewRequest(http.MethodPatch, "/decks/test/cards/72bdff92-5bc8-4e1d-9217-d0b23e22ff33/answers/7e6926da-82b2-4ae8-99b4-1b803ebf1877", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing deck id")
	})

	t.Run("failure_invalid_answer_uuid", func(t *testing.T) {
		decksClient := &mockDecksClient{}
		usersClient := &mockUsersClient{}

		router := gin.New()
		router.PATCH("/decks/:deckId/cards/:cardId/answers/:answerId", func(ctx *gin.Context) {
			UpdateAnswer(ctx, usersClient, decksClient)
		})

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

		router := gin.New()
		router.PATCH("/decks/:deckId/cards/:cardId/answers/:answerId", func(ctx *gin.Context) {
			UpdateAnswer(ctx, usersClient, decksClient)
		})

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

func (m *mockDecksClient) DeleteCard(ctx context.Context, req *pbDeck.DeleteCardRequest, opts ...grpc.CallOption) (*pbDeck.DeleteCardResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.DeleteCardResponse), args.Error(1)
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

func (m *mockUsersClient) GetUsersByIDs(ctx context.Context, req *pbUser.GetUsersByIDsRequest, opts ...grpc.CallOption) (*pbUser.GetUsersByIDsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbUser.GetUsersByIDsResponse), args.Error(1)
}