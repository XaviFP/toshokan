package gate

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/juju/errors"

	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
	pbUser "github.com/XaviFP/toshokan/user/api/proto/v1"
)

func GetDeck(ctx *gin.Context, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	deckID := ctx.Param("id")
	if deckID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing deck id"})
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(deckID); err != nil {
		slog.Error("GetDeck: failed to parse deck ID", "error", err, "deckId", deckID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck id format"})
		return
	}

	req := &pbDeck.GetDeckRequest{DeckId: deckID, UserId: getUserID(ctx)}

	res, err := decksClient.GetDeck(ctx, req)
	if err != nil {
		// TODO: Handle these errors properly
		if strings.Contains(err.Error(), "deck: deck not found") {
			slog.Error("GetDeck: deck not found", "error", err, "deckId", deckID)
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		slog.Error("GetDeck: gRPC call failed", "error", err, "deckId", deckID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, toDeckResponse(res.Deck))
}

func CreateDeck(ctx *gin.Context, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	var d pbDeck.Deck
	if err := ctx.ShouldBindJSON(&d); err != nil {
		slog.Error("CreateDeck: failed to bind JSON", "error", err, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate that deck has at least one card
	if len(d.Cards) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "deck must have at least one card"})
		return
	}

	userID := getUserID(ctx)
	if userID == "" {
		slog.Error("CreateDeck: missing user ID from context")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	d.AuthorId = userID

	res, err := decksClient.CreateDeck(ctx, &pbDeck.CreateDeckRequest{Deck: &d})
	if err != nil {
		// Check if it's a validation error (invalid deck)
		// TODO: Handle these errors properly
		if strings.Contains(err.Error(), "deck: invalid deck") {
			slog.Error("CreateDeck: validation error", "error", err, "title", d.Title)
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Error("CreateDeck: gRPC call failed", "error", err, "title", d.Title, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, toDeckResponse(res.Deck))
}

func DeleteDeck(ctx *gin.Context, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing deck id"})
		return
	}

	if _, err := uuid.Parse(id); err != nil {
		slog.Error("DeleteDeck: failed to parse deck ID", "error", err, "deckId", id, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck id format"})
		return
	}

	_, err := decksClient.DeleteDeck(ctx, &pbDeck.DeleteDeckRequest{Id: id})
	if err != nil {
		slog.Error("DeleteDeck: gRPC call failed", "error", err, "deckId", id, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}

func UpdateDeck(ctx *gin.Context, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing deck id"})
		return
	}

	if _, err := uuid.Parse(id); err != nil {
		slog.Error("UpdateDeck: failed to parse deck ID", "error", err, "deckId", id, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck id format"})
		return
	}

	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		IsPublic    *bool   `json:"is_public"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Title == nil && req.Description == nil && req.IsPublic == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "at least one field must be provided"})
		return
	}

	res, err := decksClient.UpdateDeck(ctx, &pbDeck.UpdateDeckRequest{
		Id:          id,
		Title:       req.Title,
		Description: req.Description,
		IsPublic:    req.IsPublic,
	})
	if err != nil {
		if strings.Contains(err.Error(), "deck: deck not found") {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "deck not found"})
			return
		}
		slog.Error("UpdateDeck: gRPC call failed", "error", err, "deckId", id, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, toDeckResponse(res.Deck))
}

func UpdateCard(ctx *gin.Context, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	deckID := ctx.Param("deckId")
	cardID := ctx.Param("cardId")

	if deckID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing deck id"})
		return
	}
	if cardID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing card id"})
		return
	}

	if _, err := uuid.Parse(deckID); err != nil {
		slog.Error("UpdateCard: failed to parse deck ID", "error", err, "deckId", deckID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck id format"})
		return
	}
	if _, err := uuid.Parse(cardID); err != nil {
		slog.Error("UpdateCard: failed to parse card ID", "error", err, "cardId", cardID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id format"})
		return
	}

	var req struct {
		Title       *string `json:"title"`
		Explanation *string `json:"explanation"`
		Kind        *string `json:"kind"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Title == nil && req.Explanation == nil && req.Kind == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "at least one field must be provided"})
		return
	}

	res, err := decksClient.UpdateCard(ctx, &pbDeck.UpdateCardRequest{
		DeckId:      deckID,
		CardId:      cardID,
		Title:       req.Title,
		Explanation: req.Explanation,
		Kind:        req.Kind,
	})
	if err != nil {
		if strings.Contains(err.Error(), "deck: invalid card") {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			return
		}
		if strings.Contains(err.Error(), "deck: invalid kind") {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid card kind"})
			return
		}
		slog.Error("UpdateCard: gRPC call failed", "error", err, "deckId", deckID, "cardId", cardID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, toCardResponse(res.Card))
}

func UpdateAnswer(ctx *gin.Context, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	deckID := ctx.Param("deckId")
	cardID := ctx.Param("cardId")
	answerID := ctx.Param("answerId")

	if deckID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing deck id"})
		return
	}
	if cardID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing card id"})
		return
	}
	if answerID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing answer id"})
		return
	}

	if _, err := uuid.Parse(deckID); err != nil {
		slog.Error("UpdateAnswer: failed to parse deck ID", "error", err, "deckId", deckID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck id format"})
		return
	}
	if _, err := uuid.Parse(cardID); err != nil {
		slog.Error("UpdateAnswer: failed to parse card ID", "error", err, "cardId", cardID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id format"})
		return
	}
	if _, err := uuid.Parse(answerID); err != nil {
		slog.Error("UpdateAnswer: failed to parse answer ID", "error", err, "answerId", answerID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid answer id format"})
		return
	}

	var req struct {
		Text      *string `json:"text"`
		IsCorrect *bool   `json:"is_correct"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Text == nil && req.IsCorrect == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "at least one field must be provided"})
		return
	}

	res, err := decksClient.UpdateAnswer(ctx, &pbDeck.UpdateAnswerRequest{
		DeckId:    deckID,
		CardId:    cardID,
		AnswerId:  answerID,
		Text:      req.Text,
		IsCorrect: req.IsCorrect,
	})
	if err != nil {
		slog.Error("UpdateAnswer: gRPC call failed", "error", err, "deckId", deckID, "cardId", cardID, "answerId", answerID, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, toAnswerResponse(res.Answer))
}

func RegisterDeckRoutes(r *gin.RouterGroup, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient, adminCfg AdminConfig) {
	r.POST("/decks", RequireAdmin(adminCfg, adminCfg.CreateDeckAdminOnly), func(ctx *gin.Context) {
		CreateDeck(ctx, usersClient, decksClient)
	})

	r.DELETE("/decks/:id", func(ctx *gin.Context) {
		DeleteDeck(ctx, usersClient, decksClient)
	})

	r.GET("/decks/:id", func(ctx *gin.Context) {
		GetDeck(ctx, usersClient, decksClient)
	})

	r.PATCH("/decks/:id", RequireAdmin(adminCfg, adminCfg.UpdateDeckAdminOnly), func(ctx *gin.Context) {
		UpdateDeck(ctx, usersClient, decksClient)
	})

	r.PATCH("/decks/:deckId/cards/:cardId", RequireAdmin(adminCfg, adminCfg.UpdateCardAdminOnly), func(ctx *gin.Context) {
		UpdateCard(ctx, usersClient, decksClient)
	})

	r.PATCH("/decks/:deckId/cards/:cardId/answers/:answerId", RequireAdmin(adminCfg, adminCfg.UpdateAnswerAdminOnly), func(ctx *gin.Context) {
		UpdateAnswer(ctx, usersClient, decksClient)
	})
}

func RegisterMiddlewares(r *gin.RouterGroup, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	r.Use(func(ctx *gin.Context) {
		isAuthorized(ctx, usersClient)
	})
}

func isAuthorized(ctx *gin.Context, usersClient pbUser.UserAPIClient) {
	tok, ok := getRequestToken(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
		ctx.Abort()
		return
	}

	res, err := usersClient.GetUserID(ctx, &pbUser.GetUserIDRequest{
		By: &pbUser.GetUserIDRequest_Token{
			Token: tok,
		},
	})
	if err != nil {
		slog.Error("isAuthorized: failed to get user ID from token", "error", err, "stack", errors.ErrorStack(err))
		ctx.JSON(http.StatusUnauthorized, gin.H{})
		ctx.Abort()
		return
	}

	ctx.Set("userID", res.Id)
}

func getRequestToken(ctx *gin.Context) (string, bool) {
	aH := ctx.Request.Header["Authorization"]

	if len(aH) == 0 {
		return "", false
	}

	auth := strings.Split(aH[0], " ")
	if len(auth) != 2 {
		return "", false
	}

	tok := auth[1]
	return tok, true
}

func getUserID(ctx *gin.Context) string {
	userID, found := ctx.Get("userID")
	if !found {
		return ""
	}

	out, _ := userID.(string)

	return out
}

func GinContextToContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

type deckResponse struct {
	ID          string         `json:"id"`
	AuthorID    string         `json:"author_id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Cards       []cardResponse `json:"cards"`
}

type cardResponse struct {
	ID              string           `json:"id"`
	DeckID          string           `json:"deck_id"`
	Title           string           `json:"title"`
	PossibleAnswers []answerResponse `json:"possible_answers"`
	Explanation     string           `json:"explanation"`
	Kind            string           `json:"kind"`
}

type answerResponse struct {
	ID        string `json:"id"`
	CardID    string `json:"card_id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
}

func toDeckResponse(d *pbDeck.Deck) deckResponse {
	if d == nil {
		return deckResponse{}
	}

	out := deckResponse{
		ID:          d.GetId(),
		AuthorID:    d.GetAuthorId(),
		Title:       d.GetTitle(),
		Description: d.GetDescription(),
	}

	if len(d.Cards) > 0 {
		out.Cards = make([]cardResponse, 0, len(d.Cards))
		for _, c := range d.Cards {
			out.Cards = append(out.Cards, toCardResponse(c))
		}
	}

	return out
}

func toCardResponse(c *pbDeck.Card) cardResponse {
	if c == nil {
		return cardResponse{}
	}

	out := cardResponse{
		ID:          c.GetId(),
		DeckID:      c.GetDeckId(),
		Title:       c.GetTitle(),
		Explanation: c.GetExplanation(),
		Kind:        c.GetKind(),
	}

	if len(c.PossibleAnswers) > 0 {
		out.PossibleAnswers = make([]answerResponse, 0, len(c.PossibleAnswers))
		for _, a := range c.PossibleAnswers {
			out.PossibleAnswers = append(out.PossibleAnswers, toAnswerResponse(a))
		}
	}

	return out
}

func toAnswerResponse(a *pbDeck.Answer) answerResponse {
	if a == nil {
		return answerResponse{}
	}

	return answerResponse{
		ID:        a.GetId(),
		CardID:    a.GetCardId(),
		Text:      a.GetText(),
		IsCorrect: a.GetIsCorrect(),
	}
}
