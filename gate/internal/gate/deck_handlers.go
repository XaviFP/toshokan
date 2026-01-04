package gate

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck id format"})
		return
	}

	req := &pbDeck.GetDeckRequest{DeckId: deckID, UserId: getUserID(ctx)}

	res, err := decksClient.GetDeck(ctx, req)
	if err != nil {
		// TODO: Handle these errors properly
		if strings.Contains(err.Error(), "deck: deck not found") {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, toDeckResponse(res.Deck))
}

func CreateDeck(ctx *gin.Context, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	var d pbDeck.Deck
	if err := ctx.ShouldBindJSON(&d); err != nil {
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	d.AuthorId = userID

	res, err := decksClient.CreateDeck(ctx, &pbDeck.CreateDeckRequest{Deck: &d})
	if err != nil {
		// Check if it's a validation error (invalid deck)
		// TODO: Handle these errors properly
		if strings.Contains(err.Error(), "deck: invalid deck") {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck id format"})
		return
	}

	_, err := decksClient.DeleteDeck(ctx, &pbDeck.DeleteDeckRequest{Id: id})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}

func RegisterDeckRoutes(r *gin.RouterGroup, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	r.POST("/decks", func(ctx *gin.Context) {
		CreateDeck(ctx, usersClient, decksClient)
	})

	r.DELETE("/decks/:id", func(ctx *gin.Context) {
		DeleteDeck(ctx, usersClient, decksClient)
	})

	r.GET("/decks/:id", func(ctx *gin.Context) {
		GetDeck(ctx, usersClient, decksClient)
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
