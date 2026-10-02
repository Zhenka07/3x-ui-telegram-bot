package observability

import (
	"log/slog"
	"strings"
)

// Config defines the configuration for logging and distributed tracing.
type Config struct {
	LogLevel     string // "debug", "info", "warn", "error"
	LogFormat    string // "json" or "text"
	OTelEnabled  bool   // true to enable OpenTelemetry tracing
	ServiceName  string // OTel service.name resource attribute
	ExporterType string // "stdout" or "otlp"
	OTLPEndpoint string // e.g. "localhost:4317"
	Insecure     bool   // true to disable TLS for OTLP gRPC exporter
}

// SlogLevel converts the configured LogLevel string to slog.Level.
func (c Config) SlogLevel() slog.Level {
	switch strings.ToLower(strings.TrimSpace(c.LogLevel)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
