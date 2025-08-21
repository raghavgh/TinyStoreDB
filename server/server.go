package server

import (
	"log"
	"net"

	"github.com/raghavgh/TinyStoreDB/config"
	tspb "github.com/raghavgh/TinyStoreDB/server/proto"
	"github.com/raghavgh/TinyStoreDB/store"
	"google.golang.org/grpc"
)

func StartGRPCServer(cfg *config.Config, kv *store.KVStore) error {
	err := kv.Replay()
	if err != nil {
		panic(err)
	}

	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		return err
	}

	var grpcServer *grpc.Server

	if cfg.Secret != nil {
		interceptor := AuthInterceptor(config.Unpack(cfg.Secret))
		grpcServer = grpc.NewServer(grpc.UnaryInterceptor(interceptor))
	} else {
		grpcServer = grpc.NewServer()
	}

	tspb.RegisterTinyStoreServiceServer(grpcServer, &TinyStoreHandler{
		Store: kv,
	})

	log.Printf("🚀 gRPC server listening on %s", cfg.Port)
	return grpcServer.Serve(lis)
}
