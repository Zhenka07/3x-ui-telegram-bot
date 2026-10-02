package observability

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	tele "gopkg.in/telebot.v3"
)

// TelegramTraceMiddleware creates an OpenTelemetry span for every incoming Telegram update,
// extracts metadata, enriches logging context, and measures execution duration.
func TelegramTraceMiddleware(tracer trace.Tracer, logger *slog.Logger) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if tracer == nil {
				tracer = Tracer("telegram-bot")
			}
			if logger == nil {
				logger = slog.Default()
			}

			var (
				userID     int64
				username   string
				chatID     int64
				updateType = "unknown"
				payload    = ""
				opName     = "telegram.update"
			)

			if sender := c.Sender(); sender != nil {
				userID = sender.ID
				username = sender.Username
			}
			if chat := c.Chat(); chat != nil {
				chatID = chat.ID
			}

			if c.Callback() != nil {
				updateType = "callback"
				payload = c.Callback().Data
				opName = fmt.Sprintf("telegram.callback: %s", payload)
			} else if c.Message() != nil {
				text := strings.TrimSpace(c.Message().Text)
				payload = text
				if strings.HasPrefix(text, "/") {
					updateType = "command"
					cmd := strings.Split(text, " ")[0]
					opName = fmt.Sprintf("telegram.command: %s", cmd)
				} else {
					updateType = "message"
				}
			}

			ctx, span := tracer.Start(context.Background(), opName,
				trace.WithAttributes(
					attribute.Int64("telegram.user_id", userID),
					attribute.String("telegram.username", username),
					attribute.Int64("telegram.chat_id", chatID),
					attribute.String("telegram.update_type", updateType),
					attribute.String("telegram.payload", payload),
				),
			)
			defer span.End()

			// Propagate trace context into telebot.Context
			SetTeleContext(c, ctx)

			started := time.Now()
			err := next(c)
			duration := time.Since(started)

			span.SetAttributes(attribute.Int64("telegram.duration_ms", duration.Milliseconds()))

			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				logger.ErrorContext(ctx, "ошибка обработки telegram обновления",
					slog.Int64("tg_id", userID),
					slog.String("username", username),
					slog.String("update_type", updateType),
					slog.String("payload", payload),
					slog.Int64("duration_ms", duration.Milliseconds()),
					slog.Any("error", err),
				)
				return err
			}

			span.SetStatus(codes.Ok, "OK")
			logger.DebugContext(ctx, "telegram обновление обработано",
				slog.Int64("tg_id", userID),
				slog.String("username", username),
				slog.String("update_type", updateType),
				slog.String("payload", payload),
				slog.Int64("duration_ms", duration.Milliseconds()),
			)
			return nil
		}
	}
}
