package config

import (
	"log/slog"
	"os"
)

type DBConfig struct {
	User     string
	Password string
	Name     string
	Host     string
	Port     string
}

type KeySet struct {
	PublicKey  []byte
	PrivateKey []byte
}

type CacheConfig struct {
	Host              string
	Port              string
	TransportProtocol string
}

type GRPCServerConfig struct {
	Host              string
	Port              string
	TransportProtocol string
}

type TokenConfig struct {
	KeySet
	SessionExpiry uint
}

var (
	dbHost     = "DB_HOST"
	dbPort     = "DB_PORT"
	dbName     = "DB_NAME"
	dbUser     = "DB_USER"
	dbPassword = "DB_PASSWORD"
)

func LoadDBConfig() DBConfig {
	config := DBConfig{}

	host := os.Getenv(dbHost)
	if host == "" {
		slog.Error("missing environment variable", "name", dbHost)
		os.Exit(1)
	}

	config.Host = host

	port := os.Getenv(dbPort)
	if port == "" {
		slog.Error("missing environment variable", "name", dbPort)
		os.Exit(1)
	}

	config.Port = port

	name := os.Getenv(dbName)
	if name == "" {
		slog.Error("missing environment variable", "name", dbName)
		os.Exit(1)
	}

	config.Name = name

	user := os.Getenv(dbUser)
	if user == "" {
		slog.Error("missing environment variable", "name", dbUser)
		os.Exit(1)
	}

	config.User = user

	password := os.Getenv(dbPassword)
	if password == "" {
		slog.Error("missing environment variable", "name", dbPassword)
		os.Exit(1)
	}

	config.Password = password

	return config
}

var (
	cacheHost              = "CACHE_HOST"
	cachePort              = "CACHE_PORT"
	cacheTransportProtocol = "CACHE_TRANSPORT_PROTOCOL"
)

func LoadCacheConfig() CacheConfig {
	config := CacheConfig{}

	host := os.Getenv(cacheHost)
	if host == "" {
		slog.Error("missing environment variable", "name", cacheHost)
		os.Exit(1)
	}

	config.Host = host

	port := os.Getenv(cachePort)
	if port == "" {
		slog.Error("missing environment variable", "name", cachePort)
		os.Exit(1)
	}

	config.Port = port

	transportProtocol := os.Getenv(cacheTransportProtocol)
	if transportProtocol == "" {
		slog.Error("missing environment variable", "name", cacheTransportProtocol)
		os.Exit(1)
	}

	config.TransportProtocol = transportProtocol

	return config
}

var (
	gRPCServerHost              = "GRPC_SERVER_HOST"
	gRPCServerPort              = "GRPC_SERVER_PORT"
	gRPCServerTransportProtocol = "GRPC_SERVER_TRANSPORT_PROTOCOL"
)

func LoadGRPCServerConfig() GRPCServerConfig {
	config := GRPCServerConfig{}

	host := os.Getenv(gRPCServerHost)
	if host == "" {
		slog.Error("missing environment variable", "name", gRPCServerHost)
		os.Exit(1)
	}

	config.Host = host

	port := os.Getenv(gRPCServerPort)
	if port == "" {
		slog.Error("missing environment variable", "name", gRPCServerPort)
		os.Exit(1)
	}

	config.Port = port

	transportProtocol := os.Getenv(gRPCServerTransportProtocol)
	if transportProtocol == "" {
		slog.Error("missing environment variable", "name", gRPCServerTransportProtocol)
		os.Exit(1)
	}

	config.TransportProtocol = transportProtocol

	return config
}
