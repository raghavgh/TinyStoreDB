package server

import (
	"context"
	"log"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	tspb "github.com/raghavgh/TinyStoreDB/server/proto"
	"github.com/raghavgh/TinyStoreDB/store"
)

type TinyStoreHandler struct {
	tspb.UnimplementedTinyStoreServiceServer
	Store *store.KVStore
}

func (h *TinyStoreHandler) Get(_ context.Context, req *tspb.GetRequest) (*tspb.GetResponse, error) {
	method := "Get"
	timer := prometheus.NewTimer(GRPCLatency.WithLabelValues(method))
	defer timer.ObserveDuration()

	value, err := h.Store.Get(req.Key)
	if err != nil {
		return nil, err
	}

	slog.Info("Get request for key, value", req.Key, value)

	return &tspb.GetResponse{
		Value: value,
	}, nil
}

func (h *TinyStoreHandler) Set(_ context.Context, req *tspb.SetRequest) (*tspb.SetResponse, error) {
	method := "Set"
	timer := prometheus.NewTimer(GRPCLatency.WithLabelValues(method))
	defer timer.ObserveDuration()

	err := h.Store.Set(req.Key, req.Value, req.Ttl)
	if err != nil {
		return nil, err
	}

	log.Printf("Set request for key: %s, value: %s", req.Key, req.Value)

	return &tspb.SetResponse{
		Success: true,
	}, nil
}

func (h *TinyStoreHandler) Delete(
	_ context.Context,
	req *tspb.DeleteRequest,
) (*tspb.DeleteResponse, error) {
	method := "Delete"
	timer := prometheus.NewTimer(GRPCLatency.WithLabelValues(method))
	defer timer.ObserveDuration()

	err := h.Store.Delete(req.Key)
	if err != nil {
		return nil, err
	}

	log.Printf("Delete request for key: %s", req.Key)

	return &tspb.DeleteResponse{
		Success: true,
	}, nil
}

func (h *TinyStoreHandler) Exist(
	_ context.Context,
	req *tspb.ExistRequest,
) (*tspb.ExistResponse, error) {
	method := "Exist"
	timer := prometheus.NewTimer(GRPCLatency.WithLabelValues(method))
	defer timer.ObserveDuration()

	exist, err := h.Store.Exist(req.Key)
	if err != nil {
		return nil, err
	}

	log.Printf("Exist request for key: %s, exist: %t", req.Key, exist)

	return &tspb.ExistResponse{
		Value: exist,
	}, nil
}

func (h *TinyStoreHandler) Compact(
	_ context.Context,
	req *tspb.CompactRequest,
) (*tspb.CompactResponse, error) {
	method := "Compact"
	timer := prometheus.NewTimer(GRPCLatency.WithLabelValues(method))
	defer timer.ObserveDuration()

	err := h.Store.Compact()
	if err != nil {
		log.Printf("Compaction failed: %v", err)
		return nil, err
	}

	log.Printf("Compact request")

	return &tspb.CompactResponse{
		Success: true,
	}, nil
}
