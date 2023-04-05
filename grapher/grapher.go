package grapher

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/playground"
	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
	"github.com/XaviFP/toshokan/grapher/graph"
	"github.com/XaviFP/toshokan/grapher/graph/generated"
	pbUser "github.com/XaviFP/toshokan/user/api/proto/v1"
	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
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

func NewDeckBatchFn(client pbDeck.DecksAPIClient) graph.BatchFn {
	return func(ctx context.Context, ids []string) (map[string]graph.Result, error) {
		out := make(map[string]graph.Result, len(ids))

		res, err := client.GetDecks(ctx, &pbDeck.GetDecksRequest{DeckIds: ids})
		if err != nil {
			return out, errors.Trace(err)
		}

		for _, d := range res.Decks {
			out[d.Id] = graph.Result{Value: d}
		}

		return out, nil
	}
}
