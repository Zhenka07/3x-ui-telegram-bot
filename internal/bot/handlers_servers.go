package bot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v3"

	"github.com/zhenya/3x-ui-admin/internal/xui"
)

// handleServersList displays all configured servers and marks the currently active one.
func (b *Bot) handleServersList(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	ctx, cancel := withTimeout()
	defer cancel()

	servers, err := b.serverRepo.List(ctx)
	if err != nil {
		b.log.Error("не удалось получить список серверов", "error", err)
		return c.Send("⚠️ Не удалось получить список серверов.")
	}

	menu := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(servers)+1)

	for _, s := range servers {
		mark := "⚪"
		if s.IsActive {
			mark = "✅"
		}
		label := fmt.Sprintf("%s %s (%s)", mark, s.Name, s.ServerHost)
		rows = append(rows, menu.Row(
			menu.Data(label, "sw_srv", s.ID),
		))
	}

	rows = append(rows, menu.Row(menu.Data("🔙 Главное меню", "main")))
	menu.Inline(rows...)

	return c.Send("🌐 <b>Управление серверами 3x-ui</b>\n\nНажмите на сервер, чтобы переключить активное подключение:", &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleSwitchServer switches the active server connection.
func (b *Bot) handleSwitchServer(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	serverID := c.Callback().Data
	if serverID == "" {
		return c.Send("⚠️ Некорректный идентификатор сервера.")
	}

	ctx, cancel := withTimeout()
	defer cancel()

	if err := b.SwitchServer(ctx, serverID); err != nil {
		b.log.Error("ошибка переключения сервера", "server_id", serverID, "error", err)
		return c.Send("⚠️ Не удалось переключить сервер: " + escapeHTML(err.Error()))
	}

	s, _ := b.serverRepo.Get(ctx, serverID)
	name := serverID
	if s != nil {
		name = s.Name
	}

	b.auditLog(c, "server_switch", 0, "", "server_id="+serverID+" name="+name)
	_ = c.Respond(&tele.CallbackResponse{Text: "Подключено: " + name})

	return b.handleServersList(c)
}

// SwitchServer activates a server and switches the in-memory API client.
func (b *Bot) SwitchServer(ctx context.Context, serverID string) error {
	s, err := b.serverRepo.Get(ctx, serverID)
	if err != nil || s == nil {
		return fmt.Errorf("сервер %q не найден: %w", serverID, err)
	}

	newClient, err := xui.New(xui.Config{
		BaseURL:            s.BaseURL,
		Username:           s.Username,
		Password:           s.Password,
		Timeout:            b.cfg.XUI.Timeout,
		InsecureSkipVerify: b.cfg.XUI.InsecureSkipVerify,
	}, b.log)
	if err != nil {
		return fmt.Errorf("создание клиента для сервера %q: %w", s.Name, err)
	}

	if err := b.serverRepo.SetActive(ctx, serverID); err != nil {
		return fmt.Errorf("сохранение активного сервера: %w", err)
	}

	b.clientMu.Lock()
	b.xui = newClient
	b.cfg.ServerHost = s.ServerHost
	b.clientMu.Unlock()

	b.log.Info("активный сервер переключен", "id", s.ID, "name", s.Name, "host", s.ServerHost)
	return nil
}
