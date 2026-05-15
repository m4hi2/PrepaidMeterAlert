package logger

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type ctxKey struct{}

// WithAttrs returns a context carrying slog attributes that will be injected
// into every log line using that context.
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	return context.WithValue(ctx, ctxKey{}, attrs)
}

type contextHandler struct {
	inner slog.Handler
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(ctxKey{}).([]slog.Attr); ok {
		r.AddAttrs(attrs...)
	}
	// Inject OTel trace/span IDs so logs can be correlated with traces in Grafana.
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{inner: h.inner.WithGroup(name)}
}

// InitLogger configures the default slog logger with the given level and output format.
func InitLogger(level slog.Level, format Format) {
	opts := &slog.HandlerOptions{Level: level}

	var inner slog.Handler
	if format == FormatJSON {
		inner = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		inner = slog.NewTextHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(&contextHandler{inner: inner}))
}
