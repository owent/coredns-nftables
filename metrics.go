package coredns_nftables

import (
	"sync"

	"github.com/coredns/coredns/plugin"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// recordCount exports a prometheus metric that is incremented every time a query is seen by the example plugin.
var recordCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: plugin.Namespace,
	Subsystem: "nftables",
	Name:      "record_count_total",
	Help:      "Counter of requests processed.",
}, []string{"server"})

var recordDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: plugin.Namespace,
	Subsystem: "nftables",
	Name:      "record_duration_microseconds",
	Buckets:   plugin.TimeBuckets,
	Help:      "Histogram of the time each record took.",
}, []string{"server"})

var _ sync.Once
