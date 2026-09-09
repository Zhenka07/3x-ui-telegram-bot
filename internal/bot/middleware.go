package bot

import (
	"time"

	tele "gopkg.in/telebot.v3"
)

// registerMiddleware registers global bot middlewares in sequential order.
func (b *Bot) registerMiddleware() {
	b.tele.Use(b.recoverMiddleware, b.loggingMiddleware, b.adminOnlyMiddleware)
}

// recoverMiddleware recovers from panics in handler executions.
func (b *Bot) recoverMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				var userID int64
				if c.Sender() != nil {
					userID = c.Sender().ID
				}
				b.log.Error("паника в обработчике", "tg_id", userID, "panic", r)
				err = errPanicRecovered
			}
		}()
		return next(c)
	}
}

// loggingMiddleware logs update processing details and duration.
func (b *Bot) loggingMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		started := time.Now()

		var (
			userID   int64
			username string
		)
		if sender := c.Sender(); sender != nil {
			userID = sender.ID
			username = sender.Username
		}
		payload := c.Text()

		err := next(c)

		attrs := []any{
			"tg_id", userID,
			"username", username,
			"text", payload,
			"duration", time.Since(started).String(),
		}
		if err != nil {
			b.log.Error("обновление обработано с ошибкой", append(attrs, "error", err)...)
			return err
		}
		b.log.Debug("обновление обработано", attrs...)
		return nil
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
			b.log.Warn("доступ запрещён (не админ)", "tg_id", sender.ID, "username", sender.Username)
			return nil
		}
		return next(c)
	}
}
