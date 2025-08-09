// Package db provides observability and metrics collection for database operations
package db

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// MetricsCollector collects and exports database metrics
type MetricsCollector struct {
	namespace string
	enabled   bool

	// Prometheus metrics
	operationDuration   *prometheus.HistogramVec
	operationCounter    *prometheus.CounterVec
	errorCounter        *prometheus.CounterVec
	connectionPoolGauge *prometheus.GaugeVec
	queryCounter        *prometheus.CounterVec
	cacheHitCounter     *prometheus.CounterVec
	batchSizeHistogram  prometheus.Histogram

	// OpenTelemetry metrics
	otelMeter             metric.Meter
	otelOperationDuration metric.Float64Histogram
	otelOperationCounter  metric.Int64Counter
	otelErrorCounter      metric.Int64Counter
	otelQueryCounter      metric.Int64Counter
	otelCacheHitCounter   metric.Int64Counter

	// OpenTelemetry tracing
	tracer trace.Tracer
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(namespace string, enabled bool) *MetricsCollector {
	if !enabled {
		return &MetricsCollector{enabled: false}
	}

	mc := &MetricsCollector{
		namespace: namespace,
		enabled:   true,
	}

	mc.initPrometheusMetrics()
	mc.initOTelMetrics()

	return mc
}

// initPrometheusMetrics initializes Prometheus metrics
func (mc *MetricsCollector) initPrometheusMetrics() {
	mc.operationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mc.namespace,
			Name:      "operation_duration_seconds",
			Help:      "Duration of database operations in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
		},
		[]string{"operation", "table", "success"},
	)

	mc.operationCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.namespace,
			Name:      "operations_total",
			Help:      "Total number of database operations",
		},
		[]string{"operation", "table", "success"},
	)

	mc.errorCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.namespace,
			Name:      "errors_total",
			Help:      "Total number of database errors",
		},
		[]string{"operation", "error_type", "error_code"},
	)

	mc.connectionPoolGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: mc.namespace,
			Name:      "connection_pool",
			Help:      "Database connection pool statistics",
		},
		[]string{"stat_type"},
	)

	mc.queryCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.namespace,
			Name:      "queries_total",
			Help:      "Total number of database queries",
		},
		[]string{"query_type"},
	)

	mc.cacheHitCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.namespace,
			Name:      "cache_operations_total",
			Help:      "Total number of cache operations",
		},
		[]string{"result"},
	)

	mc.batchSizeHistogram = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: mc.namespace,
			Name:      "batch_size",
			Help:      "Size of batch operations",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
	)
}

// initOTelMetrics initializes OpenTelemetry metrics
func (mc *MetricsCollector) initOTelMetrics() {
	mc.otelMeter = otel.Meter(mc.namespace)
	mc.tracer = otel.Tracer(mc.namespace)

	var err error

	mc.otelOperationDuration, err = mc.otelMeter.Float64Histogram(
		"db_operation_duration",
		metric.WithDescription("Duration of database operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		// Log error but don't fail
	}

	mc.otelOperationCounter, err = mc.otelMeter.Int64Counter(
		"db_operations_total",
		metric.WithDescription("Total number of database operations"),
	)
	if err != nil {
		// Log error but don't fail
	}

	mc.otelErrorCounter, err = mc.otelMeter.Int64Counter(
		"db_errors_total",
		metric.WithDescription("Total number of database errors"),
	)
	if err != nil {
		// Log error but don't fail
	}

	mc.otelQueryCounter, err = mc.otelMeter.Int64Counter(
		"db_queries_total",
		metric.WithDescription("Total number of database queries"),
	)
	if err != nil {
		// Log error but don't fail
	}

	mc.otelCacheHitCounter, err = mc.otelMeter.Int64Counter(
		"db_cache_operations_total",
		metric.WithDescription("Total number of cache operations"),
	)
	if err != nil {
		// Log error but don't fail
	}
}

// RecordOperation records metrics for a database operation
func (mc *MetricsCollector) RecordOperation(operation string, duration time.Duration, success bool) {
	if !mc.enabled {
		return
	}

	table := "unknown"
	successStr := strconv.FormatBool(success)

	// Prometheus metrics
	mc.operationDuration.WithLabelValues(operation, table, successStr).Observe(duration.Seconds())
	mc.operationCounter.WithLabelValues(operation, table, successStr).Inc()

	// OpenTelemetry metrics
	if mc.otelOperationDuration != nil {
		mc.otelOperationDuration.Record(context.Background(), duration.Seconds(),
			metric.WithAttributes(
				attribute.String("operation", operation),
				attribute.String("table", table),
				attribute.Bool("success", success),
			),
		)
	}

	if mc.otelOperationCounter != nil {
		mc.otelOperationCounter.Add(context.Background(), 1,
			metric.WithAttributes(
				attribute.String("operation", operation),
				attribute.String("table", table),
				attribute.Bool("success", success),
			),
		)
	}
}

// RecordOperationWithTable records metrics for a database operation with table information
func (mc *MetricsCollector) RecordOperationWithTable(operation, table string, duration time.Duration, success bool) {
	if !mc.enabled {
		return
	}

	successStr := strconv.FormatBool(success)

	// Prometheus metrics
	mc.operationDuration.WithLabelValues(operation, table, successStr).Observe(duration.Seconds())
	mc.operationCounter.WithLabelValues(operation, table, successStr).Inc()

	// OpenTelemetry metrics
	if mc.otelOperationDuration != nil {
		mc.otelOperationDuration.Record(context.Background(), duration.Seconds(),
			metric.WithAttributes(
				attribute.String("operation", operation),
				attribute.String("table", table),
				attribute.Bool("success", success),
			),
		)
	}

	if mc.otelOperationCounter != nil {
		mc.otelOperationCounter.Add(context.Background(), 1,
			metric.WithAttributes(
				attribute.String("operation", operation),
				attribute.String("table", table),
				attribute.Bool("success", success),
			),
		)
	}
}

// RecordQueryCount records the number of queries executed
func (mc *MetricsCollector) RecordQueryCount(count int64) {
	if !mc.enabled {
		return
	}

	// Prometheus metrics
	mc.queryCounter.WithLabelValues("select").Add(float64(count))

	// OpenTelemetry metrics
	if mc.otelQueryCounter != nil {
		mc.otelQueryCounter.Add(context.Background(), count,
			metric.WithAttributes(
				attribute.String("query_type", "select"),
			),
		)
	}
}

// RecordCacheHit records cache hit/miss statistics
func (mc *MetricsCollector) RecordCacheHit(hit bool) {
	if !mc.enabled {
		return
	}

	result := "miss"
	if hit {
		result = "hit"
	}

	// Prometheus metrics
	mc.cacheHitCounter.WithLabelValues(result).Inc()

	// OpenTelemetry metrics
	if mc.otelCacheHitCounter != nil {
		mc.otelCacheHitCounter.Add(context.Background(), 1,
			metric.WithAttributes(
				attribute.String("result", result),
			),
		)
	}
}

// RecordBatchSize records the size of batch operations
func (mc *MetricsCollector) RecordBatchSize(size int) {
	if !mc.enabled {
		return
	}

	// Prometheus metrics
	mc.batchSizeHistogram.Observe(float64(size))
}

// IncrementError increments error counters
func (mc *MetricsCollector) IncrementError(operation string, errorType string) {
	if !mc.enabled {
		return
	}

	errorCode := "unknown"

	// Prometheus metrics
	mc.errorCounter.WithLabelValues(operation, errorType, errorCode).Inc()

	// OpenTelemetry metrics
	if mc.otelErrorCounter != nil {
		mc.otelErrorCounter.Add(context.Background(), 1,
			metric.WithAttributes(
				attribute.String("operation", operation),
				attribute.String("error_type", errorType),
				attribute.String("error_code", errorCode),
			),
		)
	}
}

// IncrementErrorWithCode increments error counters with error code
func (mc *MetricsCollector) IncrementErrorWithCode(operation, errorType, errorCode string) {
	if !mc.enabled {
		return
	}

	// Prometheus metrics
	mc.errorCounter.WithLabelValues(operation, errorType, errorCode).Inc()

	// OpenTelemetry metrics
	if mc.otelErrorCounter != nil {
		mc.otelErrorCounter.Add(context.Background(), 1,
			metric.WithAttributes(
				attribute.String("operation", operation),
				attribute.String("error_type", errorType),
				attribute.String("error_code", errorCode),
			),
		)
	}
}

// RecordConnectionPoolStats records connection pool statistics
func (mc *MetricsCollector) RecordConnectionPoolStats(stats ConnectionStats) {
	if !mc.enabled {
		return
	}

	// Prometheus metrics
	mc.connectionPoolGauge.WithLabelValues("max_open").Set(float64(stats.MaxOpenConnections))
	mc.connectionPoolGauge.WithLabelValues("open").Set(float64(stats.OpenConnections))
	mc.connectionPoolGauge.WithLabelValues("in_use").Set(float64(stats.InUseConnections))
	mc.connectionPoolGauge.WithLabelValues("idle").Set(float64(stats.IdleConnections))
	mc.connectionPoolGauge.WithLabelValues("wait_count").Set(float64(stats.WaitCount))
	mc.connectionPoolGauge.WithLabelValues("wait_duration_ms").Set(float64(stats.WaitDuration.Milliseconds()))
}

// StartTrace starts a new trace span for database operations
func (mc *MetricsCollector) StartTrace(ctx context.Context, operation string) (context.Context, trace.Span) {
	if !mc.enabled || mc.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	return mc.tracer.Start(ctx, operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "go-ormx"),
			attribute.String("db.operation.name", operation),
		),
	)
}

// StartTraceWithTable starts a new trace span with table information
func (mc *MetricsCollector) StartTraceWithTable(ctx context.Context, operation, table string) (context.Context, trace.Span) {
	if !mc.enabled || mc.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	return mc.tracer.Start(ctx, operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "go-ormx"),
			attribute.String("db.operation.name", operation),
			attribute.String("db.sql.table", table),
		),
	)
}

// RecordTraceError records an error in the current trace span
func (mc *MetricsCollector) RecordTraceError(span trace.Span, err error) {
	if !mc.enabled || span == nil || err == nil {
		return
	}

	span.RecordError(err)
	span.SetAttributes(
		attribute.Bool("error", true),
		attribute.String("error.message", err.Error()),
	)
}

// SetTraceSuccess marks a trace span as successful
func (mc *MetricsCollector) SetTraceSuccess(span trace.Span) {
	if !mc.enabled || span == nil {
		return
	}

	span.SetAttributes(
		attribute.Bool("success", true),
	)
}

// IsEnabled returns whether metrics collection is enabled
func (mc *MetricsCollector) IsEnabled() bool {
	return mc.enabled
}

// Disable disables metrics collection
func (mc *MetricsCollector) Disable() {
	mc.enabled = false
}

// Enable enables metrics collection
func (mc *MetricsCollector) Enable() {
	mc.enabled = true
}

// GetPrometheusRegistry returns the Prometheus registry for custom metrics
func (mc *MetricsCollector) GetPrometheusRegistry() *prometheus.Registry {
	return prometheus.DefaultRegisterer.(*prometheus.Registry)
}

// MetricsMiddleware provides GORM middleware for automatic metrics collection
type MetricsMiddleware struct {
	collector *MetricsCollector
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(collector *MetricsCollector) *MetricsMiddleware {
	return &MetricsMiddleware{
		collector: collector,
	}
}

// Apply applies the metrics middleware to GORM
func (mm *MetricsMiddleware) Apply() func(*Database) {
	return func(db *Database) {
		if mm.collector == nil || !mm.collector.IsEnabled() {
			return
		}

		// Register GORM callbacks for metrics collection
		db.DB().Callback().Create().After("gorm:create").Register("metrics:after_create", mm.afterCreate)
		db.DB().Callback().Query().After("gorm:query").Register("metrics:after_query", mm.afterQuery)
		db.DB().Callback().Update().After("gorm:update").Register("metrics:after_update", mm.afterUpdate)
		db.DB().Callback().Delete().After("gorm:delete").Register("metrics:after_delete", mm.afterDelete)
		db.DB().Callback().Row().After("gorm:row").Register("metrics:after_row", mm.afterRow)
		db.DB().Callback().Raw().After("gorm:raw").Register("metrics:after_raw", mm.afterRaw)
	}
}

func (mm *MetricsMiddleware) afterCreate(db *gorm.DB) {
	mm.recordCallback(db, "create")
}

func (mm *MetricsMiddleware) afterQuery(db *gorm.DB) {
	mm.recordCallback(db, "query")
}

func (mm *MetricsMiddleware) afterUpdate(db *gorm.DB) {
	mm.recordCallback(db, "update")
}

func (mm *MetricsMiddleware) afterDelete(db *gorm.DB) {
	mm.recordCallback(db, "delete")
}

func (mm *MetricsMiddleware) afterRow(db *gorm.DB) {
	mm.recordCallback(db, "row")
}

func (mm *MetricsMiddleware) afterRaw(db *gorm.DB) {
	mm.recordCallback(db, "raw")
}

func (mm *MetricsMiddleware) recordCallback(db *gorm.DB, operation string) {
	if mm.collector == nil || !mm.collector.IsEnabled() {
		return
	}

	// Calculate duration
	start, exists := db.Get("start_time")
	if !exists {
		return
	}

	startTime, ok := start.(time.Time)
	if !ok {
		return
	}

	duration := time.Since(startTime)

	// Get table name
	table := "unknown"
	if db.Statement != nil && db.Statement.Table != "" {
		table = db.Statement.Table
	}

	// Check for errors
	success := db.Error == nil

	// Record metrics
	mm.collector.RecordOperationWithTable(operation, table, duration, success)

	if !success {
		errorType := "database_error"
		if db.Error != nil {
			errorType = "go-ormx_error"
		}
		mm.collector.IncrementError(operation, errorType)
	}
}

// BeforeCallback is a GORM callback that runs before operations to record start time
func BeforeCallback(db *gorm.DB) {
	db.Set("start_time", time.Now())
}
