package bot

import (
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v3"
)

const clientsPerPage = 10

// handleInbounds показывает список всех инбаундов.
func (b *Bot) handleInbounds(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	ctx, cancel := withTimeout()
	defer cancel()

	inbounds, err := b.xui.ListInbounds(ctx)
	if err != nil {
		b.log.Error("не удалось получить список инбаундов", "error", err)
		return c.Send("⚠️ Не удалось получить список инбаундов. Попробуйте позже.", &tele.SendOptions{
			ParseMode: tele.ModeHTML,
		})
	}

	menu := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(inbounds)+1)

	for _, ib := range inbounds {
		status := "🟢"
		if !ib.Enable {
			status = "🔴"
		}
		label := fmt.Sprintf("%s %s :%d [%d клиентов]", status, ib.Remark, ib.Port, len(ib.Clients))
		rows = append(rows, menu.Row(menu.Data(label, "inb", strconv.Itoa(ib.ID))))
	}
	rows = append(rows, menu.Row(menu.Data("➕ Создать Inbound (Reality)", "inb_create_start")))
	rows = append(rows, menu.Row(menu.Data("🔙 Главное меню", "main")))
	menu.Inline(rows...)

	return c.Send(inboundsListText(inbounds), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleSelectInbound показывает список клиентов выбранного инбаунда.
func (b *Bot) handleSelectInbound(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	inboundID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Send("⚠️ Некорректные данные кнопки.")
	}

	return b.showClients(c, inboundID, 0)
}

// handleClientPage обрабатывает пагинацию списка клиентов.
func (b *Bot) handleClientPage(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	parts := strings.SplitN(c.Callback().Data, "|", 2)
	if len(parts) != 2 {
		return c.Send("⚠️ Некорректные данные кнопки.")
	}

	inboundID, err := strconv.Atoi(parts[0])
	if err != nil {
		return c.Send("⚠️ Некорректные данные кнопки.")
	}
	page, err := strconv.Atoi(parts[1])
	if err != nil {
		return c.Send("⚠️ Некорректные данные кнопки.")
	}

	return b.showClients(c, inboundID, page)
}

// showClients отображает страницу клиентов инбаунда.
func (b *Bot) showClients(c tele.Context, inboundID, page int) error {
	ctx, cancel := withTimeout()
	defer cancel()

	ib, err := b.xui.GetInbound(ctx, inboundID)
	if err != nil {
		b.log.Error("не удалось получить инбаунд", "inbound_id", inboundID, "error", err)
		return c.Send("⚠️ Не удалось загрузить данные инбаунда.")
	}

	clients := ib.Clients
	totalPages := (len(clients) + clientsPerPage - 1) / clientsPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * clientsPerPage
	end := start + clientsPerPage
	if end > len(clients) {
		end = len(clients)
	}
	pageClients := clients[start:end]

	menu := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(pageClients)+3)

	for _, cl := range pageClients {
		status := "🟢"
		if !cl.Enable {
			status = "🔴"
		}
		label := fmt.Sprintf("%s %s", status, cl.Email)
		rows = append(rows, menu.Row(
			menu.Data(label, "cli", fmt.Sprintf("%d|%s", inboundID, cl.Email)),
		))
	}

	// Навигация.
	var navBtns []tele.Btn
	if page > 0 {
		navBtns = append(navBtns, menu.Data("⬅️ Назад", "pg", fmt.Sprintf("%d|%d", inboundID, page-1)))
	}
	if page < totalPages-1 {
		navBtns = append(navBtns, menu.Data("➡️ Далее", "pg", fmt.Sprintf("%d|%d", inboundID, page+1)))
	}
	if len(navBtns) > 0 {
		rows = append(rows, menu.Row(navBtns...))
	}

	rows = append(rows, menu.Row(menu.Data("🔙 К инбаундам", "inbs")))
	menu.Inline(rows...)

	text := clientsListText(ib, page, totalPages)
	if len(clients) == 0 {
		text += "Нет клиентов в этом инбаунде."
	}

	return c.Send(text, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}
