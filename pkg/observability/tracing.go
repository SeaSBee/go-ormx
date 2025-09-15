package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/SeaSBee/go-ormx/pkg/logging"
)

// TraceID represents a unique trace identifier
type TraceID string

// SpanID represents a unique span identifier
type SpanID string

// SpanStatus represents the status of a span
type SpanStatus string

const (
	SpanStatusOK    SpanStatus = "ok"
	SpanStatusError SpanStatus = "error"
)

// SpanKind represents the kind of span
type SpanKind string

const (
	SpanKindInternal SpanKind = "internal"
	SpanKindServer   SpanKind = "server"
	SpanKindClient   SpanKind = "client"
	SpanKindProducer SpanKind = "producer"
	SpanKindConsumer SpanKind = "consumer"
)

// Span represents a tracing span
type Span struct {
	TraceID      TraceID           `json:"trace_id"`
	SpanID       SpanID            `json:"span_id"`
	ParentSpanID SpanID            `json:"parent_span_id,omitempty"`
	Name         string            `json:"name"`
	Kind         SpanKind          `json:"kind"`
	Status       SpanStatus        `json:"status"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time,omitempty"`
	Duration     time.Duration     `json:"duration,omitempty"`
	Attributes   map[string]string `json:"attributes"`
	Events       []SpanEvent       `json:"events,omitempty"`
	Error        error             `json:"error,omitempty"`
	Context      context.Context   `json:"-"`
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string            `json:"name"`
	Timestamp  time.Time         `json:"timestamp"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Tracer interface for distributed tracing
type Tracer interface {
	StartSpan(ctx context.Context, name string, kind SpanKind) (context.Context, *Span)
	EndSpan(span *Span, err error)
	AddSpanEvent(span *Span, name string, attributes map[string]string)
	AddSpanAttribute(span *Span, key, value string)
	GetTraceID(ctx context.Context) TraceID
	GetSpanID(ctx context.Context) SpanID
	InjectTraceContext(ctx context.Context, carrier map[string]string)
	ExtractTraceContext(carrier map[string]string) context.Context
}

// BaseTracer implements basic tracing functionality
type BaseTracer struct {
	spans   map[SpanID]*Span
	mutex   sync.RWMutex
	logger  logging.Logger
	enabled bool
}

// NewBaseTracer creates a new base tracer
func NewBaseTracer(logger logging.Logger, enabled bool) *BaseTracer {
	return &BaseTracer{
		spans:   make(map[SpanID]*Span),
		logger:  logger,
		enabled: enabled,
	}
}

// StartSpan starts a new span
func (bt *BaseTracer) StartSpan(ctx context.Context, name string, kind SpanKind) (context.Context, *Span) {
	if !bt.enabled {
		return ctx, nil
	}

	span := &Span{
		TraceID:    bt.generateTraceID(ctx),
		SpanID:     bt.generateSpanID(),
		Name:       name,
		Kind:       kind,
		Status:     SpanStatusOK,
		StartTime:  time.Now(),
		Attributes: make(map[string]string),
		Events:     make([]SpanEvent, 0),
		Context:    ctx,
	}

	// Extract parent span ID from context
	if parentSpanID := bt.GetSpanID(ctx); parentSpanID != "" {
		span.ParentSpanID = parentSpanID
	}

	// Store span
	bt.mutex.Lock()
	bt.spans[span.SpanID] = span
	bt.mutex.Unlock()

	// Create new context with span
	newCtx := context.WithValue(ctx, "span_id", span.SpanID)
	newCtx = context.WithValue(newCtx, "trace_id", span.TraceID)

	bt.logger.Debug(ctx, "Span started",
		logging.String("trace_id", string(span.TraceID)),
		logging.String("span_id", string(span.SpanID)),
		logging.String("name", name),
		logging.String("kind", string(kind)))

	return newCtx, span
}

// EndSpan ends a span
func (bt *BaseTracer) EndSpan(span *Span, err error) {
	if !bt.enabled || span == nil {
		return
	}

	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)

	if err != nil {
		span.Status = SpanStatusError
		span.Error = err
	}

	// Remove span from storage
	bt.mutex.Lock()
	delete(bt.spans, span.SpanID)
	bt.mutex.Unlock()

	bt.logger.Debug(span.Context, "Span ended",
		logging.String("trace_id", string(span.TraceID)),
		logging.String("span_id", string(span.SpanID)),
		logging.String("name", span.Name),
		logging.String("status", string(span.Status)),
		logging.Duration("duration", span.Duration),
		logging.ErrorField("error", err))
}

// AddSpanEvent adds an event to a span
func (bt *BaseTracer) AddSpanEvent(span *Span, name string, attributes map[string]string) {
	if !bt.enabled || span == nil {
		return
	}

	event := SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attributes,
	}

	span.Events = append(span.Events, event)

	bt.logger.Debug(span.Context, "Span event added",
		logging.String("trace_id", string(span.TraceID)),
		logging.String("span_id", string(span.SpanID)),
		logging.String("event_name", name),
		logging.Any("attributes", attributes))
}

// AddSpanAttribute adds an attribute to a span
func (bt *BaseTracer) AddSpanAttribute(span *Span, key, value string) {
	if !bt.enabled || span == nil {
		return
	}

	span.Attributes[key] = value

	bt.logger.Debug(span.Context, "Span attribute added",
		logging.String("trace_id", string(span.TraceID)),
		logging.String("span_id", string(span.SpanID)),
		logging.String("key", key),
		logging.String("value", value))
}

// GetTraceID gets the trace ID from context
func (bt *BaseTracer) GetTraceID(ctx context.Context) TraceID {
	if traceID, ok := ctx.Value("trace_id").(TraceID); ok {
		return traceID
	}
	return ""
}

// GetSpanID gets the span ID from context
func (bt *BaseTracer) GetSpanID(ctx context.Context) SpanID {
	if spanID, ok := ctx.Value("span_id").(SpanID); ok {
		return spanID
	}
	return ""
}

// InjectTraceContext injects trace context into a carrier
func (bt *BaseTracer) InjectTraceContext(ctx context.Context, carrier map[string]string) {
	if !bt.enabled {
		return
	}

	traceID := bt.GetTraceID(ctx)
	spanID := bt.GetSpanID(ctx)

	if traceID != "" {
		carrier["trace_id"] = string(traceID)
	}
	if spanID != "" {
		carrier["span_id"] = string(spanID)
	}
}

// ExtractTraceContext extracts trace context from a carrier
func (bt *BaseTracer) ExtractTraceContext(carrier map[string]string) context.Context {
	if !bt.enabled {
		return context.Background()
	}

	ctx := context.Background()

	if traceID, ok := carrier["trace_id"]; ok {
		ctx = context.WithValue(ctx, "trace_id", TraceID(traceID))
	}
	if spanID, ok := carrier["span_id"]; ok {
		ctx = context.WithValue(ctx, "span_id", SpanID(spanID))
	}

	return ctx
}

// generateTraceID generates a new trace ID
func (bt *BaseTracer) generateTraceID(ctx context.Context) TraceID {
	// Check if trace ID already exists in context
	if existingTraceID := bt.GetTraceID(ctx); existingTraceID != "" {
		return existingTraceID
	}

	// Generate new trace ID
	return TraceID(fmt.Sprintf("trace_%d", time.Now().UnixNano()))
}

// generateSpanID generates a new span ID
func (bt *BaseTracer) generateSpanID() SpanID {
	return SpanID(fmt.Sprintf("span_%d", time.Now().UnixNano()))
}

// ORMTracer represents ORM-specific tracing functionality
type ORMTracer struct {
	*BaseTracer
}

// NewORMTracer creates a new ORM tracer
func NewORMTracer(logger logging.Logger, enabled bool) *ORMTracer {
	return &ORMTracer{
		BaseTracer: NewBaseTracer(logger, enabled),
	}
}

// StartQuerySpan starts a span for a database query
func (ot *ORMTracer) StartQuerySpan(ctx context.Context, query string, operation string) (context.Context, *Span) {
	spanCtx, span := ot.StartSpan(ctx, fmt.Sprintf("orm.%s", operation), SpanKindInternal)

	if span != nil {
		ot.AddSpanAttribute(span, "query", query)
		ot.AddSpanAttribute(span, "operation", operation)
		ot.AddSpanAttribute(span, "component", "orm")
	}

	return spanCtx, span
}

// StartTransactionSpan starts a span for a database transaction
func (ot *ORMTracer) StartTransactionSpan(ctx context.Context, operation string) (context.Context, *Span) {
	spanCtx, span := ot.StartSpan(ctx, fmt.Sprintf("orm.transaction.%s", operation), SpanKindInternal)

	if span != nil {
		ot.AddSpanAttribute(span, "operation", operation)
		ot.AddSpanAttribute(span, "component", "orm")
		ot.AddSpanAttribute(span, "type", "transaction")
	}

	return spanCtx, span
}

// StartModelSpan starts a span for a model operation
func (ot *ORMTracer) StartModelSpan(ctx context.Context, model string, operation string) (context.Context, *Span) {
	spanCtx, span := ot.StartSpan(ctx, fmt.Sprintf("orm.model.%s.%s", model, operation), SpanKindInternal)

	if span != nil {
		ot.AddSpanAttribute(span, "model", model)
		ot.AddSpanAttribute(span, "operation", operation)
		ot.AddSpanAttribute(span, "component", "orm")
		ot.AddSpanAttribute(span, "type", "model")
	}

	return spanCtx, span
}

// StartValidationSpan starts a span for validation
func (ot *ORMTracer) StartValidationSpan(ctx context.Context, model string) (context.Context, *Span) {
	spanCtx, span := ot.StartSpan(ctx, fmt.Sprintf("orm.validation.%s", model), SpanKindInternal)

	if span != nil {
		ot.AddSpanAttribute(span, "model", model)
		ot.AddSpanAttribute(span, "component", "orm")
		ot.AddSpanAttribute(span, "type", "validation")
	}

	return spanCtx, span
}

// StartCacheSpan starts a span for cache operations
func (ot *ORMTracer) StartCacheSpan(ctx context.Context, operation string, key string) (context.Context, *Span) {
	spanCtx, span := ot.StartSpan(ctx, fmt.Sprintf("orm.cache.%s", operation), SpanKindInternal)

	if span != nil {
		ot.AddSpanAttribute(span, "operation", operation)
		ot.AddSpanAttribute(span, "key", key)
		ot.AddSpanAttribute(span, "component", "orm")
		ot.AddSpanAttribute(span, "type", "cache")
	}

	return spanCtx, span
}

// AddQuerySpanEvent adds a query-specific event to a span
func (ot *ORMTracer) AddQuerySpanEvent(span *Span, eventName string, rowsAffected int64, duration time.Duration) {
	if span == nil {
		return
	}

	attributes := map[string]string{
		"rows_affected": fmt.Sprintf("%d", rowsAffected),
		"duration_ms":   fmt.Sprintf("%.2f", float64(duration.Milliseconds())),
	}

	ot.AddSpanEvent(span, eventName, attributes)
}

// AddErrorSpanEvent adds an error event to a span
func (ot *ORMTracer) AddErrorSpanEvent(span *Span, err error) {
	if span == nil {
		return
	}

	attributes := map[string]string{
		"error_type": fmt.Sprintf("%T", err),
		"error":      err.Error(),
	}

	ot.AddSpanEvent(span, "error", attributes)
}

// TraceExporter interface for exporting traces
type TraceExporter interface {
	Export(ctx context.Context, spans []*Span) error
	ExportSpan(ctx context.Context, span *Span) error
}

// JaegerExporter exports traces to Jaeger
type JaegerExporter struct {
	logger logging.Logger
}

// NewJaegerExporter creates a new Jaeger exporter
func NewJaegerExporter(logger logging.Logger) *JaegerExporter {
	return &JaegerExporter{
		logger: logger,
	}
}

// Export exports multiple spans to Jaeger
func (je *JaegerExporter) Export(ctx context.Context, spans []*Span) error {
	for _, span := range spans {
		if err := je.ExportSpan(ctx, span); err != nil {
			return err
		}
	}
	return nil
}

// ExportSpan exports a single span to Jaeger
func (je *JaegerExporter) ExportSpan(ctx context.Context, span *Span) error {
	// This is a simplified implementation
	// In a real implementation, you would send the span to Jaeger
	je.logger.Info(ctx, "Jaeger span export",
		logging.String("trace_id", string(span.TraceID)),
		logging.String("span_id", string(span.SpanID)),
		logging.String("name", span.Name),
		logging.String("status", string(span.Status)),
		logging.Duration("duration", span.Duration))
	return nil
}

// ZipkinExporter exports traces to Zipkin
type ZipkinExporter struct {
	logger logging.Logger
}

// NewZipkinExporter creates a new Zipkin exporter
func NewZipkinExporter(logger logging.Logger) *ZipkinExporter {
	return &ZipkinExporter{
		logger: logger,
	}
}

// Export exports multiple spans to Zipkin
func (ze *ZipkinExporter) Export(ctx context.Context, spans []*Span) error {
	for _, span := range spans {
		if err := ze.ExportSpan(ctx, span); err != nil {
			return err
		}
	}
	return nil
}

// ExportSpan exports a single span to Zipkin
func (ze *ZipkinExporter) ExportSpan(ctx context.Context, span *Span) error {
	// This is a simplified implementation
	// In a real implementation, you would send the span to Zipkin
	ze.logger.Info(ctx, "Zipkin span export",
		logging.String("trace_id", string(span.TraceID)),
		logging.String("span_id", string(span.SpanID)),
		logging.String("name", span.Name),
		logging.String("status", string(span.Status)),
		logging.Duration("duration", span.Duration))
	return nil
}
