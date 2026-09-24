package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables.
// MustLoad panics on startup if any required variable is missing — fail fast by design.
type Config struct {
	Port           string
	AllowedOrigins string
	DatabaseURL    string
	RedisURL       string
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    bool
	MinioBucket    string
	JWTSecret      string
	JWTAccessTTLH  int
	JWTRefreshTTLD int
	IPHashSalt     string
}

// MustLoad reads all required env vars. Panics if any required vars are missing.
// This is intentional — a misconfigured app should fail fast at boot, not at runtime.
func MustLoad() Config {
	return Config{
		Port:           mustGetenv("PORT"),
		AllowedOrigins: mustGetenv("ALLOWED_ORIGINS"),
		DatabaseURL:    mustGetenv("DATABASE_URL"),
		RedisURL:       mustGetenv("REDIS_URL"),
		MinioEndpoint:  mustGetenv("MINIO_ENDPOINT"),
		MinioAccessKey: mustGetenv("MINIO_ACCESS_KEY"),
		MinioSecretKey: mustGetenv("MINIO_SECRET_KEY"),
		MinioUseSSL:    getenvBool("MINIO_USE_SSL", false),
		MinioBucket:    getenvDefault("MINIO_BUCKET_UPLOADS", "uploads"),
		JWTSecret:      mustGetenv("JWT_SECRET"),
		JWTAccessTTLH:  getenvInt("JWT_ACCESS_TTL_HOURS", 24),
		JWTRefreshTTLD: getenvInt("JWT_REFRESH_TTL_DAYS", 7),
		IPHashSalt:     mustGetenv("IP_HASH_SALT"),
	}
}

func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}
