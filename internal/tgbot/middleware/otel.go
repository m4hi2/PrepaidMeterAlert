package middleware

import (
	"context"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	tele "gopkg.in/telebot.v3"
)

// CtxKey is the telebot context key under which the OTel context is stored.
const CtxKey = "otel_ctx"

// Otel is a telebot middleware that adds tracing and metrics to every handler.
type Otel struct {
	tracer           trace.Tracer
	messagesReceived metric.Int64Counter
	messagesSent     metric.Int64Counter
	handlerDuration  metric.Float64Histogram
}

// NewOtel creates an Otel middleware backed by the current global OTel providers.
// Must be called after telemetry.Setup so instruments are wired to real exporters.
func NewOtel() (*Otel, error) {
	m := otel.Meter("meterbot/tgbot")

	messagesReceived, err := m.Int64Counter(
		"tg.messages.received",
		metric.WithDescription("Number of Telegram updates received"),
	)
	if err != nil {
		return nil, err
	}

	messagesSent, err := m.Int64Counter(
		"tg.messages.sent",
		metric.WithDescription("Number of Telegram responses sent or edited"),
	)
	if err != nil {
		return nil, err
	}

	handlerDuration, err := m.Float64Histogram(
		"tg.handler.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Time to handle a Telegram update end-to-end"),
	)
	if err != nil {
		return nil, err
	}

	return &Otel{
		tracer:           otel.Tracer("meterbot/tgbot"),
		messagesReceived: messagesReceived,
		messagesSent:     messagesSent,
		handlerDuration:  handlerDuration,
	}, nil
}

// Handle is the telebot middleware function. Register it with b.Use(otel.Handle).
func (o *Otel) Handle(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		handler, updateType := classifyUpdate(c)

		baseAttrs := []attribute.KeyValue{
			attribute.String("tg.handler", handler),
			attribute.String("tg.update_type", updateType),
		}

		var userID, chatID int64
		var username string
		spanAttrs := make([]attribute.KeyValue, len(baseAttrs), len(baseAttrs)+5)
		copy(spanAttrs, baseAttrs)
		if c.Sender() != nil {
			userID = c.Sender().ID
			username = c.Sender().Username
			spanAttrs = append(spanAttrs, attribute.Int64("tg.user_id", userID))
			if username != "" {
				spanAttrs = append(spanAttrs, attribute.String("tg.username", username))
			}
		}
		if c.Chat() != nil {
			chatID = c.Chat().ID
			spanAttrs = append(spanAttrs, attribute.Int64("tg.chat_id", chatID))
		}
		text := incomingText(c)
		if len(text) > 200 {
			text = text[:200]
		}
		if text != "" {
			spanAttrs = append(spanAttrs, attribute.String("tg.message_text", text))
		}

		ctx, span := o.tracer.Start(context.Background(), "tg.handle."+handler,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(spanAttrs...),
		)
		defer span.End()

		c.Set(CtxKey, ctx)
		o.messagesReceived.Add(ctx, 1, metric.WithAttributes(baseAttrs...))

		start := time.Now()
		err := next(c)
		elapsed := float64(time.Since(start).Milliseconds())

		status := "ok"
		if err != nil {
			status = "error"
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}

		o.handlerDuration.Record(ctx, elapsed, metric.WithAttributes(
			append(baseAttrs, attribute.String("status", status))...,
		))
		o.messagesSent.Add(ctx, 1, metric.WithAttributes(baseAttrs...))

		return err
	}
}

// incomingText returns the user-supplied text from a message or callback.
func incomingText(c tele.Context) string {
	if cb := c.Callback(); cb != nil {
		return cb.Data
	}
	if msg := c.Message(); msg != nil {
		return msg.Text
	}
	return ""
}

// classifyUpdate returns (handlerName, updateType) for an incoming update.
func classifyUpdate(c tele.Context) (string, string) {
	if cb := c.Callback(); cb != nil {
		if cb.Unique != "" {
			return cb.Unique, "callback_query"
		}
		return "callback", "callback_query"
	}
	if msg := c.Message(); msg != nil && msg.Text != "" {
		if msg.Text[0] == '/' {
			cmd := strings.SplitN(msg.Text, " ", 2)[0]
			cmd = strings.TrimPrefix(cmd, "/")
			if idx := strings.Index(cmd, "@"); idx >= 0 {
				cmd = cmd[:idx]
			}
			return cmd, "command"
		}
		return "text", "message"
	}
	return "unknown", "unknown"
}
