package gate

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
	pbUser "github.com/XaviFP/toshokan/user/api/proto/v1"
	pbDealer "github.com/XaviFP/toshokan/dealer/api/proto/v1"
)

func GetDeck(ctx *gin.Context, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	deckID := ctx.Param("id")
	if deckID == "" {
		ctx.IndentedJSON(http.StatusBadRequest, "missing deck id")
		return
	}

	req := &pbDeck.GetDeckRequest{DeckId: deckID, UserId: getUserID(ctx)}

	res, err := decksClient.GetDeck(ctx, req)
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, err)
		return
	}

	ctx.IndentedJSON(http.StatusOK, res.Deck)
}

func GetDecks(ctx *gin.Context, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	userID := getUserID(ctx)
	if userID == "" {
		ctx.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	req := &pbDeck.GetDecksRequest{UserId: userID}

	res, err := decksClient.GetDecks(ctx, req)
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, err)
		return
	}

	ctx.IndentedJSON(http.StatusOK, res.Decks)
}

func CreateDeck(ctx *gin.Context, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	var d pbDeck.Deck
	if err := ctx.BindJSON(&d); err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, err)
		return
	}

	userID := getUserID(ctx)
	if userID == "" {
		ctx.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	d.AuthorId = userID

	res, err := decksClient.CreateDeck(ctx, &pbDeck.CreateDeckRequest{Deck: &d})
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.IndentedJSON(http.StatusOK, res.Deck)
}

func DeleteDeck(ctx *gin.Context, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	id := ctx.Param("id")
	if id == "" {
		ctx.IndentedJSON(http.StatusBadRequest, nil)
		return
	}

	_, err := decksClient.DeleteDeck(ctx, &pbDeck.DeleteDeckRequest{Id: id})
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.IndentedJSON(http.StatusOK, nil)
}

func RegisterDeckRoutes(r *gin.RouterGroup, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient) {
	r.GET("/decks/:id", func(ctx *gin.Context) {
		GetDeck(ctx, usersClient, decksClient)
	})

	r.GET("/decks", func(ctx *gin.Context) {
		GetDecks(ctx, usersClient, decksClient)
	})

	r.POST("/decks/create", func(ctx *gin.Context) {
		CreateDeck(ctx, usersClient, decksClient)
	})

	r.POST("/decks/delete/:id", func(ctx *gin.Context) {
		DeleteDeck(ctx, usersClient, decksClient)
	})
}

func RegisterMiddlewares(r *gin.RouterGroup, usersClient pbUser.UserAPIClient, decksClient pbDeck.DecksAPIClient, dealerClient pbDealer.DealerClient) {
	r.Use(func(ctx *gin.Context) {
		isAuthorized(ctx, usersClient)
		dealerClient.Deal(context.Background(), &pbDealer.DealRequest{UserId: "", DeckId: "", NumberOfCards: 0})
	})
}

func isAuthorized(ctx *gin.Context, usersClient pbUser.UserAPIClient) {
	tok, ok := getRequestToken(ctx)
	if !ok {
		ctx.IndentedJSON(http.StatusUnauthorized, nil)
		ctx.Abort()
		return
	}

	res, err := usersClient.GetUserID(ctx, &pbUser.GetUserIDRequest{
		By: &pbUser.GetUserIDRequest_Token{
			Token: tok,
		},
	})
	if err != nil {
		ctx.IndentedJSON(http.StatusUnauthorized, nil)
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
