package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbCourse "github.com/XaviFP/toshokan/course/api/proto/v1"
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
	allowedOrigins  []string
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
	course config
}

func corsMiddleware(allowed []string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		allowedSet[strings.TrimSpace(a)] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if _, ok := allowedSet[origin]; ok {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
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

	coursesGRPCConn, err := grpc.Dial(c.course.GRPCAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer coursesGRPCConn.Close()

	coursesClient := pbCourse.NewCourseAPIClient(coursesGRPCConn)

	router := gin.Default()
	// Return 405 for routes that exist but do not support the requested HTTP method
	router.HandleMethodNotAllowed = true

	router.Use(gate.GinContextToContextMiddleware())
	router.Use(corsMiddleware(c.gate.allowedOrigins))

	queryPath := "/query"
	router.GET("/play", grapher.NewPlaygroundHandler(queryPath))

	authorized := router.Group("/")
	gate.RegisterMiddlewares(authorized, userClient, deckClient)
	authorized.POST(queryPath, grapher.NewGraphqlHandler(deckClient, userClient, dealerClient))
	gate.RegisterDeckRoutes(authorized, userClient, deckClient)
	gate.RegisterUserRoutes(router, c.gate.signupEnabled, userClient)
	gate.RegisterCoursesRoutes(authorized, coursesClient)

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

	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins != "" {
		gateConfig.allowedOrigins = strings.Split(allowedOrigins, ",")
	}

	usersConfig := config{}
	usersConfig.grpcHost = os.Getenv("USERS_GRPC_SERVER_HOST")
	usersConfig.grpcPort = os.Getenv("USERS_GRPC_SERVER_PORT")

	decksConfig := config{}
	decksConfig.grpcHost = os.Getenv("DECKS_GRPC_SERVER_HOST")
	decksConfig.grpcPort = os.Getenv("DECKS_GRPC_SERVER_PORT")

	dealerConfig := config{}
	dealerConfig.grpcHost = os.Getenv("DEALER_GRPC_SERVER_HOST")
	dealerConfig.grpcPort = os.Getenv("DEALER_GRPC_SERVER_PORT")

	coursesConfig := config{}
	coursesConfig.grpcHost = os.Getenv("COURSE_GRPC_SERVER_HOST")
	coursesConfig.grpcPort = os.Getenv("COURSE_GRPC_SERVER_PORT")

	return globalConfig{
		gate:   gateConfig,
		users:  usersConfig,
		decks:  decksConfig,
		dealer: dealerConfig,
		course: coursesConfig,
	}
}
