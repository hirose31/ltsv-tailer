package metrics

import "github.com/prometheus/client_golang/prometheus"

// Counter contains prometheus.CounterVec.
type Counter struct {
	Name     string
	ValueKey string
	Labels   []string
	Metric   *prometheus.CounterVec
}

// Histogram contains prometheus.HistogramVec.
type Histogram struct {
	Name     string
	ValueKey string
	Labels   []string
	Metric   *prometheus.HistogramVec
}
