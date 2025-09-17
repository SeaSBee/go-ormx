package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/seasbee/go-ormx/pkg/errors"
	"github.com/seasbee/go-ormx/pkg/logging"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric represents a single metric
type Metric struct {
	Name        string            `json:"name"`
	Type        MetricType        `json:"type"`
	Value       float64           `json:"value"`
	Labels      map[string]string `json:"labels"`
	Timestamp   time.Time         `json:"timestamp"`
	Description string            `json:"description"`
	Unit        string            `json:"unit"`
}

// MetricCollector interface for collecting metrics
type MetricCollector interface {
	Collect(ctx context.Context) ([]Metric, error)
	GetMetric(name string) (*Metric, error)
	GetMetricsByType(metricType MetricType) ([]Metric, error)
	GetMetricsByLabels(labels map[string]string) ([]Metric, error)
	Reset()
}

// BaseMetricCollector implements basic metric collection functionality
type BaseMetricCollector struct {
	metrics map[string]*Metric
	mutex   sync.RWMutex
	logger  logging.Logger
}

// NewBaseMetricCollector creates a new base metric collector
func NewBaseMetricCollector(logger logging.Logger) *BaseMetricCollector {
	return &BaseMetricCollector{
		metrics: make(map[string]*Metric),
		logger:  logger,
	}
}

// Collect returns all collected metrics
func (bmc *BaseMetricCollector) Collect(ctx context.Context) ([]Metric, error) {
	bmc.mutex.RLock()
	defer bmc.mutex.RUnlock()

	metrics := make([]Metric, 0, len(bmc.metrics))
	for _, metric := range bmc.metrics {
		metrics = append(metrics, *metric)
	}

	return metrics, nil
}

// GetMetric returns a specific metric by name
func (bmc *BaseMetricCollector) GetMetric(name string) (*Metric, error) {
	bmc.mutex.RLock()
	defer bmc.mutex.RUnlock()

	metric, exists := bmc.metrics[name]
	if !exists {
		return nil, errors.New(errors.ErrorTypeNotFound, fmt.Sprintf("Metric %s not found", name))
	}

	return metric, nil
}

// GetMetricsByType returns metrics filtered by type
func (bmc *BaseMetricCollector) GetMetricsByType(metricType MetricType) ([]Metric, error) {
	bmc.mutex.RLock()
	defer bmc.mutex.RUnlock()

	var metrics []Metric
	for _, metric := range bmc.metrics {
		if metric.Type == metricType {
			metrics = append(metrics, *metric)
		}
	}

	return metrics, nil
}

// GetMetricsByLabels returns metrics filtered by labels
func (bmc *BaseMetricCollector) GetMetricsByLabels(labels map[string]string) ([]Metric, error) {
	bmc.mutex.RLock()
	defer bmc.mutex.RUnlock()

	var metrics []Metric
	for _, metric := range bmc.metrics {
		if bmc.matchesLabels(metric.Labels, labels) {
			metrics = append(metrics, *metric)
		}
	}

	return metrics, nil
}

// Reset clears all metrics
func (bmc *BaseMetricCollector) Reset() {
	bmc.mutex.Lock()
	defer bmc.mutex.Unlock()

	bmc.metrics = make(map[string]*Metric)
}

// matchesLabels checks if metric labels match the provided labels
func (bmc *BaseMetricCollector) matchesLabels(metricLabels, queryLabels map[string]string) bool {
	for key, value := range queryLabels {
		if metricValue, exists := metricLabels[key]; !exists || metricValue != value {
			return false
		}
	}
	return true
}

// setMetric sets a metric value
func (bmc *BaseMetricCollector) setMetric(name string, metricType MetricType, value float64, labels map[string]string, description, unit string) {
	bmc.mutex.Lock()
	defer bmc.mutex.Unlock()

	bmc.metrics[name] = &Metric{
		Name:        name,
		Type:        metricType,
		Value:       value,
		Labels:      labels,
		Timestamp:   time.Now(),
		Description: description,
		Unit:        unit,
	}
}

// incrementMetric increments a counter metric
func (bmc *BaseMetricCollector) incrementMetric(name string, labels map[string]string) {
	bmc.mutex.Lock()
	defer bmc.mutex.Unlock()

	if metric, exists := bmc.metrics[name]; exists && metric.Type == MetricTypeCounter {
		metric.Value++
		metric.Timestamp = time.Now()
	} else {
		bmc.metrics[name] = &Metric{
			Name:      name,
			Type:      MetricTypeCounter,
			Value:     1,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

// ORMMetrics represents ORM-specific metrics
type ORMMetrics struct {
	*BaseMetricCollector
}

// NewORMMetrics creates new ORM metrics collector
func NewORMMetrics(logger logging.Logger) *ORMMetrics {
	return &ORMMetrics{
		BaseMetricCollector: NewBaseMetricCollector(logger),
	}
}

// RecordQueryMetrics records query performance metrics
func (om *ORMMetrics) RecordQueryMetrics(ctx context.Context, query string, duration time.Duration, rowsAffected int64, success bool) {
	labels := map[string]string{
		"query":   query,
		"success": fmt.Sprintf("%t", success),
	}

	// Query duration histogram
	om.setMetric("orm_query_duration_seconds", MetricTypeHistogram, duration.Seconds(), labels, "Query execution duration", "seconds")

	// Query count
	om.incrementMetric("orm_query_total", labels)

	// Rows affected
	om.setMetric("orm_rows_affected", MetricTypeGauge, float64(rowsAffected), labels, "Number of rows affected by query", "rows")

	// Success/failure rate
	if success {
		om.incrementMetric("orm_query_success_total", labels)
	} else {
		om.incrementMetric("orm_query_error_total", labels)
	}

	om.logger.Debug(ctx, "Query metrics recorded",
		logging.String("query", query),
		logging.Duration("duration", duration),
		logging.Int64("rows_affected", rowsAffected),
		logging.Bool("success", success))
}

// RecordConnectionMetrics records connection pool metrics
func (om *ORMMetrics) RecordConnectionMetrics(ctx context.Context, activeConnections, idleConnections, maxConnections int) {
	labels := map[string]string{
		"pool": "database",
	}

	om.setMetric("orm_connections_active", MetricTypeGauge, float64(activeConnections), labels, "Number of active connections", "connections")
	om.setMetric("orm_connections_idle", MetricTypeGauge, float64(idleConnections), labels, "Number of idle connections", "connections")
	om.setMetric("orm_connections_max", MetricTypeGauge, float64(maxConnections), labels, "Maximum number of connections", "connections")

	// Connection utilization percentage
	utilization := 0.0
	if maxConnections > 0 {
		utilization = float64(activeConnections) / float64(maxConnections) * 100
	}
	om.setMetric("orm_connections_utilization_percent", MetricTypeGauge, utilization, labels, "Connection pool utilization percentage", "percent")

	om.logger.Debug(ctx, "Connection metrics recorded",
		logging.Int("active", activeConnections),
		logging.Int("idle", idleConnections),
		logging.Int("max", maxConnections),
		logging.Float64("utilization_percent", utilization))
}

// RecordCacheMetrics records cache performance metrics
func (om *ORMMetrics) RecordCacheMetrics(ctx context.Context, hits, misses, evictions int64, size, maxSize int64) {
	labels := map[string]string{
		"cache": "orm",
	}

	om.setMetric("orm_cache_hits_total", MetricTypeCounter, float64(hits), labels, "Total cache hits", "hits")
	om.setMetric("orm_cache_misses_total", MetricTypeCounter, float64(misses), labels, "Total cache misses", "misses")
	om.setMetric("orm_cache_evictions_total", MetricTypeCounter, float64(evictions), labels, "Total cache evictions", "evictions")
	om.setMetric("orm_cache_size", MetricTypeGauge, float64(size), labels, "Current cache size", "items")
	om.setMetric("orm_cache_max_size", MetricTypeGauge, float64(maxSize), labels, "Maximum cache size", "items")

	// Hit rate calculation
	hitRate := 0.0
	if hits+misses > 0 {
		hitRate = float64(hits) / float64(hits+misses) * 100
	}
	om.setMetric("orm_cache_hit_rate_percent", MetricTypeGauge, hitRate, labels, "Cache hit rate percentage", "percent")

	om.logger.Debug(ctx, "Cache metrics recorded",
		logging.Int64("hits", hits),
		logging.Int64("misses", misses),
		logging.Int64("evictions", evictions),
		logging.Int64("size", size),
		logging.Int64("max_size", maxSize),
		logging.Float64("hit_rate_percent", hitRate))
}

// RecordTransactionMetrics records transaction metrics
func (om *ORMMetrics) RecordTransactionMetrics(ctx context.Context, operation string, duration time.Duration, success bool) {
	labels := map[string]string{
		"operation": operation,
		"success":   fmt.Sprintf("%t", success),
	}

	om.setMetric("orm_transaction_duration_seconds", MetricTypeHistogram, duration.Seconds(), labels, "Transaction duration", "seconds")
	om.incrementMetric("orm_transaction_total", labels)

	if success {
		om.incrementMetric("orm_transaction_success_total", labels)
	} else {
		om.incrementMetric("orm_transaction_error_total", labels)
	}

	om.logger.Debug(ctx, "Transaction metrics recorded",
		logging.String("operation", operation),
		logging.Duration("duration", duration),
		logging.Bool("success", success))
}

// RecordModelMetrics records model operation metrics
func (om *ORMMetrics) RecordModelMetrics(ctx context.Context, model string, operation string, duration time.Duration, success bool) {
	labels := map[string]string{
		"model":     model,
		"operation": operation,
		"success":   fmt.Sprintf("%t", success),
	}

	om.setMetric("orm_model_operation_duration_seconds", MetricTypeHistogram, duration.Seconds(), labels, "Model operation duration", "seconds")
	om.incrementMetric("orm_model_operation_total", labels)

	if success {
		om.incrementMetric("orm_model_operation_success_total", labels)
	} else {
		om.incrementMetric("orm_model_operation_error_total", labels)
	}

	om.logger.Debug(ctx, "Model metrics recorded",
		logging.String("model", model),
		logging.String("operation", operation),
		logging.Duration("duration", duration),
		logging.Bool("success", success))
}

// RecordValidationMetrics records validation metrics
func (om *ORMMetrics) RecordValidationMetrics(ctx context.Context, model string, valid, invalid int) {
	labels := map[string]string{
		"model": model,
	}

	om.setMetric("orm_validation_valid_total", MetricTypeCounter, float64(valid), labels, "Total valid validations", "validations")
	om.setMetric("orm_validation_invalid_total", MetricTypeCounter, float64(invalid), labels, "Total invalid validations", "validations")

	total := valid + invalid
	successRate := 0.0
	if total > 0 {
		successRate = float64(valid) / float64(total) * 100
	}
	om.setMetric("orm_validation_success_rate_percent", MetricTypeGauge, successRate, labels, "Validation success rate", "percent")

	om.logger.Debug(ctx, "Validation metrics recorded",
		logging.String("model", model),
		logging.Int("valid", valid),
		logging.Int("invalid", invalid),
		logging.Float64("success_rate_percent", successRate))
}

// RecordErrorMetrics records error metrics
func (om *ORMMetrics) RecordErrorMetrics(ctx context.Context, errorType string, errorCode string) {
	labels := map[string]string{
		"error_type": errorType,
		"error_code": errorCode,
	}

	om.incrementMetric("orm_errors_total", labels)

	om.logger.Debug(ctx, "Error metrics recorded",
		logging.String("error_type", errorType),
		logging.String("error_code", errorCode))
}

// RecordSystemMetrics records system-level metrics
func (om *ORMMetrics) RecordSystemMetrics(ctx context.Context, memoryUsage, cpuUsage float64, goroutineCount int) {
	labels := map[string]string{
		"component": "orm",
	}

	om.setMetric("orm_memory_usage_mb", MetricTypeGauge, memoryUsage, labels, "Memory usage in MB", "MB")
	om.setMetric("orm_cpu_usage_percent", MetricTypeGauge, cpuUsage, labels, "CPU usage percentage", "percent")
	om.setMetric("orm_goroutines", MetricTypeGauge, float64(goroutineCount), labels, "Number of goroutines", "goroutines")

	om.logger.Debug(ctx, "System metrics recorded",
		logging.Float64("memory_usage_mb", memoryUsage),
		logging.Float64("cpu_usage_percent", cpuUsage),
		logging.Int("goroutines", goroutineCount))
}

// GetMetricsSummary returns a summary of all metrics
func (om *ORMMetrics) GetMetricsSummary(ctx context.Context) map[string]interface{} {
	metrics, err := om.Collect(ctx)
	if err != nil {
		om.logger.Error(ctx, "Failed to collect metrics", logging.ErrorField("error", err))
		return nil
	}

	summary := make(map[string]interface{})
	summary["total_metrics"] = len(metrics)
	summary["timestamp"] = time.Now()

	// Group metrics by type
	byType := make(map[MetricType]int)
	for _, metric := range metrics {
		byType[metric.Type]++
	}
	summary["by_type"] = byType

	// Get latest values for key metrics
	keyMetrics := make(map[string]interface{})
	for _, metric := range metrics {
		if om.isKeyMetric(metric.Name) {
			keyMetrics[metric.Name] = map[string]interface{}{
				"value":     metric.Value,
				"type":      metric.Type,
				"labels":    metric.Labels,
				"timestamp": metric.Timestamp,
			}
		}
	}
	summary["key_metrics"] = keyMetrics

	return summary
}

// isKeyMetric checks if a metric is considered a key metric
func (om *ORMMetrics) isKeyMetric(name string) bool {
	keyMetrics := []string{
		"orm_query_total",
		"orm_query_success_total",
		"orm_query_error_total",
		"orm_cache_hit_rate_percent",
		"orm_connections_utilization_percent",
		"orm_transaction_success_total",
		"orm_errors_total",
	}

	for _, keyMetric := range keyMetrics {
		if name == keyMetric {
			return true
		}
	}
	return false
}

// MetricsExporter interface for exporting metrics
type MetricsExporter interface {
	Export(ctx context.Context, metrics []Metric) error
	ExportSummary(ctx context.Context, summary map[string]interface{}) error
}

// PrometheusExporter exports metrics in Prometheus format
type PrometheusExporter struct {
	logger logging.Logger
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter(logger logging.Logger) *PrometheusExporter {
	return &PrometheusExporter{
		logger: logger,
	}
}

// Export exports metrics in Prometheus format
func (pe *PrometheusExporter) Export(ctx context.Context, metrics []Metric) error {
	// This is a simplified implementation
	// In a real implementation, you would format metrics according to Prometheus specification
	for _, metric := range metrics {
		pe.logger.Info(ctx, "Prometheus metric",
			logging.String("name", metric.Name),
			logging.String("type", string(metric.Type)),
			logging.Float64("value", metric.Value),
			logging.Any("labels", metric.Labels))
	}

	return nil
}

// ExportSummary exports metrics summary
func (pe *PrometheusExporter) ExportSummary(ctx context.Context, summary map[string]interface{}) error {
	pe.logger.Info(ctx, "Prometheus metrics summary", logging.Any("summary", summary))
	return nil
}

// JSONExporter exports metrics in JSON format
type JSONExporter struct {
	logger logging.Logger
}

// NewJSONExporter creates a new JSON exporter
func NewJSONExporter(logger logging.Logger) *JSONExporter {
	return &JSONExporter{
		logger: logger,
	}
}

// Export exports metrics in JSON format
func (je *JSONExporter) Export(ctx context.Context, metrics []Metric) error {
	je.logger.Info(ctx, "JSON metrics export", logging.Any("metrics", metrics))
	return nil
}

// ExportSummary exports metrics summary in JSON format
func (je *JSONExporter) ExportSummary(ctx context.Context, summary map[string]interface{}) error {
	je.logger.Info(ctx, "JSON metrics summary", logging.Any("summary", summary))
	return nil
}
