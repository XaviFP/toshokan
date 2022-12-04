package grapher

import (
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
