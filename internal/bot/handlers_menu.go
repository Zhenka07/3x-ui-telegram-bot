package bot

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

// handleStart показывает главное меню админки.
func (b *Bot) handleStart(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	// Если вызов через команду (не callback), активируем постоянную reply-клавиатуру-дублёр.
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
		menu.Row(menu.Data("➕ Новое подключение", "wiz_start")),
		menu.Row(menu.Data("🖥 Системный статус", "sys")),
	)

	return c.Send(welcomeText(), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleServerStatus отображает нагрузку сервера и статус Xray.
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

// handleText обрабатывает произвольный текст.
// Если пользователь находится в сессии конструктора — перенаправляет на FSM.
// Также обрабатывает кнопки Reply-клавиатуры-дублёра.
func (b *Bot) handleText(c tele.Context) error {
	chatID := c.Chat().ID

	// Если активна сессия конструктора — обрабатываем текст как ввод FSM.
	if b.fsm.IsActive(chatID) {
		return b.handleWizardTextInput(c)
	}

	switch strings.TrimSpace(c.Text()) {
	case "📡 Прокси", "📡 Прокси (Inbounds)":
		return b.handleInbounds(c)
	case "➕ Новое подключение":
		return b.handleWizardStart(c)
	case "🖥 Системный статус", "🖥 Статус сервера", "🖥 Статус":
		return b.handleServerStatus(c)
	}

	return c.Send("Не понимаю эту команду. Используйте /start для главного меню.", &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	})
}
