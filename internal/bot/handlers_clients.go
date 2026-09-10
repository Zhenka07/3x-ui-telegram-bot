package bot

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v3"

	"github.com/zhenya/3x-ui-admin/internal/vless"
	"github.com/zhenya/3x-ui-admin/internal/xui"
)

// handleSelectClient displays the client details card with available actions.
func (b *Bot) handleSelectClient(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	inboundID, email, err := parseInboundEmail(c.Callback().Data)
	if err != nil {
		return c.Send("⚠️ Некорректные данные кнопки.")
	}

	ctx, cancel := withTimeout()
	defer cancel()

	ib, err := b.xui.GetInbound(ctx, inboundID)
	if err != nil {
		b.log.Error("не удалось получить инбаунд", "inbound_id", inboundID, "error", err)
		return c.Send("⚠️ Не удалось загрузить данные инбаунда.")
	}

	client := ib.FindClient(email)
	if client == nil {
		return c.Send(fmt.Sprintf("⚠️ Клиент %q не найден в инбаунде.", email))
	}

	var traffic *xui.ClientTraffic
	if t, err := b.xui.GetClientTraffics(ctx, email); err == nil {
		traffic = t
	}

	menu := &tele.ReplyMarkup{}
	cbData := fmt.Sprintf("%d|%s", inboundID, email)

	toggleLabel := "⏸ Выключить"
	if !client.Enable {
		toggleLabel = "▶️ Включить"
	}

	menu.Inline(
		menu.Row(
			menu.Data(toggleLabel, "tgl", cbData),
			menu.Data("🔄 Сбросить трафик", "rst", cbData),
		),
		menu.Row(
			menu.Data("🔗 Ссылка и QR", "lnk", cbData),
			menu.Data("❌ Удалить", "del", cbData),
		),
		menu.Row(menu.Data("🔙 К клиентам", "inb", strconv.Itoa(inboundID))),
	)

	return c.Send(clientCardText(client, traffic, ib), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleToggleClient toggles the client enabled status.
func (b *Bot) handleToggleClient(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	inboundID, email, err := parseInboundEmail(c.Callback().Data)
	if err != nil {
		return c.Send("⚠️ Некорректные данные кнопки.")
	}

	ctx, cancel := withTimeout()
	defer cancel()

	ib, err := b.xui.GetInbound(ctx, inboundID)
	if err != nil {
		return c.Send("⚠️ Не удалось загрузить инбаунд.")
	}

	client := ib.FindClient(email)
	if client == nil {
		return c.Send(fmt.Sprintf("⚠️ Клиент %q не найден.", email))
	}

	tgID := int64(0)
	if parsed, err := strconv.ParseInt(client.TgID, 10, 64); err == nil {
		tgID = parsed
	}

	spec := xui.ClientSpec{
		UUID:              client.ID,
		Email:             client.Email,
		Flow:              client.Flow,
		LimitIP:           client.LimitIP,
		TrafficLimitBytes: client.TotalGB,
		ExpiryAt:          client.ExpiryAt(),
		TelegramID:        tgID,
		SubID:             client.SubID,
		Enable:            !client.Enable,
	}

	if err := b.xui.UpdateClient(ctx, inboundID, spec); err != nil {
		b.log.Error("не удалось переключить клиента", "email", email, "error", err)
		return c.Send("⚠️ Не удалось переключить статус клиента.")
	}

	action := "включён"
	if client.Enable {
		action = "выключен"
	}
	b.log.Info("клиент переключён", "email", email, "action", action)
	b.auditLog(c, "client_toggle", inboundID, email, action)

	return b.handleSelectClient(c)
}

// handleResetConfirm shows a confirmation dialog for resetting client traffic.
func (b *Bot) handleResetConfirm(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	inboundID, email, err := parseInboundEmail(c.Callback().Data)
	if err != nil {
		return c.Send("⚠️ Некорректные данные кнопки.")
	}

	cbData := fmt.Sprintf("%d|%s", inboundID, email)
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(
			menu.Data("✅ Да, сбросить", "rst_y", cbData),
			menu.Data("❌ Отмена", "cli", cbData),
		),
	)

	return c.Send(confirmResetText(email), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleResetTraffic resets the client traffic counters.
func (b *Bot) handleResetTraffic(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	inboundID, email, err := parseInboundEmail(c.Callback().Data)
	if err != nil {
		return c.Send("⚠️ Некорректные данные кнопки.")
	}

	ctx, cancel := withTimeout()
	defer cancel()

	if err := b.xui.ResetClientTraffic(ctx, inboundID, email); err != nil {
		b.log.Error("не удалось сбросить трафик", "email", email, "error", err)
		return c.Send("⚠️ Не удалось сбросить трафик клиента.")
	}

	b.log.Info("трафик сброшен", "email", email, "inbound", inboundID)
	b.auditLog(c, "client_reset", inboundID, email, "")

	return b.handleSelectClient(c)
}

// handleDeleteConfirm shows a confirmation dialog for deleting a client.
func (b *Bot) handleDeleteConfirm(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	inboundID, email, err := parseInboundEmail(c.Callback().Data)
	if err != nil {
		return c.Send("⚠️ Некорректные данные кнопки.")
	}

	cbData := fmt.Sprintf("%d|%s", inboundID, email)
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(
			menu.Data("🗑 Да, удалить", "del_y", cbData),
			menu.Data("❌ Отмена", "cli", cbData),
		),
	)

	return c.Send(confirmDeleteText(email), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleDeleteClient deletes the client from the inbound.
func (b *Bot) handleDeleteClient(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	inboundID, email, err := parseInboundEmail(c.Callback().Data)
	if err != nil {
		return c.Send("⚠️ Некорректные данные кнопки.")
	}

	ctx, cancel := withTimeout()
	defer cancel()

	ib, err := b.xui.GetInbound(ctx, inboundID)
	if err != nil {
		return c.Send("⚠️ Не удалось загрузить инбаунд.")
	}

	client := ib.FindClient(email)
	if client == nil {
		return c.Send(fmt.Sprintf("⚠️ Клиент %q уже отсутствует.", email))
	}

	if err := b.xui.DeleteClient(ctx, inboundID, client.ID, email); err != nil {
		b.log.Error("не удалось удалить клиента", "email", email, "error", err)
		return c.Send("⚠️ Не удалось удалить клиента.")
	}

	b.log.Info("клиент удалён", "email", email, "inbound", inboundID)
	b.auditLog(c, "client_delete", inboundID, email, "uuid="+client.ID)

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("🔙 К клиентам", "inb", strconv.Itoa(inboundID))),
	)
	return c.Send(fmt.Sprintf("✅ Клиент <b>%s</b> удалён.", escapeHTML(email)), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleGenerateLink generates a VLESS link and QR code for the client.
func (b *Bot) handleGenerateLink(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	inboundID, email, err := parseInboundEmail(c.Callback().Data)
	if err != nil {
		return c.Send("⚠️ Некорректные данные кнопки.")
	}

	ctx, cancel := withTimeout()
	defer cancel()

	if err := c.Notify(tele.UploadingPhoto); err != nil {
		b.log.Debug("не удалось отправить индикатор действия", "error", err)
	}

	ib, err := b.xui.GetInbound(ctx, inboundID)
	if err != nil {
		return c.Send("⚠️ Не удалось загрузить инбаунд.")
	}

	client := ib.FindClient(email)
	if client == nil {
		return c.Send(fmt.Sprintf("⚠️ Клиент %q не найден.", email))
	}

	uri, qr, err := b.buildClientLink(ctx, ib, client)
	if err != nil {
		b.log.Error("не удалось сгенерировать ссылку", "email", email, "error", err)
		return c.Send("⚠️ Не удалось сгенерировать ссылку. Проверьте параметры инбаунда.")
	}

	photo := &tele.Photo{
		File:    tele.FromReader(bytes.NewReader(qr)),
		Caption: fmt.Sprintf("🔗 <b>%s</b>\n\n<code>%s</code>\n\n👆 Нажмите на ссылку, чтобы скопировать.", escapeHTML(email), escapeHTML(uri)),
	}

	if err := c.Send(photo, &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	}); err != nil {
		return c.Send(fmt.Sprintf("🔗 <b>%s</b>\n\n<code>%s</code>", escapeHTML(email), escapeHTML(uri)), &tele.SendOptions{
			ParseMode: tele.ModeHTML,
		})
	}
	return nil
}

// buildClientLink constructs a connection URI and QR code based on inbound and client parameters.
// It first attempts to fetch exact share links from 3x-ui subLinks API, and falls back to universal local generation.
func (b *Bot) buildClientLink(ctx context.Context, ib *xui.Inbound, client *xui.Client) (string, []byte, error) {
	var uri string

	if client.SubID != "" {
		links, subErr := b.xui.GetClientSubLinks(ctx, client.SubID)
		if subErr == nil && len(links) > 0 {
			uri = links[0]
			b.log.Debug("получена ссылка через subLinks API", "email", client.Email, "sub_id", client.SubID)
		} else if subErr != nil {
			b.log.Debug("subLinks API недоступен, переход к локальному генератору", "error", subErr)
		}
	}

	if uri == "" {
		var err error
		uri, err = vless.BuildUniversalLink(b.cfg.ServerHost, ib, client)
		if err != nil {
			return "", nil, err
		}
	}

	qr, err := vless.GenerateQR(uri, 0)
	if err != nil {
		return "", nil, fmt.Errorf("генерация QR: %w", err)
	}

	return uri, qr, nil
}

// parseInboundEmail parses callback data formatted as inboundID|email.
func parseInboundEmail(data string) (int, string, error) {
	parts := strings.SplitN(data, "|", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("ожидался формат 'id|email', получено %q", data)
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("некорректный inbound id: %w", err)
	}
	return id, parts[1], nil
}
