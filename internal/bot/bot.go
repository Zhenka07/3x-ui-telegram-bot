package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tele "gopkg.in/telebot.v3"

	"github.com/zhenya/3x-ui-admin/internal/config"
	"github.com/zhenya/3x-ui-admin/internal/storage"
	"github.com/zhenya/3x-ui-admin/internal/updater"
	"github.com/zhenya/3x-ui-admin/internal/xui"
)

const handlerTimeout = 30 * time.Second

type Bot struct {
	tele    *tele.Bot
	xui     *xui.APIClient
	log     *slog.Logger
	cfg     config.Config
	fsm     *FSM
	audit   *storage.AuditRepo
	updater *updater.Checker
}

// New initializes and configures the Telegram admin bot.
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
		tele:    teleBot,
		xui:     xuiClient,
		log:     log,
		cfg:     cfg,
		fsm:     newFSM(),
		audit:   audit,
		updater: updater.NewChecker(nil, time.Hour),
	}
	b.registerMiddleware()
	b.registerHandlers()

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			b.fsm.Cleanup()
		}
	}()

	return b, nil
}

// registerHandlers registers command and callback query handlers with the bot.
func (b *Bot) registerHandlers() {
	b.tele.Handle("/start", b.handleStart)
	b.tele.Handle("/help", b.handleStart)
	b.tele.Handle("/inbounds", b.handleInbounds)
	b.tele.Handle("/search", b.handleSearchStart)
	b.tele.Handle("/status", b.handleServerStatus)

	b.tele.Handle("\fmain", b.handleStart)
	b.tele.Handle("\finbs", b.handleInbounds)
	b.tele.Handle("\fwiz_start", b.handleWizardStart)
	b.tele.Handle("\fcli_search", b.handleSearchStart)
	b.tele.Handle("\fsys", b.handleServerStatus)

	b.tele.Handle("\finb", b.handleSelectInbound)
	b.tele.Handle("\fpg", b.handleClientPage)
	b.tele.Handle("\fcli", b.handleSelectClient)

	b.tele.Handle("\ftgl", b.handleToggleClient)
	b.tele.Handle("\frst", b.handleResetConfirm)
	b.tele.Handle("\frst_y", b.handleResetTraffic)
	b.tele.Handle("\flnk", b.handleGenerateLink)
	b.tele.Handle("\fdel", b.handleDeleteConfirm)
	b.tele.Handle("\fdel_y", b.handleDeleteClient)

	b.tele.Handle("\fwiz_inb", b.handleWizardInbound)
	b.tele.Handle("\fwiz_tf", b.handleWizardTraffic)
	b.tele.Handle("\fwiz_tf_custom", b.handleWizardTrafficCustom)
	b.tele.Handle("\fwiz_ex", b.handleWizardExpiry)
	b.tele.Handle("\fwiz_ex_custom", b.handleWizardExpiryCustom)
	b.tele.Handle("\fwiz_ok", b.handleWizardConfirm)
	b.tele.Handle("\fwiz_no", b.handleWizardCancel)

	b.tele.Handle("\finb_create_start", b.handleInboundCreateStart)
	b.tele.Handle("\finb_create_cancel", b.handleInboundCreateCancel)
	b.tele.Handle("\finb_port_rnd", b.handleInboundPortRandom)
	b.tele.Handle("\finb_port_443", b.handleInboundPort443)
	b.tele.Handle("\finb_sni", b.handleInboundSNISelect)
	b.tele.Handle("\finb_sni_custom", b.handleInboundSNICustom)
	b.tele.Handle("\finb_create_ok", b.handleInboundConfirmSubmit)

	b.tele.Handle(tele.OnText, b.handleText)
}

// Start begins polling Telegram for updates and blocks until context cancellation.
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

// withTimeout creates a new context with default handler execution timeout.
func withTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), handlerTimeout)
}

// auditLog records an administrative action into the audit storage.
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
