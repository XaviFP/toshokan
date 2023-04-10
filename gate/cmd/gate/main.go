package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbDealer "github.com/XaviFP/toshokan/dealer/api/proto/v1"
	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
	"github.com/XaviFP/toshokan/gate/internal/gate"
	"github.com/XaviFP/toshokan/grapher"
	pbUser "github.com/XaviFP/toshokan/user/api/proto/v1"
)

type config struct {
	httpHost string
	httpPort string
	grpcHost string
	grpcPort string
}

type gateConfig struct {
	config
	signupEnabled   bool
	certificatePath string
	privateKeyPath  string
}

func (c gateConfig) canListenTLS() bool {
	return c.certificatePath != "" && c.privateKeyPath != ""
}

func (c config) HTTPAddress() string {
	return fmt.Sprintf("%s:%s", c.httpHost, c.httpPort)
}

func (c config) GRPCAddress() string {
	return fmt.Sprintf("%s:%s", c.grpcHost, c.grpcPort)
}

type globalConfig struct {
	gate   gateConfig
	users  config
	decks  config
	dealer config
}

func main() {
	c := loadConfig()

	userGRPCConn, err := grpc.Dial(c.users.GRPCAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer userGRPCConn.Close()

	userClient := pbUser.NewUserAPIClient(userGRPCConn)

	deckGRPCConn, err := grpc.Dial(c.decks.GRPCAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer deckGRPCConn.Close()

	deckClient := pbDeck.NewDecksAPIClient(deckGRPCConn)

	dealerGRPCConn, err := grpc.Dial(c.dealer.GRPCAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer dealerGRPCConn.Close()

	dealerClient := pbDealer.NewDealerClient(dealerGRPCConn)

	router := gin.Default()
	router.Use(gate.GinContextToContextMiddleware())

	queryPath := "/query"
	router.GET("/play", grapher.NewPlaygroundHandler(queryPath))

	authorized := router.Group("/")
	gate.RegisterMiddlewares(authorized, userClient, deckClient)
	authorized.POST(queryPath, grapher.NewGraphqlHandler(deckClient, userClient, dealerClient))
	gate.RegisterDeckRoutes(authorized, userClient, deckClient)
	gate.RegisterUserRoutes(router, c.gate.signupEnabled, userClient)

	if c.gate.canListenTLS() {
		if err := router.RunTLS("", c.gate.certificatePath, c.gate.privateKeyPath); err != nil {
			panic(err)
		}
	} else {
		if err := router.Run(c.gate.HTTPAddress()); err != nil {
			panic(err)
		}
	}
}

func loadConfig() globalConfig {
	gateConfig := gateConfig{}
	gateConfig.httpHost = os.Getenv("HTTP_HOST")
	gateConfig.httpPort = os.Getenv("HTTP_PORT")

	signupEnabled, err := strconv.ParseBool(os.Getenv("SIGNUP_ENABLED"))
	if err != nil {
		panic(errors.Annotate(err, "wrong or missing configuration value for SIGNUP_ENABLED"))
	}
	gateConfig.signupEnabled = signupEnabled

	gateConfig.certificatePath = os.Getenv("CERTIFICATE_PATH")
	gateConfig.privateKeyPath = os.Getenv("PRIVATE_KEY_PATH")

	usersConfig := config{}
	usersConfig.grpcHost = os.Getenv("USERS_GRPC_SERVER_HOST")
	usersConfig.grpcPort = os.Getenv("USERS_GRPC_SERVER_PORT")

	decksConfig := config{}
	decksConfig.grpcHost = os.Getenv("DECKS_GRPC_SERVER_HOST")
	decksConfig.grpcPort = os.Getenv("DECKS_GRPC_SERVER_PORT")

	dealerConfig := config{}
	dealerConfig.grpcHost = os.Getenv("DEALER_GRPC_SERVER_HOST")
	dealerConfig.grpcPort = os.Getenv("DEALER_GRPC_SERVER_PORT")

	return globalConfig{
		gate:   gateConfig,
		users:  usersConfig,
		decks:  decksConfig,
		dealer: dealerConfig,
	}
}
