package unit

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/seasbee/go-ormx/pkg/logging"
	"github.com/seasbee/go-ormx/pkg/observability"
	"github.com/stretchr/testify/assert"
)

func TestDefaultObservabilityConfig(t *testing.T) {
	config := observability.DefaultObservabilityConfig()

	assert.True(t, config.MetricsEnabled)
	assert.True(t, config.TracingEnabled)
	assert.True(t, config.MonitoringEnabled)
	assert.Equal(t, 5*time.Minute, config.ExportInterval)
	assert.Empty(t, config.MetricsExporters)
	assert.Empty(t, config.TraceExporters)
}

func TestNewObservabilityManager(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})

	manager := observability.NewObservabilityManager(config, logger)

	assert.NotNil(t, manager)
	assert.Equal(t, config, manager.GetConfig())
	assert.False(t, manager.IsStarted())
}

func TestObservabilityManager_Start(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	// Start the manager
	err := manager.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, manager.IsStarted())

	// Starting again should not error
	err = manager.Start(ctx)
	assert.NoError(t, err)
}

func TestObservabilityManager_Stop(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	// Start the manager
	err := manager.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, manager.IsStarted())

	// Stop the manager
	err = manager.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, manager.IsStarted())

	// Stopping again should not error
	err = manager.Stop(ctx)
	assert.NoError(t, err)
}

func TestObservabilityManager_GetMetrics(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	metrics := manager.GetMetrics()
	assert.NotNil(t, metrics)
}

func TestObservabilityManager_GetTracer(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	tracer := manager.GetTracer()
	assert.NotNil(t, tracer)
}

func TestObservabilityManager_RecordQueryMetrics(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	// Test successful query
	manager.RecordQueryMetrics(ctx, "SELECT * FROM users", 100*time.Millisecond, 10, true)

	// Test failed query
	manager.RecordQueryMetrics(ctx, "SELECT * FROM users", 50*time.Millisecond, 0, false)
}

func TestObservabilityManager_RecordTransactionMetrics(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	// Test successful transaction
	manager.RecordTransactionMetrics(ctx, "user_registration", 200*time.Millisecond, true)

	// Test failed transaction
	manager.RecordTransactionMetrics(ctx, "user_registration", 100*time.Millisecond, false)
}

func TestObservabilityManager_RecordModelMetrics(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	// Test successful model operation
	manager.RecordModelMetrics(ctx, "User", "create", 150*time.Millisecond, true)

	// Test failed model operation
	manager.RecordModelMetrics(ctx, "User", "update", 75*time.Millisecond, false)
}

func TestObservabilityManager_RecordValidationMetrics(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	manager.RecordValidationMetrics(ctx, "User", 95, 5)
}

func TestObservabilityManager_RecordCacheMetrics(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	manager.RecordCacheMetrics(ctx, 80, 20, 5, 100, 1000)
}

func TestObservabilityManager_RecordConnectionMetrics(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	manager.RecordConnectionMetrics(ctx, 25, 15, 100)
}

func TestObservabilityManager_RecordErrorMetrics(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	manager.RecordErrorMetrics(ctx, "validation", "INVALID_EMAIL", assert.AnError)
}

func TestObservabilityManager_RecordSystemMetrics(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	manager.RecordSystemMetrics(ctx, 45.5, 23.1, 150)
}

func TestObservabilityManager_StartQuerySpan(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	spanCtx, span := manager.StartQuerySpan(ctx, "SELECT * FROM users", "query")
	assert.NotNil(t, spanCtx)

	if span != nil {
		manager.EndSpan(span, nil)
	}
}

func TestObservabilityManager_StartTransactionSpan(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	spanCtx, span := manager.StartTransactionSpan(ctx, "user_registration")
	assert.NotNil(t, spanCtx)

	if span != nil {
		manager.EndSpan(span, nil)
	}
}

func TestObservabilityManager_StartModelSpan(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	spanCtx, span := manager.StartModelSpan(ctx, "User", "create")
	assert.NotNil(t, spanCtx)

	if span != nil {
		manager.EndSpan(span, nil)
	}
}

func TestObservabilityManager_StartValidationSpan(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	spanCtx, span := manager.StartValidationSpan(ctx, "User")
	assert.NotNil(t, spanCtx)

	if span != nil {
		manager.EndSpan(span, nil)
	}
}

func TestObservabilityManager_StartCacheSpan(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	spanCtx, span := manager.StartCacheSpan(ctx, "get", "user:123")
	assert.NotNil(t, spanCtx)

	if span != nil {
		manager.EndSpan(span, nil)
	}
}

func TestObservabilityManager_EndSpan(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	_, span := manager.StartQuerySpan(ctx, "SELECT * FROM users", "query")
	if span != nil {
		manager.EndSpan(span, nil)
		// Should not panic
	}
}

func TestObservabilityManager_AddSpanEvent(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	_, span := manager.StartQuerySpan(ctx, "SELECT * FROM users", "query")
	if span != nil {
		manager.AddSpanEvent(span, "query_executed", map[string]string{"rows": "10"})
		manager.EndSpan(span, nil)
	}
}

func TestObservabilityManager_AddSpanAttribute(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	_, span := manager.StartQuerySpan(ctx, "SELECT * FROM users", "query")
	if span != nil {
		manager.AddSpanAttribute(span, "table", "users")
		manager.EndSpan(span, nil)
	}
}

func TestObservabilityManager_GetMetricsSummary(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	summary := manager.GetMetricsSummary(ctx)
	assert.NotNil(t, summary)
}

func TestObservabilityManager_GetTraceID(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Start the manager first
	ctx := context.Background()
	err := manager.Start(ctx)
	assert.NoError(t, err)
	defer manager.Stop(ctx)

	// Start a span to generate a trace ID
	spanCtx, span := manager.StartQuerySpan(ctx, "test_query", "test_operation")
	assert.NotNil(t, span)
	assert.NotNil(t, spanCtx)

	// Get trace ID from the span context, not the original context
	traceID := manager.GetTraceID(spanCtx)
	assert.NotEmpty(t, traceID)
}

func TestObservabilityManager_GetSpanID(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Start the manager first
	ctx := context.Background()
	err := manager.Start(ctx)
	assert.NoError(t, err)
	defer manager.Stop(ctx)

	// Start a span to generate a span ID
	spanCtx, span := manager.StartQuerySpan(ctx, "test_query", "test_operation")
	assert.NotNil(t, span)
	assert.NotNil(t, spanCtx)

	// Get span ID from the span context, not the original context
	spanID := manager.GetSpanID(spanCtx)
	assert.NotEmpty(t, spanID)
}

func TestObservabilityManager_InjectTraceContext(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Start the manager first
	ctx := context.Background()
	err := manager.Start(ctx)
	assert.NoError(t, err)
	defer manager.Stop(ctx)

	// Start a span to generate trace context
	spanCtx, span := manager.StartQuerySpan(ctx, "test_query", "test_operation")
	assert.NotNil(t, span)
	assert.NotNil(t, spanCtx)

	carrier := make(map[string]string)
	manager.InjectTraceContext(spanCtx, carrier)

	// Carrier should contain trace context
	assert.NotEmpty(t, carrier)
}

func TestObservabilityManager_ExtractTraceContext(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	carrier := map[string]string{
		"trace_id": "trace-123",
		"span_id":  "span-456",
	}

	ctx := manager.ExtractTraceContext(carrier)
	assert.NotNil(t, ctx)
}

func TestObservabilityManager_AddMetricsExporter(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Mock exporter
	exporter := &mockMetricsExporter{}

	manager.AddMetricsExporter(exporter)

	// Check that exporter was added
	managerConfig := manager.GetConfig()
	assert.Len(t, managerConfig.MetricsExporters, 1)
}

func TestObservabilityManager_AddTraceExporter(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Mock exporter
	exporter := &mockTraceExporter{}

	manager.AddTraceExporter(exporter)

	// Check that exporter was added
	managerConfig := manager.GetConfig()
	assert.Len(t, managerConfig.TraceExporters, 1)
}

func TestObservabilityManager_UpdateConfig(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Update config
	newConfig := observability.DefaultObservabilityConfig()
	newConfig.MetricsEnabled = false
	newConfig.TracingEnabled = false

	manager.UpdateConfig(newConfig)

	// Check that config was updated
	updatedConfig := manager.GetConfig()
	assert.False(t, updatedConfig.MetricsEnabled)
	assert.False(t, updatedConfig.TracingEnabled)
}

func TestObservabilityManager_DisabledFeatures(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	config.MetricsEnabled = false
	config.TracingEnabled = false

	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	// Metrics should be disabled
	summary := manager.GetMetricsSummary(ctx)
	assert.Nil(t, summary)

	// Tracing should be disabled
	traceID := manager.GetTraceID(ctx)
	assert.Empty(t, traceID)

	spanID := manager.GetSpanID(ctx)
	assert.Empty(t, spanID)
}

func TestObservabilityManager_ConcurrentAccess(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			// These should be safe for concurrent access
			manager.RecordQueryMetrics(ctx, "SELECT * FROM users", 100*time.Millisecond, 10, true)
			manager.RecordTransactionMetrics(ctx, "user_registration", 200*time.Millisecond, true)
			manager.RecordModelMetrics(ctx, "User", "create", 150*time.Millisecond, true)
			manager.RecordValidationMetrics(ctx, "User", 95, 5)
			manager.RecordCacheMetrics(ctx, 80, 20, 5, 100, 1000)
			manager.RecordConnectionMetrics(ctx, 25, 15, 100)
			manager.RecordErrorMetrics(ctx, "validation", "INVALID_EMAIL", assert.AnError)
			manager.RecordSystemMetrics(ctx, 45.5, 23.1, 150)

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestObservabilityManager_ContextHandling(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Start the manager first
	ctx := context.Background()
	err := manager.Start(ctx)
	assert.NoError(t, err)
	defer manager.Stop(ctx)

	// Test with background context
	manager.RecordQueryMetrics(ctx, "SELECT * FROM users", 100*time.Millisecond, 10, true)

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	manager.RecordQueryMetrics(cancelledCtx, "SELECT * FROM users", 100*time.Millisecond, 10, true)

	// Note: Testing with nil context would cause panic in the current implementation
	// as the tracer methods don't handle nil context gracefully
}

func TestObservabilityManager_EdgeCases(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	// Test with zero duration
	manager.RecordQueryMetrics(ctx, "SELECT * FROM users", 0, 0, true)

	// Test with very long duration
	manager.RecordQueryMetrics(ctx, "SELECT * FROM users", 24*time.Hour, 1000000, true)

	// Test with negative values
	manager.RecordValidationMetrics(ctx, "User", -5, -10)
	manager.RecordCacheMetrics(ctx, -10, -20, -5, -100, -1000)
	manager.RecordConnectionMetrics(ctx, -5, -10, -100)
	manager.RecordSystemMetrics(ctx, -50.0, -25.0, -50)
}

// Mock implementations for testing
type mockMetricsExporter struct{}

func (m *mockMetricsExporter) Export(ctx context.Context, metrics []observability.Metric) error {
	return nil
}

func (m *mockMetricsExporter) ExportSummary(ctx context.Context, summary map[string]interface{}) error {
	return nil
}

type mockTraceExporter struct{}

func (m *mockTraceExporter) Export(ctx context.Context, spans []*observability.Span) error {
	return nil
}

func (m *mockTraceExporter) ExportSpan(ctx context.Context, span *observability.Span) error {
	return nil
}

// Add missing test scenarios
func TestObservabilityManager_ErrorHandling(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Test with nil context
	assert.Panics(t, func() {
		manager.RecordQueryMetrics(nil, "SELECT 1", 100*time.Millisecond, 10, true)
	})

	// Test with nil span
	_ = context.Background()
	manager.EndSpan(nil, nil)
	// Should not panic

	// Test with nil error
	manager.EndSpan(nil, nil)
	// Should not panic
}

func TestObservabilityManager_ConfigurationEdgeCases(t *testing.T) {
	// Test with nil logger
	config := observability.DefaultObservabilityConfig()
	manager := observability.NewObservabilityManager(config, nil)
	assert.NotNil(t, manager)

	// Test with empty config
	emptyConfig := observability.ObservabilityConfig{}
	manager = observability.NewObservabilityManager(emptyConfig, logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{}))
	assert.NotNil(t, manager)
}

func TestObservabilityManager_MetricsEdgeCases(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	// Test with zero values
	manager.RecordQueryMetrics(ctx, "SELECT 1", 0, 0, true)
	manager.RecordTransactionMetrics(ctx, "test", 0, true)
	manager.RecordModelMetrics(ctx, "Test", "test", 0, true)
	manager.RecordValidationMetrics(ctx, "Test", 0, 0)
	manager.RecordCacheMetrics(ctx, 0, 0, 0, 0, 0)
	manager.RecordConnectionMetrics(ctx, 0, 0, 0)
	manager.RecordSystemMetrics(ctx, 0, 0, 0)

	// Test with negative values
	manager.RecordQueryMetrics(ctx, "SELECT 1", -1*time.Second, -1, true)
	manager.RecordTransactionMetrics(ctx, "test", -1*time.Second, true)
	manager.RecordModelMetrics(ctx, "Test", "test", -1*time.Second, true)
	manager.RecordValidationMetrics(ctx, "Test", -1, -1)
	manager.RecordCacheMetrics(ctx, -1, -1, -1, -1, -1)
	manager.RecordConnectionMetrics(ctx, -1, -1, -1)
	manager.RecordSystemMetrics(ctx, -1.0, -1.0, -1)

	// Test with very large values
	manager.RecordQueryMetrics(ctx, "SELECT 1", 24*365*time.Hour, 999999999, true)
	manager.RecordTransactionMetrics(ctx, "test", 24*365*time.Hour, true)
	manager.RecordModelMetrics(ctx, "Test", "test", 24*365*time.Hour, true)
	manager.RecordValidationMetrics(ctx, "Test", 999999999, 999999999)
	manager.RecordCacheMetrics(ctx, 999999999, 999999999, 999999999, 999999999, 999999999)
	manager.RecordConnectionMetrics(ctx, 999999999, 999999999, 999999999)
	manager.RecordSystemMetrics(ctx, 999999999.0, 999999999.0, 999999999)
}

func TestObservabilityManager_SpanEdgeCases(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	// Test with empty operation names
	spanCtx, span := manager.StartQuerySpan(ctx, "", "")
	assert.NotNil(t, spanCtx)
	if span != nil {
		manager.EndSpan(span, nil)
	}

	spanCtx, span = manager.StartTransactionSpan(ctx, "")
	assert.NotNil(t, spanCtx)
	if span != nil {
		manager.EndSpan(span, nil)
	}

	spanCtx, span = manager.StartModelSpan(ctx, "", "")
	assert.NotNil(t, spanCtx)
	if span != nil {
		manager.EndSpan(span, nil)
	}

	spanCtx, span = manager.StartValidationSpan(ctx, "")
	assert.NotNil(t, spanCtx)
	if span != nil {
		manager.EndSpan(span, nil)
	}

	spanCtx, span = manager.StartCacheSpan(ctx, "", "")
	assert.NotNil(t, spanCtx)
	if span != nil {
		manager.EndSpan(span, nil)
	}

	// Test with very long operation names
	longName := strings.Repeat("very_long_operation_name_", 100)
	spanCtx, span = manager.StartQuerySpan(ctx, longName, longName)
	assert.NotNil(t, spanCtx)
	if span != nil {
		manager.EndSpan(span, nil)
	}

	// Test with special characters in operation names
	specialName := "operation_with_special_chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"
	spanCtx, span = manager.StartQuerySpan(ctx, specialName, specialName)
	assert.NotNil(t, spanCtx)
	if span != nil {
		manager.EndSpan(span, nil)
	}
}

func TestObservabilityManager_ContextEdgeCases(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Test with nil carrier
	manager.InjectTraceContext(context.Background(), nil)
	// Should not panic

	// Test with empty carrier
	emptyCarrier := make(map[string]string)
	manager.InjectTraceContext(context.Background(), emptyCarrier)
	assert.Empty(t, emptyCarrier)

	// Test with nil carrier for extraction
	ctx := manager.ExtractTraceContext(nil)
	assert.NotNil(t, ctx)

	// Test with empty carrier for extraction
	emptyCarrier = map[string]string{}
	ctx = manager.ExtractTraceContext(emptyCarrier)
	assert.NotNil(t, ctx)

	// Test with invalid trace context
	invalidCarrier := map[string]string{
		"invalid_key": "invalid_value",
	}
	ctx = manager.ExtractTraceContext(invalidCarrier)
	assert.NotNil(t, ctx)
}

func TestObservabilityManager_ExporterEdgeCases(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Test with nil exporters
	manager.AddMetricsExporter(nil)
	manager.AddTraceExporter(nil)

	// Test with multiple nil exporters
	manager.AddMetricsExporter(nil)
	manager.AddMetricsExporter(nil)
	manager.AddTraceExporter(nil)
	manager.AddTraceExporter(nil)

	// Test with mixed nil and valid exporters
	validMetricsExporter := &mockMetricsExporter{}
	validTraceExporter := &mockTraceExporter{}

	manager.AddMetricsExporter(nil)
	manager.AddMetricsExporter(validMetricsExporter)
	manager.AddMetricsExporter(nil)

	manager.AddTraceExporter(nil)
	manager.AddTraceExporter(validTraceExporter)
	manager.AddTraceExporter(nil)
}

func TestObservabilityManager_ConfigUpdateEdgeCases(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Test with empty config update
	emptyConfig := observability.ObservabilityConfig{}
	manager.UpdateConfig(emptyConfig)

	// Test with same config
	manager.UpdateConfig(config)

	// Test with modified config
	modifiedConfig := config
	modifiedConfig.MetricsEnabled = !modifiedConfig.MetricsEnabled
	modifiedConfig.TracingEnabled = !modifiedConfig.TracingEnabled
	manager.UpdateConfig(modifiedConfig)
}

func TestObservabilityManager_StartStopEdgeCases(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Test starting with nil context
	err := manager.Start(nil)
	assert.Error(t, err)

	// Test stopping with nil context
	err = manager.Stop(nil)
	assert.Error(t, err)

	// Test starting multiple times
	ctx := context.Background()
	err = manager.Start(ctx)
	assert.NoError(t, err)

	err = manager.Start(ctx)
	assert.NoError(t, err)

	// Test stopping multiple times
	err = manager.Stop(ctx)
	assert.NoError(t, err)

	err = manager.Stop(ctx)
	assert.NoError(t, err)
}

func TestObservabilityManager_MetricsSummaryEdgeCases(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Test with nil context
	assert.Panics(t, func() {
		manager.GetMetricsSummary(nil)
	})

	// Test with background context
	ctx := context.Background()
	summary := manager.GetMetricsSummary(ctx)
	assert.NotNil(t, summary)

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	summary = manager.GetMetricsSummary(cancelledCtx)
	assert.NotNil(t, summary)
}

func TestObservabilityManager_TraceContextEdgeCases(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)

	// Test with background context (no trace)
	traceID := manager.GetTraceID(context.Background())
	assert.Empty(t, traceID)

	spanID := manager.GetSpanID(context.Background())
	assert.Empty(t, spanID)

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = manager.GetTraceID(cancelledCtx)
	_ = manager.GetSpanID(cancelledCtx)
}

func TestObservabilityManager_ConcurrentAccessEdgeCases(t *testing.T) {
	config := observability.DefaultObservabilityConfig()
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	// Test concurrent access with high load
	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func(id int) {
			// These should be safe for concurrent access
			manager.RecordQueryMetrics(ctx, "SELECT 1", 100*time.Millisecond, 10, true)
			manager.RecordTransactionMetrics(ctx, "test", 200*time.Millisecond, true)
			manager.RecordModelMetrics(ctx, "Test", "test", 150*time.Millisecond, true)
			manager.RecordValidationMetrics(ctx, "Test", 95, 5)
			manager.RecordCacheMetrics(ctx, 80, 20, 5, 100, 1000)
			manager.RecordConnectionMetrics(ctx, 25, 15, 100)
			manager.RecordErrorMetrics(ctx, "test", "TEST_ERROR", assert.AnError)
			manager.RecordSystemMetrics(ctx, 45.5, 23.1, 150)

			// Test span operations
			_, span := manager.StartQuerySpan(ctx, "SELECT 1", "test")
			if span != nil {
				manager.AddSpanEvent(span, "test_event", map[string]string{"key": "value"})
				manager.AddSpanAttribute(span, "test_attr", "test_value")
				manager.EndSpan(span, nil)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestObservabilityManager_DisabledFeaturesEdgeCases(t *testing.T) {
	// Test with all features disabled
	config := observability.DefaultObservabilityConfig()
	config.MetricsEnabled = false
	config.TracingEnabled = false
	config.MonitoringEnabled = false

	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	// Test metrics operations when disabled
	manager.RecordQueryMetrics(ctx, "SELECT 1", 100*time.Millisecond, 10, true)
	manager.RecordTransactionMetrics(ctx, "test", 200*time.Millisecond, true)
	manager.RecordModelMetrics(ctx, "Test", "test", 150*time.Millisecond, true)
	manager.RecordValidationMetrics(ctx, "Test", 95, 5)
	manager.RecordCacheMetrics(ctx, 80, 20, 5, 100, 1000)
	manager.RecordConnectionMetrics(ctx, 25, 15, 100)
	manager.RecordErrorMetrics(ctx, "test", "TEST_ERROR", assert.AnError)
	manager.RecordSystemMetrics(ctx, 45.5, 23.1, 150)

	// Test span operations when disabled
	spanCtx, span := manager.StartQuerySpan(ctx, "SELECT 1", "test")
	assert.NotNil(t, spanCtx)
	if span != nil {
		manager.AddSpanEvent(span, "test_event", map[string]string{"key": "value"})
		manager.AddSpanAttribute(span, "test_attr", "test_value")
		manager.EndSpan(span, nil)
	}

	// Test trace context operations when disabled
	traceID := manager.GetTraceID(ctx)
	assert.Empty(t, traceID)

	spanID := manager.GetSpanID(ctx)
	assert.Empty(t, spanID)

	// Test metrics summary when disabled
	summary := manager.GetMetricsSummary(ctx)
	assert.Nil(t, summary)
}

func TestObservabilityManager_ExportIntervalEdgeCases(t *testing.T) {
	// Test with very short export interval
	config := observability.DefaultObservabilityConfig()
	config.ExportInterval = 1 * time.Microsecond

	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	manager := observability.NewObservabilityManager(config, logger)
	ctx := context.Background()

	// Should handle very short intervals gracefully
	err := manager.Start(ctx)
	assert.NoError(t, err)
	defer manager.Stop(ctx)

	// Test with very long export interval
	config.ExportInterval = 24 * 365 * time.Hour // 1 year
	manager = observability.NewObservabilityManager(config, logger)

	err = manager.Start(ctx)
	assert.NoError(t, err)
	defer manager.Stop(ctx)

	// Test with zero export interval
	config.ExportInterval = 0
	manager = observability.NewObservabilityManager(config, logger)

	err = manager.Start(ctx)
	assert.NoError(t, err)
	defer manager.Stop(ctx)

	// Test with negative export interval
	config.ExportInterval = -1 * time.Second
	manager = observability.NewObservabilityManager(config, logger)

	err = manager.Start(ctx)
	assert.NoError(t, err)
	defer manager.Stop(ctx)
}
