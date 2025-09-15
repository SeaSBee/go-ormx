package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/SeaSBee/go-ormx/pkg/logging"
)

// ObservabilityConfig represents observability configuration
type ObservabilityConfig struct {
	MetricsEnabled    bool
	TracingEnabled    bool
	MonitoringEnabled bool
	ExportInterval    time.Duration
	MetricsExporters  []MetricsExporter
	TraceExporters    []TraceExporter
}

// DefaultObservabilityConfig returns default observability configuration
func DefaultObservabilityConfig() ObservabilityConfig {
	return ObservabilityConfig{
		MetricsEnabled:    true,
		TracingEnabled:    true,
		MonitoringEnabled: true,
		ExportInterval:    time.Minute * 5,
		MetricsExporters:  []MetricsExporter{},
		TraceExporters:    []TraceExporter{},
	}
}

// ObservabilityManager manages all observability features
type ObservabilityManager struct {
	config  ObservabilityConfig
	metrics *ORMMetrics
	tracer  *ORMTracer
	logger  logging.Logger
	mutex   sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
}

// NewObservabilityManager creates a new observability manager
func NewObservabilityManager(config ObservabilityConfig, logger logging.Logger) *ObservabilityManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ObservabilityManager{
		config:  config,
		metrics: NewORMMetrics(logger),
		tracer:  NewORMTracer(logger, config.TracingEnabled),
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
		started: false,
	}
}

// Start starts the observability manager
func (om *ObservabilityManager) Start(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	om.mutex.Lock()
	defer om.mutex.Unlock()

	if om.started {
		return nil
	}

	om.logger.Info(ctx, "Starting observability manager")

	// Start metrics collection
	if om.config.MetricsEnabled {
		om.logger.Info(ctx, "Metrics collection enabled")
	}

	// Start tracing
	if om.config.TracingEnabled {
		om.logger.Info(ctx, "Tracing enabled")
	}

	// Start monitoring
	if om.config.MonitoringEnabled {
		om.logger.Info(ctx, "Monitoring enabled")
	}

	// Start export goroutine
	if len(om.config.MetricsExporters) > 0 || len(om.config.TraceExporters) > 0 {
		go om.exportRoutine()
	}

	om.started = true
	om.logger.Info(ctx, "Observability manager started successfully")
	return nil
}

// Stop stops the observability manager
func (om *ObservabilityManager) Stop(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	om.mutex.Lock()
	defer om.mutex.Unlock()

	if !om.started {
		return nil
	}

	om.logger.Info(ctx, "Stopping observability manager")

	// Cancel context to stop goroutines
	om.cancel()

	// Export final metrics and traces
	if err := om.exportAll(ctx); err != nil {
		om.logger.Error(ctx, "Failed to export final observability data", logging.ErrorField("error", err))
	}

	om.started = false
	om.logger.Info(ctx, "Observability manager stopped successfully")
	return nil
}

// GetMetrics returns the metrics collector
func (om *ObservabilityManager) GetMetrics() *ORMMetrics {
	return om.metrics
}

// GetTracer returns the tracer
func (om *ObservabilityManager) GetTracer() *ORMTracer {
	return om.tracer
}

// RecordQueryMetrics records query metrics with tracing
func (om *ObservabilityManager) RecordQueryMetrics(ctx context.Context, query string, duration time.Duration, rowsAffected int64, success bool) {
	// Record metrics
	if om.config.MetricsEnabled {
		om.metrics.RecordQueryMetrics(ctx, query, duration, rowsAffected, success)
	}

	// Record tracing
	if om.config.TracingEnabled {
		spanCtx, span := om.tracer.StartQuerySpan(ctx, query, "query")
		if span != nil {
			om.tracer.AddQuerySpanEvent(span, "query_executed", rowsAffected, duration)
			if !success {
				om.tracer.AddErrorSpanEvent(span, fmt.Errorf("query failed"))
			}
			om.tracer.EndSpan(span, nil)
		}
		ctx = spanCtx
	}
}

// RecordTransactionMetrics records transaction metrics with tracing
func (om *ObservabilityManager) RecordTransactionMetrics(ctx context.Context, operation string, duration time.Duration, success bool) {
	// Record metrics
	if om.config.MetricsEnabled {
		om.metrics.RecordTransactionMetrics(ctx, operation, duration, success)
	}

	// Record tracing
	if om.config.TracingEnabled {
		spanCtx, span := om.tracer.StartTransactionSpan(ctx, operation)
		if span != nil {
			om.tracer.AddQuerySpanEvent(span, "transaction_executed", 0, duration)
			if !success {
				om.tracer.AddErrorSpanEvent(span, fmt.Errorf("transaction failed"))
			}
			om.tracer.EndSpan(span, nil)
		}
		ctx = spanCtx
	}
}

// RecordModelMetrics records model operation metrics with tracing
func (om *ObservabilityManager) RecordModelMetrics(ctx context.Context, model string, operation string, duration time.Duration, success bool) {
	// Record metrics
	if om.config.MetricsEnabled {
		om.metrics.RecordModelMetrics(ctx, model, operation, duration, success)
	}

	// Record tracing
	if om.config.TracingEnabled {
		spanCtx, span := om.tracer.StartModelSpan(ctx, model, operation)
		if span != nil {
			om.tracer.AddQuerySpanEvent(span, "model_operation_executed", 0, duration)
			if !success {
				om.tracer.AddErrorSpanEvent(span, fmt.Errorf("model operation failed"))
			}
			om.tracer.EndSpan(span, nil)
		}
		ctx = spanCtx
	}
}

// RecordValidationMetrics records validation metrics with tracing
func (om *ObservabilityManager) RecordValidationMetrics(ctx context.Context, model string, valid, invalid int) {
	// Record metrics
	if om.config.MetricsEnabled {
		om.metrics.RecordValidationMetrics(ctx, model, valid, invalid)
	}

	// Record tracing
	if om.config.TracingEnabled {
		spanCtx, span := om.tracer.StartValidationSpan(ctx, model)
		if span != nil {
			om.tracer.AddSpanAttribute(span, "valid_count", fmt.Sprintf("%d", valid))
			om.tracer.AddSpanAttribute(span, "invalid_count", fmt.Sprintf("%d", invalid))
			om.tracer.EndSpan(span, nil)
		}
		ctx = spanCtx
	}
}

// RecordCacheMetrics records cache metrics with tracing
func (om *ObservabilityManager) RecordCacheMetrics(ctx context.Context, hits, misses, evictions int64, size, maxSize int64) {
	// Record metrics
	if om.config.MetricsEnabled {
		om.metrics.RecordCacheMetrics(ctx, hits, misses, evictions, size, maxSize)
	}
}

// RecordConnectionMetrics records connection metrics
func (om *ObservabilityManager) RecordConnectionMetrics(ctx context.Context, activeConnections, idleConnections, maxConnections int) {
	// Record metrics
	if om.config.MetricsEnabled {
		om.metrics.RecordConnectionMetrics(ctx, activeConnections, idleConnections, maxConnections)
	}
}

// RecordErrorMetrics records error metrics with tracing
func (om *ObservabilityManager) RecordErrorMetrics(ctx context.Context, errorType string, errorCode string, err error) {
	// Record metrics
	if om.config.MetricsEnabled {
		om.metrics.RecordErrorMetrics(ctx, errorType, errorCode)
	}

	// Record tracing
	if om.config.TracingEnabled && err != nil {
		spanCtx, span := om.tracer.StartSpan(ctx, "orm.error", SpanKindInternal)
		if span != nil {
			om.tracer.AddSpanAttribute(span, "error_type", errorType)
			om.tracer.AddSpanAttribute(span, "error_code", errorCode)
			om.tracer.AddErrorSpanEvent(span, err)
			om.tracer.EndSpan(span, err)
		}
		ctx = spanCtx
	}
}

// RecordSystemMetrics records system-level metrics
func (om *ObservabilityManager) RecordSystemMetrics(ctx context.Context, memoryUsage, cpuUsage float64, goroutineCount int) {
	// Record metrics
	if om.config.MetricsEnabled {
		om.metrics.RecordSystemMetrics(ctx, memoryUsage, cpuUsage, goroutineCount)
	}
}

// StartQuerySpan starts a query span with metrics tracking
func (om *ObservabilityManager) StartQuerySpan(ctx context.Context, query string, operation string) (context.Context, *Span) {
	if om.config.TracingEnabled {
		return om.tracer.StartQuerySpan(ctx, query, operation)
	}
	return ctx, nil
}

// StartTransactionSpan starts a transaction span with metrics tracking
func (om *ObservabilityManager) StartTransactionSpan(ctx context.Context, operation string) (context.Context, *Span) {
	if om.config.TracingEnabled {
		return om.tracer.StartTransactionSpan(ctx, operation)
	}
	return ctx, nil
}

// StartModelSpan starts a model operation span with metrics tracking
func (om *ObservabilityManager) StartModelSpan(ctx context.Context, model string, operation string) (context.Context, *Span) {
	if om.config.TracingEnabled {
		return om.tracer.StartModelSpan(ctx, model, operation)
	}
	return ctx, nil
}

// StartValidationSpan starts a validation span with metrics tracking
func (om *ObservabilityManager) StartValidationSpan(ctx context.Context, model string) (context.Context, *Span) {
	if om.config.TracingEnabled {
		return om.tracer.StartValidationSpan(ctx, model)
	}
	return ctx, nil
}

// StartCacheSpan starts a cache operation span
func (om *ObservabilityManager) StartCacheSpan(ctx context.Context, operation string, key string) (context.Context, *Span) {
	if om.config.TracingEnabled {
		return om.tracer.StartCacheSpan(ctx, operation, key)
	}
	return ctx, nil
}

// EndSpan ends a span
func (om *ObservabilityManager) EndSpan(span *Span, err error) {
	if om.config.TracingEnabled {
		om.tracer.EndSpan(span, err)
	}
}

// AddSpanEvent adds an event to a span
func (om *ObservabilityManager) AddSpanEvent(span *Span, name string, attributes map[string]string) {
	if om.config.TracingEnabled {
		om.tracer.AddSpanEvent(span, name, attributes)
	}
}

// AddSpanAttribute adds an attribute to a span
func (om *ObservabilityManager) AddSpanAttribute(span *Span, key, value string) {
	if om.config.TracingEnabled {
		om.tracer.AddSpanAttribute(span, key, value)
	}
}

// GetMetricsSummary returns a summary of all metrics
func (om *ObservabilityManager) GetMetricsSummary(ctx context.Context) map[string]interface{} {
	if ctx == nil {
		panic("context cannot be nil")
	}

	if !om.config.MetricsEnabled {
		return nil
	}
	return om.metrics.GetMetricsSummary(ctx)
}

// GetTraceID gets the trace ID from context
func (om *ObservabilityManager) GetTraceID(ctx context.Context) TraceID {
	if om.config.TracingEnabled {
		return om.tracer.GetTraceID(ctx)
	}
	return ""
}

// GetSpanID gets the span ID from context
func (om *ObservabilityManager) GetSpanID(ctx context.Context) SpanID {
	if om.config.TracingEnabled {
		return om.tracer.GetSpanID(ctx)
	}
	return ""
}

// InjectTraceContext injects trace context into a carrier
func (om *ObservabilityManager) InjectTraceContext(ctx context.Context, carrier map[string]string) {
	if om.config.TracingEnabled {
		om.tracer.InjectTraceContext(ctx, carrier)
	}
}

// ExtractTraceContext extracts trace context from a carrier
func (om *ObservabilityManager) ExtractTraceContext(carrier map[string]string) context.Context {
	if om.config.TracingEnabled {
		return om.tracer.ExtractTraceContext(carrier)
	}
	return context.Background()
}

// AddMetricsExporter adds a metrics exporter
func (om *ObservabilityManager) AddMetricsExporter(exporter MetricsExporter) {
	om.mutex.Lock()
	defer om.mutex.Unlock()

	om.config.MetricsExporters = append(om.config.MetricsExporters, exporter)
}

// AddTraceExporter adds a trace exporter
func (om *ObservabilityManager) AddTraceExporter(exporter TraceExporter) {
	om.mutex.Lock()
	defer om.mutex.Unlock()

	om.config.TraceExporters = append(om.config.TraceExporters, exporter)
}

// exportRoutine runs the export routine
func (om *ObservabilityManager) exportRoutine() {
	ticker := time.NewTicker(om.config.ExportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-om.ctx.Done():
			return
		case <-ticker.C:
			if err := om.exportAll(om.ctx); err != nil {
				om.logger.Error(om.ctx, "Failed to export observability data", logging.ErrorField("error", err))
			}
		}
	}
}

// exportAll exports all observability data
func (om *ObservabilityManager) exportAll(ctx context.Context) error {
	// Export metrics
	if om.config.MetricsEnabled && len(om.config.MetricsExporters) > 0 {
		metrics, err := om.metrics.Collect(ctx)
		if err != nil {
			return fmt.Errorf("failed to collect metrics: %w", err)
		}

		for _, exporter := range om.config.MetricsExporters {
			if err := exporter.Export(ctx, metrics); err != nil {
				om.logger.Error(ctx, "Failed to export metrics", logging.ErrorField("error", err))
			}
		}

		// Export metrics summary
		summary := om.metrics.GetMetricsSummary(ctx)
		for _, exporter := range om.config.MetricsExporters {
			if err := exporter.ExportSummary(ctx, summary); err != nil {
				om.logger.Error(ctx, "Failed to export metrics summary", logging.ErrorField("error", err))
			}
		}
	}

	// Export traces
	if om.config.TracingEnabled && len(om.config.TraceExporters) > 0 {
		// Note: In a real implementation, you would collect spans from the tracer
		// For now, we'll just log that tracing is enabled
		om.logger.Debug(ctx, "Tracing enabled, spans would be exported")
	}

	return nil
}

// IsStarted returns whether the observability manager is started
func (om *ObservabilityManager) IsStarted() bool {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	return om.started
}

// GetConfig returns the observability configuration
func (om *ObservabilityManager) GetConfig() ObservabilityConfig {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	return om.config
}

// UpdateConfig updates the observability configuration
func (om *ObservabilityManager) UpdateConfig(config ObservabilityConfig) {
	om.mutex.Lock()
	defer om.mutex.Unlock()

	om.config = config
	om.tracer = NewORMTracer(om.logger, config.TracingEnabled)
}
