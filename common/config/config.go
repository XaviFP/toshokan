package config

import (
	"log"
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
	var config = DBConfig{}

	host := os.Getenv(dbHost)
	if host == "" {
		log.Fatalf("missing environment variable: %s", dbHost)
	}

	config.Host = host

	port := os.Getenv(dbPort)
	if port == "" {
		log.Fatalf("missing environment variable: %s", dbPort)
	}

	config.Port = port

	name := os.Getenv(dbName)
	if name == "" {
		log.Fatalf("missing environment variable: %s", dbName)
	}

	config.Name = name

	user := os.Getenv(dbUser)
	if user == "" {
		log.Fatalf("missing environment variable: %s", dbUser)
	}

	config.User = user

	password := os.Getenv(dbPassword)
	if password == "" {
		log.Fatalf("missing environment variable: %s", dbPassword)
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
	var config = CacheConfig{}

	host := os.Getenv(cacheHost)
	if host == "" {
		log.Fatalf("missing environment variable: %s", cacheHost)
	}

	config.Host = host

	port := os.Getenv(cachePort)
	if port == "" {
		log.Fatalf("missing environment variable: %s", cachePort)
	}

	config.Port = port

	transportProtocol := os.Getenv(cacheTransportProtocol)
	if transportProtocol == "" {
		log.Fatalf("missing environment variable: %s", cacheTransportProtocol)
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
	var config = GRPCServerConfig{}

	host := os.Getenv(gRPCServerHost)
	if host == "" {
		log.Fatalf("missing environment variable: %s", gRPCServerHost)
	}

	config.Host = host

	port := os.Getenv(gRPCServerPort)
	if port == "" {
		log.Fatalf("missing environment variable: %s", gRPCServerPort)
	}

	config.Port = port

	transportProtocol := os.Getenv(gRPCServerTransportProtocol)
	if transportProtocol == "" {
		log.Fatalf("missing environment variable: %s", gRPCServerTransportProtocol)
	}

	config.TransportProtocol = transportProtocol

	return config
}
