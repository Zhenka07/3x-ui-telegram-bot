package bot

import (
	"fmt"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	tele "gopkg.in/telebot.v3"

	"github.com/zhenya/3x-ui-admin/internal/observability"
)

// registerMiddleware registers global bot middlewares in sequential order.
func (b *Bot) registerMiddleware() {
	b.tele.Use(
		observability.TelegramTraceMiddleware(b.tracer, b.log),
		b.recoverMiddleware,
		b.adminOnlyMiddleware,
	)
}

// recoverMiddleware recovers from panics in handler executions and records them in the trace span.
func (b *Bot) recoverMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				ctx := observability.ContextFromTele(c)
				span := trace.SpanFromContext(ctx)
				if span.SpanContext().IsValid() {
					span.RecordError(fmt.Errorf("panic: %v", r))
					span.SetStatus(codes.Error, "panic recovered")
				}
				var userID int64
				if c.Sender() != nil {
					userID = c.Sender().ID
				}
				b.log.ErrorContext(ctx, "паника в обработчике", "tg_id", userID, "panic", r)
				err = errPanicRecovered
			}
		}()
		return next(c)
	}
}

// adminOnlyMiddleware restricts bot access to configured administrator IDs only.
func (b *Bot) adminOnlyMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		sender := c.Sender()
		if sender == nil {
			return nil
		}
		if !b.cfg.Telegram.IsAdmin(sender.ID) {
			ctx := observability.ContextFromTele(c)
			b.log.WarnContext(ctx, "доступ запрещён (не админ)", "tg_id", sender.ID, "username", sender.Username)
			return nil
		}
		return next(c)
	}
}

