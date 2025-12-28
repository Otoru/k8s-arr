package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	torrentSearchesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "torrent_searches_total",
			Help: "Total number of searches performed per indexer",
		},
		[]string{"indexer", "status"},
	)

	torrentRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "torrent_request_duration_seconds",
			Help:    "Time taken to process a TorrentRequest end-to-end",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
		},
		[]string{"status"},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(torrentSearchesTotal, torrentRequestDuration)
}
