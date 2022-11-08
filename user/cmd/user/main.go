package main

import (
	"context"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/mediocregopher/radix/v4"

	"github.com/XaviFP/toshokan/common/config"
	"github.com/XaviFP/toshokan/common/db"
	"github.com/XaviFP/toshokan/user/internal/grpc"
	"github.com/XaviFP/toshokan/user/internal/user"
)

var conf UsersConfig

func init() {
	conf = loadConfig()
}

func main() {
	db, err := db.InitDB(conf.DBConf)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	defer db.Close()

	pgRepo := user.NewPGRepository(db)

	redis, err := (radix.PoolConfig{}).New(context.Background(), conf.CacheConf.TransportProtocol, fmt.Sprintf("%s:%s", conf.CacheConf.Host, conf.CacheConf.Port))
	if err != nil {
		panic(err)
	}

	defer redis.Close()

	repository := user.NewRedisRepository(redis, pgRepo)

	tokenRepository, err := user.NewTokenRepository(conf.TokenConfig)
	if err != nil {
		panic(err)
	}

	srv := &grpc.Server{
		GRPCTransport:   conf.GRPCConf.TransportProtocol,
		GRPCAddr:        fmt.Sprintf("%s:%s", conf.GRPCConf.Host, conf.GRPCConf.Port),
		Creator:         user.NewCreator(repository),
		Authorizer:      user.NewAuthorizer(repository, tokenRepository),
		Repository:      repository,
		TokenRepository: tokenRepository,
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

type UsersConfig struct {
	DBConf      config.DBConfig
	CacheConf   config.CacheConfig
	GRPCConf    config.GRPCServerConfig
	TokenConfig config.TokenConfig
}

func loadConfig() UsersConfig {
	var usersConfig = UsersConfig{}

	usersConfig.GRPCConf = config.LoadGRPCServerConfig()
	usersConfig.DBConf = config.LoadDBConfig()
	usersConfig.CacheConf = config.LoadCacheConfig()
	usersConfig.TokenConfig = loadTokenConfig()

	return usersConfig
}

func loadTokenConfig() config.TokenConfig {
	var tokenConfig = config.TokenConfig{}

	sExp := os.Getenv("SESSION_EXPIRY")
	if sExp == "" {
		log.Fatal("missing environment variable: SESSION_EXPIRY")
	}

	var sessionExpiry uint
	if _, err := fmt.Sscan(sExp, &sessionExpiry); err != nil {
		log.Fatal("wrong format for environment variable: SESSION_EXPIRY")
	}

	tokenConfig.SessionExpiry = sessionExpiry

	pubKey := os.Getenv("PUBLIC_KEY")
	if pubKey == "" {
		log.Fatal("missing environment variable: PUBLIC_KEY")
	}

	block, _ := pem.Decode([]byte(pubKey))
	tokenConfig.PublicKey = block.Bytes[len(block.Bytes)-32:]

	privKeyPem := os.Getenv("PRIVATE_KEY")
	if privKeyPem == "" {
		log.Fatal("missing environment variable: PRIVATE_KEY")
	}

	block, _ = pem.Decode([]byte(privKeyPem))
	privKey := block.Bytes[len(block.Bytes)-32:]

	// private key contains public key
	// represented in its last 32 bytes
	privKey = append(privKey, tokenConfig.PublicKey...)
	tokenConfig.PrivateKey = privKey

	return tokenConfig
}
