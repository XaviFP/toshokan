package grapher

import (
	"context"
	"log"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/playground"
	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
	"github.com/XaviFP/toshokan/grapher/graph"
	"github.com/XaviFP/toshokan/grapher/graph/generated"
	pbUser "github.com/XaviFP/toshokan/user/api/proto/v1"
	"github.com/gin-gonic/gin"
)

func NewGraphqlHandler(deckClient pbDeck.DecksAPIClient, userClient pbUser.UserAPIClient) gin.HandlerFunc {
	h := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{
		DeckClient: deckClient,
		UserClient: userClient,
		DeckLoader: graph.NewDataLoader(
			NewDeckBatchFn(deckClient),
			time.Minute*30,
			time.Millisecond*16,
		),
	}}))

	h.Use(extension.Introspection{})

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func NewPlaygroundHandler(targetPath string) gin.HandlerFunc {
	h := playground.Handler("GraphQL", targetPath)

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func NewDeckBatchFn(client pbDeck.DecksAPIClient) func(ctx context.Context, ids []string) map[string]any {
	return func(ctx context.Context, ids []string) map[string]any {
		out := make(map[string]any, len(ids))

		res, err := client.GetDecks(ctx, &pbDeck.GetDecksRequest{DeckIds: ids})
		if err != nil {
			log.Default().Printf("could not get decks: %s", err) // TODO: Return errors instead
			return out
		}

		for _, d := range res.Decks {
			out[d.Id] = d
		}

		return out
	}
}
