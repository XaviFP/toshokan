package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mediocregopher/radix/v4"
	"github.com/tilinna/clock"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/XaviFP/toshokan/common/config"
	"github.com/XaviFP/toshokan/common/db"
	"github.com/XaviFP/toshokan/common/logging"
	"github.com/XaviFP/toshokan/course/internal/course"
	courseGRPC "github.com/XaviFP/toshokan/course/internal/grpc"
	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
)

var conf CourseConfig

func init() {
	conf = loadConfig()
}

func main() {
	logger := logging.Setup("course")

	db, err := db.InitDB(conf.DBConf)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	redisClient, err := (radix.PoolConfig{}).New(context.Background(), conf.CacheConf.TransportProtocol, fmt.Sprintf("%s:%s", conf.CacheConf.Host, conf.CacheConf.Port))
	if err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	pgRepo := course.NewPGRepository(logger.With("component", "pgRepository"), db)
	redisRepo := course.NewRedisRepository(redisClient, pgRepo)

	// Connect to deck service
	deckConn, err := grpc.Dial(
		fmt.Sprintf("%s:%s", conf.DeckGRPCHost, conf.DeckGRPCPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Warn("Failed to connect to deck service", "error", err)
	}
	defer deckConn.Close()

	var deckClient pbDeck.DecksAPIClient
	if deckConn != nil {
		deckClient = pbDeck.NewDecksAPIClient(deckConn)
	}

	realClock := clock.Realtime()
	srv := &courseGRPC.Server{
		GRPCTransport:  conf.GRPCConf.TransportProtocol,
		GRPCAddr:       fmt.Sprintf("%s:%s", conf.GRPCConf.Host, conf.GRPCConf.Port),
		Repository:     redisRepo,
		DeckClient:     deckClient,
		Enroller:       course.NewEnroller(realClock, redisRepo, deckClient),
		LessonsBrowser: course.NewLessonsBrowser(redisRepo, course.NewStateSyncer(redisRepo, deckClient)),
		CoursesBrowser: course.NewCoursesBrowser(redisRepo),
		Answerer:       course.NewAnswerer(redisRepo, deckClient),
		StateSyncer:    course.NewStateSyncer(redisRepo, deckClient),
		Clock:          realClock,
	}

	var serverError chan error

	go func() {
		if err := srv.Start(); err != nil {
			serverError <- err
		}
	}()

	defer srv.Stop()

	exitOnTerminationSignal(logger, serverError)
}

func exitOnTerminationSignal(logger *slog.Logger, serverError chan error) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigs:
	case err := <-serverError:
		logger.Error("gRPC server failure", "error", err)
	}

	logger.Info("Shutting down...")
	os.Exit(0)
}

type CourseConfig struct {
	DBConf       config.DBConfig
	CacheConf    config.CacheConfig
	GRPCConf     config.GRPCServerConfig
	DeckGRPCHost string
	DeckGRPCPort string
}

func loadConfig() CourseConfig {
	var courseConfig = CourseConfig{}

	courseConfig.DBConf = config.LoadDBConfig()
	courseConfig.CacheConf = config.LoadCacheConfig()
	courseConfig.GRPCConf = config.LoadGRPCServerConfig()
	courseConfig.DeckGRPCHost = os.Getenv("DECKS_GRPC_SERVER_HOST")
	if courseConfig.DeckGRPCHost == "" {
		courseConfig.DeckGRPCHost = "deck"
	}
	courseConfig.DeckGRPCPort = os.Getenv("DECKS_GRPC_SERVER_PORT")
	if courseConfig.DeckGRPCPort == "" {
		courseConfig.DeckGRPCPort = "50051" // TODO: no hardcoded ports
	}

	return courseConfig
}
