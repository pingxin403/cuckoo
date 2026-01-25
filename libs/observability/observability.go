package observability

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"

	"github.com/pingxin403/cuckoo/libs/observability/logging"
	"github.com/pingxin403/cuckoo/libs/observability/metrics"
	"github.com/pingxin403/cuckoo/libs/observability/tracing"
)

// Observability provides a unified interface for metrics, tracing, and logging
type Observability interface {
	// Metrics returns the metrics collector
	Metrics() metrics.Collector

	// Tracer returns the tracer
	Tracer() tracing.Tracer

	// Logger returns the structured logger
	Logger() logging.Logger

	// Shutdown gracefully shuts down all observability components
	Shutdown(ctx context.Context) error
}

// observabilityImpl is the default implementation
type observabilityImpl struct {
	config        Config
	metrics       metrics.Collector
	tracer        tracing.Tracer
	logger        logging.Logger
	server        *http.Server
	shutdownFuncs []func(context.Context) error // Collected shutdown functions
}

// New creates a new Observability instance
func New(config Config) (Observability, error) {
	// Apply defaults
	config = config.WithDefaults()

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	obs := &observabilityImpl{
		config:        config,
		shutdownFuncs: make([]func(context.Context) error, 0),
	}

	// Initialize logging FIRST (so we can log errors from other components)
	if config.UseOTelLogs {
		// Use OpenTelemetry Logs SDK
		logger, err := logging.NewOTelLogger(logging.OTelConfig{
			ServiceName:    config.ServiceName,
			ServiceVersion: config.ServiceVersion,
			Environment:    config.Environment,
			OTLPEndpoint:   config.OTLPLogsEndpoint,
			Level:          config.LogLevel,
			Insecure:       config.OTLPInsecure,
		})
		if err != nil {
			// Graceful fallback to structured logger
			fmt.Printf("WARN: Failed to initialize OTel logs: %v, using structured logger\n", err)
			obs.logger = logging.NewStructuredLogger(logging.Config{
				ServiceName: config.ServiceName,
				Level:       config.LogLevel,
				Format:      config.LogFormat,
				Output:      config.LogOutput,
			})
		} else {
			obs.logger = logger
			obs.shutdownFuncs = append(obs.shutdownFuncs, logger.Shutdown)
		}
	} else {
		// Legacy structured logger
		obs.logger = logging.NewStructuredLogger(logging.Config{
			ServiceName: config.ServiceName,
			Level:       config.LogLevel,
			Format:      config.LogFormat,
			Output:      config.LogOutput,
		})
	}

	// Initialize metrics
	if config.EnableMetrics {
		if config.UseOTelMetrics {
			// Use OpenTelemetry Metrics SDK
			collector, err := metrics.NewOTelMetricsCollector(metrics.OTelConfig{
				ServiceName:       config.ServiceName,
				ServiceVersion:    config.ServiceVersion,
				Environment:       config.Environment,
				OTLPEndpoint:      config.OTLPMetricsEndpoint,
				PrometheusEnabled: config.PrometheusEnabled,
				Insecure:          config.OTLPInsecure,
			})
			if err != nil {
				// Graceful fallback to no-op
				obs.logger.Error(context.Background(), "Failed to initialize OTel metrics, using no-op collector", "error", err)
				obs.metrics = metrics.NewNoOpCollector()
			} else {
				obs.metrics = collector
				obs.shutdownFuncs = append(obs.shutdownFuncs, collector.Shutdown)
			}
		} else {
			// Legacy Prometheus implementation
			obs.metrics = metrics.NewPrometheusCollector(metrics.Config{
				ServiceName:    config.ServiceName,
				ServiceVersion: config.ServiceVersion,
				Environment:    config.Environment,
				Namespace:      config.MetricsNamespace,
			})
		}
	} else {
		obs.metrics = metrics.NewNoOpCollector()
	}

	// Initialize tracing
	if config.EnableTracing {
		var err error
		obs.tracer, err = tracing.NewOTelTracer(tracing.Config{
			ServiceName:    config.ServiceName,
			ServiceVersion: config.ServiceVersion,
			Environment:    config.Environment,
			Endpoint:       config.TracingEndpoint,
			SampleRate:     config.TracingSampleRate,
		})
		if err != nil {
			// Graceful fallback to no-op
			obs.logger.Error(context.Background(), "Failed to initialize tracer, using no-op tracer", "error", err)
			obs.tracer = tracing.NewNoOpTracer()
		} else {
			obs.shutdownFuncs = append(obs.shutdownFuncs, obs.tracer.Shutdown)
		}
	} else {
		obs.tracer = tracing.NewNoOpTracer()
	}

	// Start metrics server if enabled
	if config.EnableMetrics && config.MetricsPort > 0 {
		if err := obs.startMetricsServer(); err != nil {
			return nil, fmt.Errorf("failed to start metrics server: %w", err)
		}
	}

	return obs, nil
}

// Metrics returns the metrics collector
func (o *observabilityImpl) Metrics() metrics.Collector {
	return o.metrics
}

// Tracer returns the tracer
func (o *observabilityImpl) Tracer() tracing.Tracer {
	return o.tracer
}

// Logger returns the logger
func (o *observabilityImpl) Logger() logging.Logger {
	return o.logger
}

// Shutdown gracefully shuts down all observability components
func (o *observabilityImpl) Shutdown(ctx context.Context) error {
	var errs []error

	// Shutdown HTTP server
	if o.server != nil {
		if err := o.server.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
			errs = append(errs, fmt.Errorf("metrics server shutdown: %w", err))
			o.logger.Error(ctx, "Metrics server shutdown error", "error", err)
		}
	}

	// Shutdown all components in reverse order
	for i := len(o.shutdownFuncs) - 1; i >= 0; i-- {
		if err := o.shutdownFuncs[i](ctx); err != nil {
			errs = append(errs, err)
			o.logger.Error(ctx, "Component shutdown error", "error", err)
		}
	}

	// Flush logger (if not already shut down)
	if err := o.logger.Sync(); err != nil {
		errs = append(errs, fmt.Errorf("logger sync: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown completed with %d errors: %v", len(errs), errs)
	}

	return nil
}

// startMetricsServer starts the HTTP server for metrics endpoint
func (o *observabilityImpl) startMetricsServer() error {
	mux := http.NewServeMux()

	// Metrics endpoint
	mux.Handle(o.config.MetricsPath, o.metrics.Handler())

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// pprof endpoints (if enabled)
	if o.config.EnablePprof {
		o.registerPprofHandlers(mux)
		o.logger.Info(context.Background(), "pprof endpoints enabled",
			"block_profile_rate", o.config.PprofBlockProfileRate,
			"mutex_profile_fraction", o.config.PprofMutexProfileFraction,
		)
	}

	o.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", o.config.MetricsPort),
		Handler: mux,
	}

	go func() {
		o.logger.Info(context.Background(), "Starting metrics server",
			"port", o.config.MetricsPort,
			"path", o.config.MetricsPath,
		)
		if err := o.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			o.logger.Error(context.Background(), "Metrics server error", "error", err)
		}
	}()

	return nil
}

// registerPprofHandlers registers pprof HTTP handlers
func (o *observabilityImpl) registerPprofHandlers(mux *http.ServeMux) {
	// Set profiling rates
	if o.config.PprofBlockProfileRate > 0 {
		runtime.SetBlockProfileRate(o.config.PprofBlockProfileRate)
	}
	if o.config.PprofMutexProfileFraction > 0 {
		runtime.SetMutexProfileFraction(o.config.PprofMutexProfileFraction)
	}

	// Register pprof handlers
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Additional pprof endpoints
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
}
