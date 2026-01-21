package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal counts total requests by method and status
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shortener_requests_total",
			Help: "Total number of requests by method and status",
		},
		[]string{"method", "status"},
	)

	// RequestDuration tracks request duration by method
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "shortener_request_duration_seconds",
			Help:    "Request duration in seconds by method",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	// CacheHits counts cache hits by layer (L1, L2)
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shortener_cache_hits_total",
			Help: "Total number of cache hits by layer",
		},
		[]string{"layer"},
	)

	// CacheMisses counts cache misses by layer (L1, L2)
	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shortener_cache_misses_total",
			Help: "Total number of cache misses by layer",
		},
		[]string{"layer"},
	)

	// ErrorsTotal counts errors by type
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shortener_errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"type"},
	)

	// SingleflightWaits counts singleflight wait events
	SingleflightWaits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "shortener_singleflight_waits_total",
			Help: "Total number of singleflight wait events",
		},
	)

	// RedirectsTotal counts successful redirects
	RedirectsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "shortener_redirects_total",
			Help: "Total number of successful redirects",
		},
	)

	// LinksCreated counts created short links
	LinksCreated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "shortener_links_created_total",
			Help: "Total number of short links created",
		},
	)

	// LinksDeleted counts deleted short links
	LinksDeleted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "shortener_links_deleted_total",
			Help: "Total number of short links deleted",
		},
	)

	// ClickEventsLogged counts click events logged to Kafka
	// Requirements: 7.1, 7.2
	ClickEventsLogged = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "shortener_click_events_logged_total",
			Help: "Total number of click events logged to Kafka",
		},
	)
)
