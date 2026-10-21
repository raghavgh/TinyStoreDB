package cmd

import (
	"strconv"

	"github.com/raghavgh/TinyStoreDB/client"
)

func getClient() (*client.TinyStoreClient, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	if hostFlag != "" {
		cfg.Host = hostFlag
	}
	if portFlag != 0 {
		cfg.Port = portFlag
	}
	if secretFlag != "" {
		cfg.Secret = secretFlag
	}

	var secret *string
	if cfg.Secret != "" {
		secret = &cfg.Secret
	}

	return client.New(cfg.Host+":"+strconv.Itoa(cfg.Port), secret)
}
