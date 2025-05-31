package server

import "github.com/prometheus/client_golang/prometheus"

var (
	GRPCRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests received.",
		},
		[]string{"method"},
	)

	GRPCErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_errors_total",
			Help: "Total number of gRPC requests that errored.",
		},
		[]string{"method"},
	)

	GRPCLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_latency_seconds",
			Help:    "Histogram of response latency (seconds).",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)
)

func InitMetrics() {
	prometheus.MustRegister(GRPCRequests, GRPCErrors, GRPCLatency)
}
