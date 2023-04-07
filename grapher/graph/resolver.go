package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"context"
	"fmt"

	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
	pbUser "github.com/XaviFP/toshokan/user/api/proto/v1"
	"github.com/gin-gonic/gin"
)

type Resolver struct {
	DeckClient pbDeck.DecksAPIClient
	UserClient pbUser.UserAPIClient
	DeckLoader DataLoader
	CardLoader DataLoader
}

func (r *Resolver) getUserID(ctx context.Context) string {
	gc, err := GinContextFromContext(ctx)
	if err != nil {
		// TODO: Log
		return ""
	}

	userID, found := gc.Get("userID")
	if !found {
		return ""
	}

	// should be uuid.Parse()
	out, _ := userID.(string)

	return out
}

func GinContextFromContext(ctx context.Context) (*gin.Context, error) {
	ginContext := ctx.Value("GinContextKey")
	if ginContext == nil {
		err := fmt.Errorf("could not retrieve gin.Context")
		return nil, err
	}

	gc, ok := ginContext.(*gin.Context)
	if !ok {
		err := fmt.Errorf("gin.Context has wrong type")
		return nil, err
	}

	return gc, nil
}
