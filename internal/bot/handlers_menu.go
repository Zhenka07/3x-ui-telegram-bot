package bot

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"

	"github.com/zhenya/3x-ui-admin/internal/xui"
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
			replyMenu.Row(replyMenu.Text("🔍 Поиск клиента"), replyMenu.Text("🖥 Системный статус")),
		)
		_ = c.Send("Панель управления активирована.", replyMenu)
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("📡 Прокси (Inbounds)", "inbs")),
		menu.Row(menu.Data("➕ Новое подключение (Клиент)", "wiz_start")),
		menu.Row(menu.Data("➕ Создать Inbound (Reality)", "inb_create_start")),
		menu.Row(menu.Data("🔍 Поиск клиента", "cli_search")),
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

// handleSearchStart prompts the user for client search criteria.
func (b *Bot) handleSearchStart(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	sess := b.fsm.GetOrCreate(chatID)
	sess.reset()
	sess.State = stateClientSearch
	b.fsm.Set(chatID, sess)

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("❌ Отмена", "main")),
	)

	return c.Send("🔍 <b>Поиск клиента</b>\n\nВведите email или UUID клиента для поиска:", &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// performClientSearch searches for clients matching the query across all inbounds.
func (b *Bot) performClientSearch(c tele.Context, rawQuery string) error {
	query := strings.ToLower(strings.TrimSpace(rawQuery))
	if len(query) < 2 {
		return c.Send("⚠️ Слишком короткий поисковый запрос (минимум 2 символа).")
	}

	ctx, cancel := withTimeout()
	defer cancel()

	inbounds, err := b.xui.ListInbounds(ctx)
	if err != nil {
		b.log.Error("поиск клиента: не удалось получить список инбаундов", "error", err)
		return c.Send("⚠️ Не удалось выполнить поиск. Ошибка связи с панелью.")
	}

	type match struct {
		inboundID int
		remark    string
		port      int
		client    xui.Client
	}

	var matches []match
	for _, ib := range inbounds {
		for _, cl := range ib.Clients {
			if strings.Contains(strings.ToLower(cl.Email), query) || strings.Contains(strings.ToLower(cl.ID), query) {
				matches = append(matches, match{
					inboundID: ib.ID,
					remark:    ib.Remark,
					port:      ib.Port,
					client:    cl,
				})
			}
		}
	}

	if len(matches) == 0 {
		return c.Send(fmt.Sprintf("🔍 По запросу <code>%s</code> клиентов не найдено.", escapeHTML(rawQuery)), &tele.SendOptions{
			ParseMode: tele.ModeHTML,
		})
	}

	if len(matches) == 1 {
		return b.renderClientCard(c, matches[0].inboundID, matches[0].client.Email)
	}

	menu := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(matches)+1)

	for _, m := range matches {
		if len(rows) >= 10 {
			break
		}
		status := "🟢"
		if !m.client.Enable {
			status = "⏸"
		}
		label := fmt.Sprintf("%s %s (%s :%d)", status, m.client.Email, m.remark, m.port)
		rows = append(rows, menu.Row(
			menu.Data(label, "cli", fmt.Sprintf("%d|%s", m.inboundID, m.client.Email)),
		))
	}
	rows = append(rows, menu.Row(menu.Data("🔙 Главное меню", "main")))
	menu.Inline(rows...)

	return c.Send(fmt.Sprintf("🔍 Найдено совпадений: <b>%d</b> по запросу <code>%s</code>:\n\nВыберите клиента:", len(matches), escapeHTML(rawQuery)), &tele.SendOptions{
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
	case "🔍 Поиск", "🔍 Поиск клиента":
		return b.handleSearchStart(c)
	case "🖥 Системный статус", "🖥 Статус сервера", "🖥 Статус":
		return b.handleServerStatus(c)
	}

	if len(strings.TrimSpace(c.Text())) >= 2 {
		return b.performClientSearch(c, c.Text())
	}

	return c.Send("Не понимаю эту команду. Используйте /start для главного меню.", &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	})
}
