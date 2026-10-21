package server

import (
	"log"
	"net"

	"github.com/raghavgh/TinyStoreDB/config"
	tspb "github.com/raghavgh/TinyStoreDB/server/proto"
	"github.com/raghavgh/TinyStoreDB/store"
	"google.golang.org/grpc"
)

// NewGRPCServer creates and configures a gRPC server for TinyStoreDB.
func NewGRPCServer(kv *store.KVStore, secret *string) *grpc.Server {
	var grpcServer *grpc.Server

	if secret != nil {
		interceptor := AuthInterceptor(config.Unpack(secret))
		grpcServer = grpc.NewServer(grpc.UnaryInterceptor(interceptor))
	} else {
		grpcServer = grpc.NewServer()
	}

	tspb.RegisterTinyStoreServiceServer(grpcServer, &TinyStoreHandler{
		Store: kv,
	})

	return grpcServer
}

// StartGRPCServer initializes the store replay, binds to the configured port, and starts serving.
func StartGRPCServer(cfg *config.Config, kv *store.KVStore) error {
	err := kv.Replay()
	if err != nil {
		panic(err)
	}

	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		return err
	}

	grpcServer := NewGRPCServer(kv, cfg.Secret)
	log.Printf("🚀 gRPC server listening on %s", cfg.Port)
	return grpcServer.Serve(lis)
}
