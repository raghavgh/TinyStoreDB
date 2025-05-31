package config

import (
	"flag"
	"os"
)

var (
	// Config is the global configuration variable.
	config *Config
)

type Config struct {
	Port     string
	DataDir  string
	LogLevel string
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
	port := flag.String("port", getEnvOrDefault("TINYSTOREDB_PORT", "7389"), "Port to run TinyStoreDB")
	dataDir := flag.String("data-dir", getEnvOrDefault("TINYSTOREDB_DATA_DIR", "data"), "Data directory")
	logLevel := flag.String("log-level", getEnvOrDefault("TINYSTOREDB_LOG_LEVEL", "info"), "Log level")

	flag.Parse()

	return &Config{
		Port:     *port,
		DataDir:  *dataDir,
		LogLevel: *logLevel,
	}
}
