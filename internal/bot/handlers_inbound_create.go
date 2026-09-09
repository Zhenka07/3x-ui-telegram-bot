package bot

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"

	"github.com/zhenya/3x-ui-admin/internal/xui"
)

var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// handleInboundCreateStart starts the wizard for creating a new inbound connection.
func (b *Bot) handleInboundCreateStart(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	sess := b.fsm.GetOrCreate(chatID)
	sess.reset()
	sess.State = stateInboundWaitRemark
	b.fsm.Set(chatID, sess)

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("❌ Отмена", "inb_create_cancel")),
	)

	return c.Send(
		"Введите название (Remark) для нового подключения (например: <code>DE-Reality-Main</code>):",
		&tele.SendOptions{
			ParseMode:   tele.ModeHTML,
			ReplyMarkup: menu,
		},
	)
}

// handleInboundCreateCancel cancels inbound creation and returns to the inbounds list.
func (b *Bot) handleInboundCreateCancel(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	b.fsm.Delete(chatID)

	return b.handleInbounds(c)
}

// handleInboundRemarkInput processes user text input for the inbound remark.
func (b *Bot) handleInboundRemarkInput(c tele.Context, sess *wizardSession, text string) error {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) < 2 || len(trimmed) > 64 {
		return c.Send("⚠️ Название должно содержать от 2 до 64 символов.\n\nПопробуйте ещё раз:")
	}

	chatID := c.Chat().ID
	sess.InboundRemark = trimmed
	sess.State = stateInboundWaitPort
	b.fsm.Set(chatID, sess)

	return b.showInboundPortPrompt(c)
}

// showInboundPortPrompt displays port selection options or prompts for custom port input.
func (b *Bot) showInboundPortPrompt(c tele.Context) error {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(
			menu.Data("🎲 Случайный порт", "inb_port_rnd"),
			menu.Data("Порт 443", "inb_port_443"),
		),
		menu.Row(
			menu.Data("❌ Отмена", "inb_create_cancel"),
		),
	)

	return c.Send(
		"Укажите порт для подключения (1000–65535) или нажмите кнопку ниже:",
		&tele.SendOptions{
			ParseMode:   tele.ModeHTML,
			ReplyMarkup: menu,
		},
	)
}

// handleInboundPortRandom assigns a random available port to the inbound session.
func (b *Bot) handleInboundPortRandom(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	sess := b.fsm.Get(chatID)
	if sess == nil || sess.State != stateInboundWaitPort {
		return b.handleInboundCreateStart(c)
	}

	port := b.getRandomInboundPort()
	sess.InboundPort = port
	sess.State = stateInboundWaitSNI
	b.fsm.Set(chatID, sess)

	return b.showInboundSNIPrompt(c)
}

// handleInboundPort443 selects port 443 for the inbound session.
func (b *Bot) handleInboundPort443(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	sess := b.fsm.Get(chatID)
	if sess == nil || sess.State != stateInboundWaitPort {
		return b.handleInboundCreateStart(c)
	}

	sess.InboundPort = 443
	sess.State = stateInboundWaitSNI
	b.fsm.Set(chatID, sess)

	return b.showInboundSNIPrompt(c)
}

// handleInboundPortInput processes user text input for the inbound port.
func (b *Bot) handleInboundPortInput(c tele.Context, sess *wizardSession, text string) error {
	port, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || port < 1000 || port > 65535 {
		return c.Send("⚠️ Введите целое число от 1000 до 65535:")
	}

	chatID := c.Chat().ID
	sess.InboundPort = port
	sess.State = stateInboundWaitSNI
	b.fsm.Set(chatID, sess)

	return b.showInboundSNIPrompt(c)
}

// showInboundSNIPrompt displays SNI domain choices or prompts for custom input.
func (b *Bot) showInboundSNIPrompt(c tele.Context) error {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(
			menu.Data("🍏 apple.com", "inb_sni", "apple.com"),
			menu.Data("🟣 yahoo.com", "inb_sni", "yahoo.com"),
		),
		menu.Row(
			menu.Data("☁️ gateway.icloud.com", "inb_sni", "gateway.icloud.com"),
			menu.Data("📦 dl.google.com", "inb_sni", "dl.google.com"),
		),
		menu.Row(
			menu.Data("✏️ Ввести свой вручную", "inb_sni_custom"),
		),
		menu.Row(
			menu.Data("❌ Отмена", "inb_create_cancel"),
		),
	)

	return c.Send(
		"Выберите домен для Reality-маскировки или отправьте свой домен в чат:",
		&tele.SendOptions{
			ParseMode:   tele.ModeHTML,
			ReplyMarkup: menu,
		},
	)
}

// handleInboundSNISelect processes a preset SNI domain button selection.
func (b *Bot) handleInboundSNISelect(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	sess := b.fsm.Get(chatID)
	if sess == nil || (sess.State != stateInboundWaitSNI && sess.State != stateInboundWaitSNICustom) {
		return b.handleInboundCreateStart(c)
	}

	domain := strings.TrimSpace(c.Callback().Data)
	if domain == "" {
		return c.Send("⚠️ Некорректный домен.")
	}

	sess.InboundSNI = domain
	sess.State = stateInboundConfirm
	b.fsm.Set(chatID, sess)

	return b.showInboundConfirm(c, sess)
}

// handleInboundSNICustom prompts the user for custom SNI domain text input.
func (b *Bot) handleInboundSNICustom(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	sess := b.fsm.Get(chatID)
	if sess == nil {
		return b.handleInboundCreateStart(c)
	}

	sess.State = stateInboundWaitSNICustom
	b.fsm.Set(chatID, sess)

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("❌ Отмена", "inb_create_cancel")),
	)

	return c.Send(
		"Введите домен для Reality-маскировки (например: <code>microsoft.com</code>):",
		&tele.SendOptions{
			ParseMode:   tele.ModeHTML,
			ReplyMarkup: menu,
		},
	)
}

// handleInboundSNIInput processes user text input for custom SNI domain.
func (b *Bot) handleInboundSNIInput(c tele.Context, sess *wizardSession, text string) error {
	domain := cleanDomain(text)
	if !domainRegex.MatchString(domain) {
		return c.Send("⚠️ Введите корректный домен (например: <code>gateway.icloud.com</code>):")
	}

	chatID := c.Chat().ID
	sess.InboundSNI = domain
	sess.State = stateInboundConfirm
	b.fsm.Set(chatID, sess)

	return b.showInboundConfirm(c, sess)
}

// showInboundConfirm generates keys if needed and displays the inbound creation confirmation card.
func (b *Bot) showInboundConfirm(c tele.Context, sess *wizardSession) error {
	chatID := c.Chat().ID

	if sess.InboundPrivateKey == "" || sess.InboundPublicKey == "" {
		ctx, cancel := withTimeout()
		cert, err := b.xui.GetNewX25519Cert(ctx)
		cancel()

		if err != nil {
			b.log.Warn("не удалось получить ключи от панели, используем локальную генерацию", "err", err)
			localCert, genErr := xui.GenerateX25519Keys()
			if genErr != nil {
				return c.Send("⚠️ Ошибка генерации Reality-ключей.")
			}
			cert = localCert
		}

		sess.InboundPrivateKey = cert.PrivateKey
		sess.InboundPublicKey = cert.PublicKey
		sess.InboundShortID = xui.GenerateShortID()
		b.fsm.Set(chatID, sess)
	}

	text := fmt.Sprintf(
		"⚙️ <b>Новое подключение VLESS-Reality:</b>\n\n"+
			"• <b>Название:</b> <code>%s</code>\n"+
			"• <b>Протокол:</b> <code>vless (TCP + Reality)</code>\n"+
			"• <b>Порт:</b> <code>%d</code>\n"+
			"• <b>Маскировка (SNI):</b> <code>%s</code>\n"+
			"• <b>Public Key:</b> <code>%s</code>\n"+
			"• <b>Short ID:</b> <code>%s</code>\n\n"+
			"Создать подключение на сервере?",
		escapeHTML(sess.InboundRemark),
		sess.InboundPort,
		escapeHTML(sess.InboundSNI),
		escapeHTML(sess.InboundPublicKey),
		escapeHTML(sess.InboundShortID),
	)

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(
			menu.Data("✅ Создать", "inb_create_ok"),
			menu.Data("❌ Отмена", "inb_create_cancel"),
		),
	)

	return c.Send(text, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleInboundConfirmSubmit creates the inbound connection on the 3x-ui server.
func (b *Bot) handleInboundConfirmSubmit(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	sess := b.fsm.Get(chatID)
	if sess == nil || sess.State != stateInboundConfirm {
		return c.Send("⚠️ Сессия создания подключения не найдена. Начните заново из меню.")
	}

	ctx, cancel := withTimeout()
	defer cancel()

	spec := xui.CreateInboundSpec{
		Remark:     sess.InboundRemark,
		Port:       sess.InboundPort,
		DestDomain: sess.InboundSNI,
		PrivateKey: sess.InboundPrivateKey,
		PublicKey:  sess.InboundPublicKey,
		ShortID:    sess.InboundShortID,
	}

	inbound, err := b.xui.AddInbound(ctx, spec)
	if err != nil {
		b.log.Error("не удалось создать подключение", "error", err, "spec", spec)
		errMenu := &tele.ReplyMarkup{}
		errMenu.Inline(errMenu.Row(errMenu.Data("🔙 Главное меню", "main")))
		return c.Send(
			fmt.Sprintf("⚠️ Не удалось создать подключение:\n<code>%s</code>", escapeHTML(err.Error())),
			&tele.SendOptions{
				ParseMode:   tele.ModeHTML,
				ReplyMarkup: errMenu,
			},
		)
	}

	b.auditLog(c, "inbound_create", inbound.ID, "", fmt.Sprintf("port=%d remark=%s sni=%s", spec.Port, spec.Remark, spec.DestDomain))
	b.fsm.Delete(chatID)

	text := fmt.Sprintf(
		"✅ <b>Подключение успешно создано!</b>\n\n"+
			"ID в панели: <code>%d</code>\n"+
			"Название: <b>%s</b>\n"+
			"Порт: <code>%d</code>\n"+
			"SNI: <code>%s</code>\n\n"+
			"Теперь вы можете добавлять пользователей в это подключение.",
		inbound.ID,
		escapeHTML(spec.Remark),
		spec.Port,
		escapeHTML(spec.DestDomain),
	)

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(
			menu.Data("➕ Добавить первого клиента", "wiz_inb", strconv.Itoa(inbound.ID)),
		),
		menu.Row(
			menu.Data("📡 Список подключений", "inbs"),
			menu.Data("🔙 Главное меню", "main"),
		),
	)

	return c.Send(text, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// getRandomInboundPort returns an available port within the 15000-55000 range.
func (b *Bot) getRandomInboundPort() int {
	used := make(map[int]bool)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if list, err := b.xui.ListInbounds(ctx); err == nil {
		for _, ib := range list {
			used[ib.Port] = true
		}
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 100; i++ {
		p := 15000 + rnd.Intn(40001)
		if !used[p] {
			return p
		}
	}
	return 25443
}

// cleanDomain strips protocol schemes, ports, and trailing paths from a domain string.
func cleanDomain(input string) string {
	d := strings.TrimSpace(strings.ToLower(input))
	d = strings.TrimPrefix(d, "https://")
	d = strings.TrimPrefix(d, "http://")
	if idx := strings.Index(d, "/"); idx != -1 {
		d = d[:idx]
	}
	if idx := strings.Index(d, ":"); idx != -1 {
		d = d[:idx]
	}
	return strings.TrimSpace(d)
}
