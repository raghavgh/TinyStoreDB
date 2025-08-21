package cmd

import (
	"log"
	"strconv"

	"github.com/raghavgh/TinyStoreDB/client"
)

func getClient() (*client.TinyStoreClient, error) {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	secret := func() *string {
		if cfg.Secret == "" {
			return nil
		}

		return &cfg.Secret
	}

	return client.New(cfg.Host+":"+strconv.Itoa(cfg.Port), secret())
}
