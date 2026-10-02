package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	tele "gopkg.in/telebot.v3"
)

func TestTracingHandler_TraceIDInjection(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		LogLevel:  "debug",
		LogFormat: "json",
	}
	logger := NewLogger(cfg, &buf)

	// Case 1: Trace ID from context helper
	ctx := ContextWithTraceID(context.Background(), "test-trace-12345")
	logger.InfoContext(ctx, "hello test trace")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse log json: %v", err)
	}

	if record["trace_id"] != "test-trace-12345" {
		t.Errorf("expected trace_id=test-trace-12345, got %v", record["trace_id"])
	}
	if record["msg"] != "hello test trace" {
		t.Errorf("expected msg='hello test trace', got %v", record["msg"])
	}
}

func TestSanitizeAttr(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		LogLevel:  "info",
		LogFormat: "json",
	}
	logger := NewLogger(cfg, &buf)

	logger.Info("user auth",
		slog.String("username", "admin"),
		slog.String("password", "supersecret123"),
		slog.String("bot_token", "12345:ABCDE"),
		slog.String("session_cookie", "cookie_val"),
	)

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse log json: %v", err)
	}

	if record["password"] != "[REDACTED]" {
		t.Errorf("expected password to be [REDACTED], got %v", record["password"])
	}
	if record["bot_token"] != "[REDACTED]" {
		t.Errorf("expected bot_token to be [REDACTED], got %v", record["bot_token"])
	}
	if record["session_cookie"] != "[REDACTED]" {
		t.Errorf("expected session_cookie to be [REDACTED], got %v", record["session_cookie"])
	}
	if record["username"] != "admin" {
		t.Errorf("expected username to be 'admin', got %v", record["username"])
	}
}

func TestInitTracer_Disabled(t *testing.T) {
	cfg := Config{
		OTelEnabled: false,
	}

	tracer, shutdown, err := InitTracer(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tracer == nil {
		t.Fatal("expected non-nil tracer")
	}

	ctx, span := tracer.Start(context.Background(), "test-span")
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()

	if err := shutdown(ctx); err != nil {
		t.Errorf("unexpected error on shutdown: %v", err)
	}
}

func TestInitTracer_EnabledStdout(t *testing.T) {
	cfg := Config{
		OTelEnabled:  true,
		ServiceName:  "test-service",
		ExporterType: "stdout",
	}

	tracer, shutdown, err := InitTracer(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tracer == nil {
		t.Fatal("expected non-nil tracer")
	}

	ctx, span := tracer.Start(context.Background(), "test-span-enabled")
	if !span.SpanContext().IsValid() {
		t.Errorf("expected valid span context")
	}
	span.End()

	if err := shutdown(ctx); err != nil {
		t.Errorf("unexpected error on shutdown: %v", err)
	}
}

type mockTeleContext struct {
	tele.Context
	store   map[string]any
	sender  *tele.User
	chat    *tele.Chat
	message *tele.Message
}

func (m *mockTeleContext) Get(key string) any {
	return m.store[key]
}

func (m *mockTeleContext) Set(key string, val any) {
	m.store[key] = val
}

func (m *mockTeleContext) Sender() *tele.User {
	return m.sender
}

func (m *mockTeleContext) Chat() *tele.Chat {
	return m.chat
}

func (m *mockTeleContext) Message() *tele.Message {
	return m.message
}

func (m *mockTeleContext) Callback() *tele.Callback {
	return nil
}

func TestTelegramTraceMiddleware(t *testing.T) {
	tracer, shutdown, err := InitTracer(context.Background(), Config{
		OTelEnabled:  true,
		ServiceName:  "test-bot",
		ExporterType: "stdout",
	})
	if err != nil {
		t.Fatalf("failed to init tracer: %v", err)
	}
	defer shutdown(context.Background())

	logger := slog.Default()
	mw := TelegramTraceMiddleware(tracer, logger)

	mockCtx := &mockTeleContext{
		store: make(map[string]any),
		sender: &tele.User{
			ID:       12345,
			Username: "testuser",
		},
		chat: &tele.Chat{
			ID: 67890,
		},
		message: &tele.Message{
			Text: "/start",
		},
	}

	var handlerExecuted bool
	handler := func(c tele.Context) error {
		handlerExecuted = true
		ctx := ContextFromTele(c)
		if ctx == nil {
			t.Error("expected non-nil context from tele")
		}
		span := trace.SpanFromContext(ctx)
		if !span.SpanContext().IsValid() {
			t.Error("expected valid span in handler")
		}
		return nil
	}

	err = mw(handler)(mockCtx)
	if err != nil {
		t.Fatalf("unexpected middleware error: %v", err)
	}
	if !handlerExecuted {
		t.Error("expected handler to be executed")
	}
}

func TestTelegramTraceMiddleware_ErrorRecording(t *testing.T) {
	tracer := otel.GetTracerProvider().Tracer("test-bot")
	logger := slog.Default()
	mw := TelegramTraceMiddleware(tracer, logger)

	mockCtx := &mockTeleContext{
		store: make(map[string]any),
		sender: &tele.User{
			ID: 999,
		},
		chat: &tele.Chat{
			ID: 888,
		},
		message: &tele.Message{
			Text: "hello",
		},
	}

	testErr := errors.New("something went wrong")
	handler := func(c tele.Context) error {
		return testErr
	}

	err := mw(handler)(mockCtx)
	if !errors.Is(err, testErr) {
		t.Errorf("expected testErr, got %v", err)
	}
}

func TestConfig_SlogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
	}

	for _, tt := range tests {
		cfg := Config{LogLevel: tt.input}
		if lvl := cfg.SlogLevel(); lvl != tt.expected {
			t.Errorf("for input %q expected %v, got %v", tt.input, tt.expected, lvl)
		}
	}
}
