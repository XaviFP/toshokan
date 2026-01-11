package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mediocregopher/radix/v4"
	"github.com/tilinna/clock"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/XaviFP/toshokan/common/config"
	"github.com/XaviFP/toshokan/common/db"
	"github.com/XaviFP/toshokan/course/internal/course"
	courseGRPC "github.com/XaviFP/toshokan/course/internal/grpc"
	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
)

var conf CourseConfig

func init() {
	conf = loadConfig()
}

func main() {
	db, err := db.InitDB(conf.DBConf)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	redisClient, err := (radix.PoolConfig{}).New(context.Background(), conf.CacheConf.TransportProtocol, fmt.Sprintf("%s:%s", conf.CacheConf.Host, conf.CacheConf.Port))
	if err != nil {
		panic(err)
	}
	defer redisClient.Close()

	pgRepo := course.NewPGRepository(db)
	redisRepo := course.NewRedisRepository(redisClient, pgRepo)

	// Connect to deck service
	deckConn, err := grpc.Dial(
		fmt.Sprintf("%s:%s", conf.DeckGRPCHost, conf.DeckGRPCPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("warning: failed to connect to deck service: %v", err)
	}
	defer deckConn.Close()

	var deckClient pbDeck.DecksAPIClient
	if deckConn != nil {
		deckClient = pbDeck.NewDecksAPIClient(deckConn)
	}

	realClock := clock.Realtime()
	srv := &courseGRPC.Server{
		GRPCTransport:   conf.GRPCConf.TransportProtocol,
		GRPCAddr:        fmt.Sprintf("%s:%s", conf.GRPCConf.Host, conf.GRPCConf.Port),
		Repository:      redisRepo,
		DeckClient:      deckClient,
		Enroller:        course.NewEnroller(realClock, redisRepo, deckClient),
		LessonsBrowser:  course.NewLessonsBrowser(redisRepo),
		CoursesBrowser:  course.NewCoursesBrowser(redisRepo),
		Answerer:        course.NewAnswerer(redisRepo, course.NewStateSyncer(redisRepo, deckClient), deckClient),
		Clock:           realClock,
	}

	var serverError chan error

	go func() {
		if err := srv.Start(); err != nil {
			serverError <- err
		}
	}()

	defer srv.Stop()

	exitOnTerminationSignal(serverError)
}

func exitOnTerminationSignal(serverError chan error) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigs:
	case err := <-serverError:
		log.Printf("gRPC server failure: %s\n", err)
	}

	os.Exit(0)
	log.Println("Shutting down...")
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
