package telemetry

import (
	"context"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type bunStartKey struct{}
type bunSpanKey struct{}

// BunHook implements bun.QueryHook, adding traces and metrics to every SQL query.
type BunHook struct {
	tracer        trace.Tracer
	queryDuration metric.Float64Histogram
	queryErrors   metric.Int64Counter
}

// NewBunHook creates a BunHook using the current global OTel providers.
// Call after telemetry.Setup so the instruments are backed by real exporters.
func NewBunHook() (*BunHook, error) {
	meter := otel.Meter("meterbot/db")
	tracer := otel.Tracer("meterbot/db")

	queryDuration, err := meter.Float64Histogram(
		"db.query.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Duration of database queries"),
	)
	if err != nil {
		return nil, err
	}

	queryErrors, err := meter.Int64Counter(
		"db.query.errors",
		metric.WithDescription("Number of database query errors"),
	)
	if err != nil {
		return nil, err
	}

	return &BunHook{
		tracer:        tracer,
		queryDuration: queryDuration,
		queryErrors:   queryErrors,
	}, nil
}

func (h *BunHook) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	ctx, span := h.tracer.Start(ctx, "db.query",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	ctx = context.WithValue(ctx, bunSpanKey{}, span)
	return context.WithValue(ctx, bunStartKey{}, time.Now())
}

func (h *BunHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	span, ok := ctx.Value(bunSpanKey{}).(trace.Span)
	if !ok || span == nil {
		return
	}
	defer span.End()

	start, _ := ctx.Value(bunStartKey{}).(time.Time)
	elapsed := float64(time.Since(start).Milliseconds())
	op := queryOp(event.Query)

	attrs := []attribute.KeyValue{
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", op),
	}
	span.SetAttributes(attrs...)
	// Truncate long statements to avoid huge span payloads.
	stmt := event.Query
	if len(stmt) > 1000 {
		stmt = stmt[:1000] + "…"
	}
	span.SetAttributes(attribute.String("db.statement", stmt))

	h.queryDuration.Record(ctx, elapsed, metric.WithAttributes(attrs...))

	if event.Err != nil {
		span.SetStatus(codes.Error, event.Err.Error())
		span.RecordError(event.Err)
		h.queryErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

func queryOp(q string) string {
	q = strings.TrimLeft(q, " \t\r\n")
	i := strings.IndexByte(q, ' ')
	if i < 0 {
		return "unknown"
	}
	return strings.ToLower(q[:i])
}
