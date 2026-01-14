package main

import (
	"context"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/mediocregopher/radix/v4"

	"github.com/XaviFP/toshokan/common/config"
	"github.com/XaviFP/toshokan/common/db"
	"github.com/XaviFP/toshokan/common/logging"
	"github.com/XaviFP/toshokan/user/internal/grpc"
	"github.com/XaviFP/toshokan/user/internal/user"
)

var conf UsersConfig

func init() {
	conf = loadConfig()
}

func main() {
	logger := logging.Setup("user")

	db, err := db.InitDB(conf.DBConf)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	defer db.Close()

	pgRepo := user.NewPGRepository(db)

	redis, err := (radix.PoolConfig{}).New(context.Background(), conf.CacheConf.TransportProtocol, fmt.Sprintf("%s:%s", conf.CacheConf.Host, conf.CacheConf.Port))
	if err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	defer redis.Close()

	repository := user.NewRedisRepository(redis, pgRepo)

	tokenRepository, err := user.NewTokenRepository(conf.TokenConfig)
	if err != nil {
		logger.Error("Failed to create token repository", "error", err)
		os.Exit(1)
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
		slog.Error("Missing environment variable: SESSION_EXPIRY")
		os.Exit(1)
	}

	var sessionExpiry uint
	if _, err := fmt.Sscan(sExp, &sessionExpiry); err != nil {
		slog.Error("Wrong format for environment variable: SESSION_EXPIRY", "error", err)
		os.Exit(1)
	}

	tokenConfig.SessionExpiry = sessionExpiry

	pubKey := os.Getenv("PUBLIC_KEY")
	if pubKey == "" {
		slog.Error("Missing environment variable: PUBLIC_KEY")
		os.Exit(1)
	}

	block, _ := pem.Decode([]byte(pubKey))
	tokenConfig.PublicKey = block.Bytes[len(block.Bytes)-32:]

	privKeyPem := os.Getenv("PRIVATE_KEY")
	if privKeyPem == "" {
		slog.Error("Missing environment variable: PRIVATE_KEY")
		os.Exit(1)
	}

	block, _ = pem.Decode([]byte(privKeyPem))
	privKey := block.Bytes[len(block.Bytes)-32:]

	// private key contains public key
	// represented in its last 32 bytes
	privKey = append(privKey, tokenConfig.PublicKey...)
	tokenConfig.PrivateKey = privKey

	return tokenConfig
}
