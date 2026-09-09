package bot

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

// handleStart displays the main menu of the admin bot.
func (b *Bot) handleStart(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	if c.Callback() == nil {
		replyMenu := &tele.ReplyMarkup{ResizeKeyboard: true}
		replyMenu.Reply(
			replyMenu.Row(replyMenu.Text("📡 Прокси"), replyMenu.Text("➕ Новое подключение")),
			replyMenu.Row(replyMenu.Text("🖥 Системный статус")),
		)
		_ = c.Send("Панель управления активирована.", replyMenu)
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("📡 Прокси (Inbounds)", "inbs")),
		menu.Row(menu.Data("➕ Новое подключение (Клиент)", "wiz_start")),
		menu.Row(menu.Data("➕ Создать Inbound (Reality)", "inb_create_start")),
		menu.Row(menu.Data("🖥 Системный статус", "sys")),
	)

	return c.Send(welcomeText(), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleServerStatus displays server load metrics and Xray service status.
func (b *Bot) handleServerStatus(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	ctx, cancel := withTimeout()
	defer cancel()

	status, err := b.xui.GetServerStatus(ctx)
	if err != nil {
		b.log.Error("не удалось получить статус сервера", "error", err)
		return c.Send("⚠️ Не удалось получить статус сервера.")
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("🔄 Обновить", "sys")),
		menu.Row(menu.Data("🔙 Главное меню", "main")),
	)

	return c.Send(serverStatusText(status), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleText processes arbitrary text messages, routing to the FSM wizard or reply menu actions.
func (b *Bot) handleText(c tele.Context) error {
	chatID := c.Chat().ID

	if b.fsm.IsActive(chatID) {
		return b.handleWizardTextInput(c)
	}

	switch strings.TrimSpace(c.Text()) {
	case "📡 Прокси", "📡 Прокси (Inbounds)":
		return b.handleInbounds(c)
	case "➕ Новое подключение", "➕ Новое подключение (Клиент)":
		return b.handleWizardStart(c)
	case "➕ Создать Inbound", "➕ Создать Inbound (Reality)":
		return b.handleInboundCreateStart(c)
	case "🖥 Системный статус", "🖥 Статус сервера", "🖥 Статус":
		return b.handleServerStatus(c)
	}

	return c.Send("Не понимаю эту команду. Используйте /start для главного меню.", &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	})
}
