package observability

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	tele "gopkg.in/telebot.v3"
)

type contextKey string

const (
	traceIDKey contextKey = "obs_trace_id"
	spanIDKey  contextKey = "obs_span_id"
	teleCtxKey            = "obs_trace_ctx"
)

// ContextWithTraceID returns a new context with the given trace ID.
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceIDFromContext extracts trace_id from the active OpenTelemetry span,
// or falls back to an explicitly stored trace ID in the context.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	if val, ok := ctx.Value(traceIDKey).(string); ok {
		return val
	}
	return ""
}

// SpanIDFromContext extracts span_id from the active OpenTelemetry span,
// or falls back to an explicitly stored span ID in the context.
func SpanIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	if val, ok := ctx.Value(spanIDKey).(string); ok {
		return val
	}
	return ""
}

// SetTeleContext stores the tracing context.Context into telebot.Context.
func SetTeleContext(c tele.Context, ctx context.Context) {
	if c != nil && ctx != nil {
		c.Set(teleCtxKey, ctx)
	}
}

// ContextFromTele retrieves the tracing context.Context from telebot.Context,
// or returns context.Background() if not found.
func ContextFromTele(c tele.Context) context.Context {
	if c == nil {
		return context.Background()
	}
	if val := c.Get(teleCtxKey); val != nil {
		if ctx, ok := val.(context.Context); ok && ctx != nil {
			return ctx
		}
	}
	return context.Background()
}
