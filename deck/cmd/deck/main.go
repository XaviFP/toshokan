package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mediocregopher/radix/v4"

	"github.com/XaviFP/toshokan/common/config"
	"github.com/XaviFP/toshokan/common/db"
	"github.com/XaviFP/toshokan/common/logging"
	"github.com/XaviFP/toshokan/deck/internal/deck"
	"github.com/XaviFP/toshokan/deck/internal/grpc"
)

var conf DecksConfig

func init() {
	conf = loadConfig()
}

func main() {
	logger := logging.Setup("deck")

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

	postgresRepo := deck.NewPGRepository(db)
	redisRepo := deck.NewRedisRepository(redisClient, postgresRepo)

	srv := &grpc.Server{
		GRPCTransport: conf.GRPCConf.TransportProtocol,
		GRPCAddr:      fmt.Sprintf("%s:%s", conf.GRPCConf.Host, conf.GRPCConf.Port),
		Repository:    redisRepo,
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

type DecksConfig struct {
	DBConf    config.DBConfig
	CacheConf config.CacheConfig
	GRPCConf  config.GRPCServerConfig
}

func loadConfig() DecksConfig {
	var decksConfig = DecksConfig{}

	decksConfig.GRPCConf = config.LoadGRPCServerConfig()
	decksConfig.DBConf = config.LoadDBConfig()
	decksConfig.CacheConf = config.LoadCacheConfig()

	return decksConfig
}
