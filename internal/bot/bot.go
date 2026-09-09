// Package bot реализует Telegram-слой админ-панели на базе telebot.v3.
package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tele "gopkg.in/telebot.v3"

	"github.com/zhenya/3x-ui-admin/internal/config"
	"github.com/zhenya/3x-ui-admin/internal/storage"
	"github.com/zhenya/3x-ui-admin/internal/xui"
)

// handlerTimeout ограничивает время обработки одного обновления.
const handlerTimeout = 30 * time.Second

// Bot — Telegram-бот админ-панели 3x-ui.
type Bot struct {
	tele *tele.Bot
	xui  *xui.APIClient
	log  *slog.Logger
	cfg  config.Config
	fsm  *FSM
	audit *storage.AuditRepo
}

// New создаёт и настраивает бота: поллер, middleware и обработчики.
func New(cfg config.Config, xuiClient *xui.APIClient, audit *storage.AuditRepo, log *slog.Logger) (*Bot, error) {
	if xuiClient == nil {
		return nil, fmt.Errorf("создание бота: не передан клиент 3x-ui")
	}
	if log == nil {
		return nil, fmt.Errorf("создание бота: не передан логгер")
	}

	pollerTimeout := cfg.Telegram.LongPollerTimeout
	if pollerTimeout <= 0 {
		pollerTimeout = 10 * time.Second
	}

	settings := tele.Settings{
		Token:  cfg.Telegram.Token,
		Poller: &tele.LongPoller{Timeout: pollerTimeout},
		OnError: func(err error, c tele.Context) {
			var userID int64
			if c != nil && c.Sender() != nil {
				userID = c.Sender().ID
			}
			log.Error("необработанная ошибка обработчика", "tg_id", userID, "error", err)
		},
	}

	teleBot, err := tele.NewBot(settings)
	if err != nil {
		return nil, fmt.Errorf("инициализация telebot: %w", err)
	}

	b := &Bot{
		tele:  teleBot,
		xui:   xuiClient,
		log:   log,
		cfg:   cfg,
		fsm:   newFSM(),
		audit: audit,
	}
	b.registerMiddleware()
	b.registerHandlers()

	// Периодическая очистка истекших FSM-сессий.
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			b.fsm.Cleanup()
		}
	}()

	return b, nil
}

// registerHandlers привязывает обработчики команд и inline-кнопок.
func (b *Bot) registerHandlers() {
	// Команды.
	b.tele.Handle("/start", b.handleStart)
	b.tele.Handle("/help", b.handleStart)
	b.tele.Handle("/inbounds", b.handleInbounds)
	b.tele.Handle("/status", b.handleServerStatus)

	// Inline-кнопки: главное меню.
	b.tele.Handle("\fmain", b.handleStart)
	b.tele.Handle("\finbs", b.handleInbounds)
	b.tele.Handle("\fwiz_start", b.handleWizardStart)
	b.tele.Handle("\fsys", b.handleServerStatus)

	// Inline-кнопки: инбаунды и клиенты.
	b.tele.Handle("\finb", b.handleSelectInbound)
	b.tele.Handle("\fpg", b.handleClientPage)
	b.tele.Handle("\fcli", b.handleSelectClient)

	// Inline-кнопки: действия с клиентом.
	b.tele.Handle("\ftgl", b.handleToggleClient)
	b.tele.Handle("\frst", b.handleResetConfirm)
	b.tele.Handle("\frst_y", b.handleResetTraffic)
	b.tele.Handle("\flnk", b.handleGenerateLink)
	b.tele.Handle("\fdel", b.handleDeleteConfirm)
	b.tele.Handle("\fdel_y", b.handleDeleteClient)

	// Inline-кнопки: конструктор (wizard).
	b.tele.Handle("\fwiz_inb", b.handleWizardInbound)
	b.tele.Handle("\fwiz_tf", b.handleWizardTraffic)
	b.tele.Handle("\fwiz_tf_custom", b.handleWizardTrafficCustom)
	b.tele.Handle("\fwiz_ex", b.handleWizardExpiry)
	b.tele.Handle("\fwiz_ex_custom", b.handleWizardExpiryCustom)
	b.tele.Handle("\fwiz_ok", b.handleWizardConfirm)
	b.tele.Handle("\fwiz_no", b.handleWizardCancel)

	// Произвольный текст: обрабатывается FSM или как неизвестная команда.
	b.tele.Handle(tele.OnText, b.handleText)
}

// Start запускает обработку обновлений и блокируется до отмены контекста.
func (b *Bot) Start(ctx context.Context) error {
	stopped := make(chan struct{})

	go func() {
		<-ctx.Done()
		b.log.Info("останавливаем Telegram-поллер")
		b.tele.Stop()
		close(stopped)
	}()

	b.log.Info("бот запущен", "username", b.tele.Me.Username)
	b.tele.Start()

	select {
	case <-stopped:
	default:
	}

	if err := ctx.Err(); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		return fmt.Errorf("работа бота прервана: %w", err)
	}
	return nil
}

// withTimeout создаёт контекст с ограничением времени обработки.
func withTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), handlerTimeout)
}

// auditLog пишет запись о действии администратора в журнал аудита
// (не ошибка, если хранилище закрыто или недоступно — только warn в лог).
func (b *Bot) auditLog(c tele.Context, action string, inboundID int, clientEmail, details string) {
	if b.audit == nil {
		return
	}
	var adminID int64
	if sender := c.Sender(); sender != nil {
		adminID = sender.ID
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := b.audit.Log(ctx, storage.AuditLog{
		AdminID:     adminID,
		Action:      action,
		InboundID:   inboundID,
		ClientEmail: clientEmail,
		Details:     details,
	}); err != nil {
		b.log.Warn("не удалось записать запись аудита", "action", action, "err", err)
	}
}
