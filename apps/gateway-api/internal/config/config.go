package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppName           string
	HTTPPort          string
	MySQLDSN          string
	MySQLMaxOpenConns int
	MySQLMaxIdleConns int
	MySQLConnLifetime time.Duration
	MySQLConnIdleTime time.Duration
	MongoURI          string
	MongoDatabase     string
	MongoMaxPoolSize  uint64
	MongoMinPoolSize  uint64
	MongoMaxIdleTime  time.Duration
	KeycloakIssuerURL string
	KeycloakJWKSBase  string
	KeycloakClientID  string
	TemporalAddress   string
	TemporalTaskQueue string
	InventoryGRPCAddr string
	NotifierGRPCAddr  string
	RequestTimeout    time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		AppName:           getenvDefault("APP_NAME", "gateway-api"),
		HTTPPort:          getenvDefault("HTTP_PORT", "8080"),
		MySQLDSN:          os.Getenv("MYSQL_DSN"),
		MongoURI:          os.Getenv("MONGO_URI"),
		MongoDatabase:     getenvDefault("MONGO_DATABASE", "order_playground"),
		KeycloakIssuerURL: os.Getenv("KEYCLOAK_ISSUER_URL"),
		KeycloakJWKSBase:  os.Getenv("KEYCLOAK_JWKS_BASE_URL"),
		KeycloakClientID:  os.Getenv("KEYCLOAK_CLIENT_ID"),
		TemporalAddress:   getenvDefault("TEMPORAL_ADDRESS", "localhost:7233"),
		TemporalTaskQueue: getenvDefault("TEMPORAL_TASK_QUEUE", "order-playground"),
		InventoryGRPCAddr: getenvDefault("INVENTORY_GRPC_ADDRESS", "localhost:9091"),
		NotifierGRPCAddr:  getenvDefault("NOTIFIER_GRPC_ADDRESS", "localhost:9092"),
	}

	var err error
	if cfg.MySQLDSN == "" {
		return cfg, fmt.Errorf("MYSQL_DSN is required")
	}
	if cfg.MongoURI == "" {
		return cfg, fmt.Errorf("MONGO_URI is required")
	}
	if cfg.KeycloakIssuerURL == "" {
		return cfg, fmt.Errorf("KEYCLOAK_ISSUER_URL is required")
	}
	if cfg.KeycloakJWKSBase == "" {
		cfg.KeycloakJWKSBase = cfg.KeycloakIssuerURL
	}
	if cfg.KeycloakClientID == "" {
		return cfg, fmt.Errorf("KEYCLOAK_CLIENT_ID is required")
	}

	if cfg.MySQLMaxOpenConns, err = getenvInt("MYSQL_MAX_OPEN_CONNS", 20); err != nil {
		return cfg, err
	}
	if cfg.MySQLMaxIdleConns, err = getenvInt("MYSQL_MAX_IDLE_CONNS", 10); err != nil {
		return cfg, err
	}
	if cfg.MySQLConnLifetime, err = getenvDuration("MYSQL_CONN_MAX_LIFETIME", 30*time.Minute); err != nil {
		return cfg, err
	}
	if cfg.MySQLConnIdleTime, err = getenvDuration("MYSQL_CONN_MAX_IDLE_TIME", 5*time.Minute); err != nil {
		return cfg, err
	}
	if cfg.MongoMaxPoolSize, err = getenvUint64("MONGO_MAX_POOL_SIZE", 20); err != nil {
		return cfg, err
	}
	if cfg.MongoMinPoolSize, err = getenvUint64("MONGO_MIN_POOL_SIZE", 5); err != nil {
		return cfg, err
	}
	if cfg.MongoMaxIdleTime, err = getenvDuration("MONGO_MAX_IDLE_TIME", 5*time.Minute); err != nil {
		return cfg, err
	}
	if cfg.RequestTimeout, err = getenvDuration("REQUEST_TIMEOUT", 10*time.Second); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func getenvDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func getenvUint64(key string, fallback uint64) (uint64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	return strconv.ParseUint(value, 10, 64)
}

func getenvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	return time.ParseDuration(value)
}
