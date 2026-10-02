package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

var sensitiveKeys = []string{
	"password", "token", "cookie", "secret", "private_key", "key_path", "authorization",
}

// sanitizeAttr masks sensitive information in log records.
func sanitizeAttr(groups []string, a slog.Attr) slog.Attr {
	key := strings.ToLower(a.Key)
	for _, sensitive := range sensitiveKeys {
		if strings.Contains(key, sensitive) {
			return slog.String(a.Key, "[REDACTED]")
		}
	}
	return a
}

// TracingHandler wraps an existing slog.Handler and enriches log records
// with trace_id and span_id from the active context.
type TracingHandler struct {
	next slog.Handler
}

// NewTracingHandler creates a new TracingHandler wrapping the provided handler.
func NewTracingHandler(next slog.Handler) *TracingHandler {
	return &TracingHandler{next: next}
}

func (h *TracingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *TracingHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			r.AddAttrs(
				slog.String("trace_id", span.SpanContext().TraceID().String()),
				slog.String("span_id", span.SpanContext().SpanID().String()),
			)
		} else if tid := TraceIDFromContext(ctx); tid != "" {
			r.AddAttrs(slog.String("trace_id", tid))
			if sid := SpanIDFromContext(ctx); sid != "" {
				r.AddAttrs(slog.String("span_id", sid))
			}
		}
	}
	return h.next.Handle(ctx, r)
}

func (h *TracingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TracingHandler{next: h.next.WithAttrs(attrs)}
}

func (h *TracingHandler) WithGroup(name string) slog.Handler {
	return &TracingHandler{next: h.next.WithGroup(name)}
}

// NewLogger initializes a structured slog.Logger based on the provided configuration.
func NewLogger(cfg Config, w ...io.Writer) *slog.Logger {
	var out io.Writer = os.Stdout
	if len(w) > 0 && w[0] != nil {
		out = w[0]
	}

	opts := &slog.HandlerOptions{
		Level:       cfg.SlogLevel(),
		ReplaceAttr: sanitizeAttr,
	}

	var baseHandler slog.Handler
	if strings.ToLower(strings.TrimSpace(cfg.LogFormat)) == "text" {
		baseHandler = slog.NewTextHandler(out, opts)
	} else {
		baseHandler = slog.NewJSONHandler(out, opts)
	}

	tracingHandler := NewTracingHandler(baseHandler)
	return slog.New(tracingHandler)
}
