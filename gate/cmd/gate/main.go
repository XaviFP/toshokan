package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
	"github.com/XaviFP/toshokan/gate/internal/gate"
	pbUser "github.com/XaviFP/toshokan/user/api/proto/v1"
)

type config struct {
	httpHost string
	httpPort string
	grpcHost string
	grpcPort string
}

func (c config) HTTPAddress() string {
	return fmt.Sprintf("%s:%s", c.httpHost, c.httpPort)
}

func (c config) GRPCAddress() string {
	return fmt.Sprintf("%s:%s", c.grpcHost, c.grpcPort)
}

type globalConfig struct {
	gate  config
	users config
	decks config
}

func main() {
	c := loadConfig()

	userGRPConn, err := grpc.Dial(c.users.GRPCAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer userGRPConn.Close()

	userClient := pbUser.NewUserAPIClient(userGRPConn)

	deckGRPCConn, err := grpc.Dial(c.decks.GRPCAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer deckGRPCConn.Close()

	deckClient := pbDeck.NewDecksAPIClient(deckGRPCConn)

	router := gin.Default()
	authorized := router.Group("/")
	gate.RegisterMiddlewares(authorized, userClient, deckClient)
	gate.RegisterlibraryRoutes(authorized, userClient, deckClient)
	gate.RegisterUserRoutes(router, userClient)

	if err := router.Run(c.gate.HTTPAddress()); err != nil {
		panic(err)
	}
}

func loadConfig() globalConfig {
	gateConfig := config{}
	gateConfig.httpHost = os.Getenv("HTTP_HOST")
	gateConfig.httpPort = os.Getenv("HTTP_PORT")

	usersConfig := config{}
	usersConfig.grpcHost = os.Getenv("USERS_GRPC_SERVER_HOST")
	usersConfig.grpcPort = os.Getenv("USERS_GRPC_SERVER_PORT")

	decksConfig := config{}
	decksConfig.grpcHost = os.Getenv("DECKS_GRPC_SERVER_HOST")
	decksConfig.grpcPort = os.Getenv("DECKS_GRPC_SERVER_PORT")

	return globalConfig{
		gate:  gateConfig,
		users: usersConfig,
		decks: decksConfig,
	}
}
