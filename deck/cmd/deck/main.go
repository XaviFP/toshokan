package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mediocregopher/radix/v4"

	"github.com/XaviFP/toshokan/common/config"
	"github.com/XaviFP/toshokan/common/db"
	"github.com/XaviFP/toshokan/deck/internal/deck"
	"github.com/XaviFP/toshokan/deck/internal/grpc"
)

var conf DecksConfig

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

	exitOnTerminationSignal(serverError)
}

func exitOnTerminationSignal(serverError chan error) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigs:
	case err := <-serverError:
		log.Printf("GRPC server failure: %s\n", err)
	}

	os.Exit(1)
	log.Println("Shutting down...")
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
