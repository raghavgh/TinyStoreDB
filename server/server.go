package server

import (
	"log"
	"net"

	tspb "github.com/raghavgh/TinyStoreDB/server/proto"
	"github.com/raghavgh/TinyStoreDB/store"
	"google.golang.org/grpc"
)

func StartGRPCServer(port string, kv *store.KVStore) error {
	err := kv.Replay()
	if err != nil {
		panic(err)
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	tspb.RegisterTinyStoreServiceServer(grpcServer, &TinyStoreHandler{
		Store: kv,
	})

	log.Printf("🚀 gRPC server listening on %s", port)
	return grpcServer.Serve(lis)
}
