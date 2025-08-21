package config

import (
	"flag"
	"os"
)

// Config is the global configuration variable.
var config *Config

type Config struct {
	Port     string
	DataDir  string
	LogLevel string
	Secret   *string
}

func getEnvOrDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func Load() *Config {
	if config == nil {
		config = loadConfig()
	}

	return config
}

func loadConfig() *Config {
	port := flag.String(
		"port",
		getEnvOrDefault("TINYSTOREDB_PORT", "7389"),
		"Port to run TinyStoreDB",
	)
	dataDir := flag.String(
		"data-dir",
		getEnvOrDefault("TINYSTOREDB_DATA_DIR", "data"),
		"Data directory",
	)
	logLevel := flag.String(
		"log-level",
		getEnvOrDefault("TINYSTOREDB_LOG_LEVEL", "info"),
		"Log level",
	)
	secret := flag.String(
		"secret",
		getEnvOrDefault("TINYSTOREDB_SECRET", "secret"),
		"Secret to use",
	)

	if Unpack(secret) == "secret" {
		secret = nil
	}

	flag.Parse()

	return &Config{
		Port:     *port,
		DataDir:  *dataDir,
		LogLevel: *logLevel,
		Secret:   secret,
	}
}

func Unpack[T any](p *T) T {
	var zero T

	if p == nil {
		return zero
	}

	return *p
}
