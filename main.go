/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/raghavgh/TinyStoreDB/cmd"
	"github.com/raghavgh/TinyStoreDB/config"
	"github.com/raghavgh/TinyStoreDB/server"
	"github.com/raghavgh/TinyStoreDB/store"
)

func main() {
	server.InitMetrics()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Printf("📈 Prometheus metrics at http://localhost:2112/metrics")
		err := http.ListenAndServe(":2112", nil)
		if err != nil {
			log.Fatalf("metrics server failed: %v", err)
		}
	}()

	go func() {
		cmd.Execute()
	}()

	cfg := config.Load()

	kv, err := store.NewKVStore(cfg)
	if err != nil {
		log.Fatalf("Failed to create KVStore: %v\n", err)
	}

	// Start the gRPC server
	if err := server.StartGRPCServer(cfg.Port, kv); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}
}
