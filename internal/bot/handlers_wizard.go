package bot

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"

	"github.com/zhenya/3x-ui-admin/internal/vless"
	"github.com/zhenya/3x-ui-admin/internal/xui"
)

// emailRegex — допустимые символы для имени клиента.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

// handleWizardStart запускает конструктор нового подключения.
func (b *Bot) handleWizardStart(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	ctx, cancel := withTimeout()
	defer cancel()

	inbounds, err := b.xui.ListInbounds(ctx)
	if err != nil {
		b.log.Error("wizard: не удалось получить инбаунды", "error", err)
		return c.Send("⚠️ Не удалось получить список инбаундов.")
	}

	// Фильтруем только Reality-инбаунды.
	var realityInbounds []xui.Inbound
	for _, ib := range inbounds {
		if ib.IsReality && ib.Enable {
			realityInbounds = append(realityInbounds, ib)
		}
	}

	if len(realityInbounds) == 0 {
		return c.Send("⚠️ Нет доступных Reality-инбаундов.")
	}

	chatID := c.Chat().ID
	sess := b.fsm.GetOrCreate(chatID)
	sess.reset()
	sess.State = stateSelectInbound
	b.fsm.Set(chatID, sess)

	menu := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(realityInbounds)+1)
	for _, ib := range realityInbounds {
		label := fmt.Sprintf("%s :%d", ib.Remark, ib.Port)
		rows = append(rows, menu.Row(menu.Data(label, "wiz_inb", strconv.Itoa(ib.ID))))
	}
	rows = append(rows, menu.Row(menu.Data("❌ Отмена", "wiz_no")))
	menu.Inline(rows...)

	return c.Send(wizardSelectInboundText(), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleWizardInbound — шаг 1: выбран инбаунд.
func (b *Bot) handleWizardInbound(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	sess := b.fsm.Get(chatID)
	if sess == nil || sess.State != stateSelectInbound {
		return b.handleWizardStart(c)
	}

	inboundID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Send("⚠️ Некорректные данные.")
	}

	sess.InboundID = inboundID
	sess.State = stateInputEmail
	b.fsm.Set(chatID, sess)

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("❌ Отмена", "wiz_no")))

	return c.Send(wizardInputEmailText(), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleWizardTextInput обрабатывает текстовый ввод в рамках FSM.
func (b *Bot) handleWizardTextInput(c tele.Context) error {
	chatID := c.Chat().ID
	sess := b.fsm.Get(chatID)
	if sess == nil {
		return nil
	}

	text := strings.TrimSpace(c.Text())

	switch sess.State {
	case stateInputEmail:
		return b.handleEmailInput(c, sess, text)
	case stateInputTraffic:
		return b.handleTrafficInput(c, sess, text)
	case stateInputExpiry:
		return b.handleExpiryInput(c, sess, text)
	default:
		return nil
	}
}

// handleEmailInput — обработка ввода email.
func (b *Bot) handleEmailInput(c tele.Context, sess *wizardSession, text string) error {
	chatID := c.Chat().ID

	if !emailRegex.MatchString(text) {
		return c.Send("⚠️ Недопустимое имя. Допустимы: латиница, цифры, <code>_</code>, <code>-</code> (3–32 символа).\n\nПопробуйте ещё раз:", &tele.SendOptions{
			ParseMode: tele.ModeHTML,
		})
	}

	sess.Email = text
	sess.State = stateSelectTraffic
	b.fsm.Set(chatID, sess)

	return b.showTrafficOptions(c)
}

// showTrafficOptions показывает кнопки выбора лимита трафика.
func (b *Bot) showTrafficOptions(c tele.Context) error {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(
			menu.Data("30 GB", "wiz_tf", "30"),
			menu.Data("50 GB", "wiz_tf", "50"),
			menu.Data("100 GB", "wiz_tf", "100"),
		),
		menu.Row(
			menu.Data("♾ Безлимит", "wiz_tf", "0"),
			menu.Data("✏️ Ввести вручную", "wiz_tf_custom"),
		),
		menu.Row(menu.Data("❌ Отмена", "wiz_no")),
	)
	return c.Send(wizardSelectTrafficText(), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleWizardTraffic — шаг 3: выбран лимит трафика.
func (b *Bot) handleWizardTraffic(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	sess := b.fsm.Get(chatID)
	if sess == nil || sess.State != stateSelectTraffic {
		return c.Send("⚠️ Сессия конструктора не найдена. Начните заново: /start")
	}

	gb, err := strconv.ParseInt(c.Callback().Data, 10, 64)
	if err != nil {
		return c.Send("⚠️ Некорректные данные.")
	}

	sess.TrafficGB = gb
	sess.State = stateSelectExpiry
	b.fsm.Set(chatID, sess)

	return b.showExpiryOptions(c)
}

// handleWizardTrafficCustom — запрос ручного ввода лимита.
func (b *Bot) handleWizardTrafficCustom(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	sess := b.fsm.Get(chatID)
	if sess == nil || sess.State != stateSelectTraffic {
		return c.Send("⚠️ Сессия не найдена.")
	}

	sess.State = stateInputTraffic
	b.fsm.Set(chatID, sess)

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("❌ Отмена", "wiz_no")))

	return c.Send(wizardInputTrafficText(), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleTrafficInput — ручной ввод трафика.
func (b *Bot) handleTrafficInput(c tele.Context, sess *wizardSession, text string) error {
	chatID := c.Chat().ID

	gb, err := strconv.ParseInt(text, 10, 64)
	if err != nil || gb < 0 {
		return c.Send("⚠️ Введите целое неотрицательное число (в ГБ):")
	}

	sess.TrafficGB = gb
	sess.State = stateSelectExpiry
	b.fsm.Set(chatID, sess)

	return b.showExpiryOptions(c)
}

// showExpiryOptions показывает кнопки выбора срока действия.
func (b *Bot) showExpiryOptions(c tele.Context) error {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(
			menu.Data("1 мес.", "wiz_ex", "30"),
			menu.Data("3 мес.", "wiz_ex", "90"),
			menu.Data("6 мес.", "wiz_ex", "180"),
		),
		menu.Row(
			menu.Data("♾ Бессрочно", "wiz_ex", "0"),
			menu.Data("✏️ Ввести вручную", "wiz_ex_custom"),
		),
		menu.Row(menu.Data("❌ Отмена", "wiz_no")),
	)
	return c.Send(wizardSelectExpiryText(), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleWizardExpiry — шаг 4: выбран срок действия.
func (b *Bot) handleWizardExpiry(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	sess := b.fsm.Get(chatID)
	if sess == nil || sess.State != stateSelectExpiry {
		return c.Send("⚠️ Сессия не найдена.")
	}

	days, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Send("⚠️ Некорректные данные.")
	}

	sess.ExpiryDays = days
	sess.State = stateConfirm
	b.fsm.Set(chatID, sess)

	return b.showConfirmation(c, sess)
}

// handleWizardExpiryCustom — запрос ручного ввода срока.
func (b *Bot) handleWizardExpiryCustom(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	sess := b.fsm.Get(chatID)
	if sess == nil || sess.State != stateSelectExpiry {
		return c.Send("⚠️ Сессия не найдена.")
	}

	sess.State = stateInputExpiry
	b.fsm.Set(chatID, sess)

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("❌ Отмена", "wiz_no")))

	return c.Send(wizardInputExpiryText(), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleExpiryInput — ручной ввод срока.
func (b *Bot) handleExpiryInput(c tele.Context, sess *wizardSession, text string) error {
	chatID := c.Chat().ID

	days, err := strconv.Atoi(text)
	if err != nil || days < 0 {
		return c.Send("⚠️ Введите целое неотрицательное число (в днях):")
	}

	sess.ExpiryDays = days
	sess.State = stateConfirm
	b.fsm.Set(chatID, sess)

	return b.showConfirmation(c, sess)
}

// showConfirmation показывает сводку перед созданием.
func (b *Bot) showConfirmation(c tele.Context, sess *wizardSession) error {
	ctx, cancel := withTimeout()
	defer cancel()

	ib, err := b.xui.GetInbound(ctx, sess.InboundID)
	if err != nil {
		return c.Send("⚠️ Не удалось загрузить инбаунд для подтверждения.")
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(
			menu.Data("✅ Создать", "wiz_ok"),
			menu.Data("❌ Отмена", "wiz_no"),
		),
	)

	return c.Send(wizardConfirmText(sess, ib.Remark), &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: menu,
	})
}

// handleWizardConfirm — создание клиента.
func (b *Bot) handleWizardConfirm(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	sess := b.fsm.Get(chatID)
	if sess == nil || sess.State != stateConfirm {
		return c.Send("⚠️ Сессия не найдена.")
	}

	ctx, cancel := withTimeout()
	defer cancel()

	if err := c.Notify(tele.UploadingPhoto); err != nil {
		b.log.Debug("не удалось отправить индикатор", "error", err)
	}

	// Генерация UUID.
	uuid, err := vless.NewUUID()
	if err != nil {
		b.log.Error("wizard: ошибка генерации UUID", "error", err)
		b.fsm.Delete(chatID)
		return c.Send("⚠️ Ошибка генерации UUID.")
	}

	// Подготовка спецификации.
	var expiryAt time.Time
	if sess.ExpiryDays > 0 {
		expiryAt = time.Now().UTC().AddDate(0, 0, sess.ExpiryDays)
	}

	var trafficBytes int64
	if sess.TrafficGB > 0 {
		trafficBytes = sess.TrafficGB * 1024 * 1024 * 1024
	}

	spec := xui.ClientSpec{
		UUID:              uuid,
		Email:             sess.Email,
		Flow:              "xtls-rprx-vision",
		TrafficLimitBytes: trafficBytes,
		ExpiryAt:          expiryAt,
		Enable:            true,
	}

	// Создание клиента в панели.
	if err := b.xui.AddClient(ctx, sess.InboundID, spec); err != nil {
		b.log.Error("wizard: не удалось создать клиента", "email", sess.Email, "error", err)
		b.fsm.Delete(chatID)
		return c.Send("⚠️ Не удалось создать клиента в панели. Попробуйте позже.")
	}

	b.log.Info("wizard: клиент создан", "email", sess.Email, "inbound", sess.InboundID)
	b.auditLog(c, "client_create", sess.InboundID, sess.Email,
		fmt.Sprintf("trafficGB=%d expiryDays=%d uuid=%s", sess.TrafficGB, sess.ExpiryDays, uuid))

	// Генерация ссылки.
	ib, err := b.xui.GetInbound(ctx, sess.InboundID)
	if err != nil {
		b.fsm.Delete(chatID)
		return c.Send(fmt.Sprintf("✅ Клиент <b>%s</b> создан, но не удалось сгенерировать ссылку.", escapeHTML(sess.Email)), &tele.SendOptions{
			ParseMode: tele.ModeHTML,
		})
	}

	client := ib.FindClient(sess.Email)
	if client == nil {
		// Клиент только что создан, но не найден — используем данные из spec.
		tempClient := &xui.Client{
			ID:    uuid,
			Email: sess.Email,
			Flow:  "xtls-rprx-vision",
		}
		client = tempClient
	}

	uri, qr, err := b.buildClientLink(ib, client)
	b.fsm.Delete(chatID)

	if err != nil {
		b.log.Warn("wizard: не удалось сгенерировать ссылку", "email", sess.Email, "error", err)
		return c.Send(fmt.Sprintf("✅ Клиент <b>%s</b> создан, но не удалось сгенерировать ссылку:\n%s",
			escapeHTML(sess.Email), escapeHTML(err.Error())), &tele.SendOptions{
			ParseMode: tele.ModeHTML,
		})
	}

	// Отправляем QR + ссылку.
	photo := &tele.Photo{
		Caption: wizardSuccessText(sess.Email, uri),
	}
	photo.File = tele.FromReader(bytes.NewReader(qr))

	if err := c.Send(photo, &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	}); err != nil {
		// Фолбэк: текст без QR.
		return c.Send(wizardSuccessText(sess.Email, uri), &tele.SendOptions{
			ParseMode: tele.ModeHTML,
		})
	}
	return nil
}

// handleWizardCancel отменяет конструктор.
func (b *Bot) handleWizardCancel(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	chatID := c.Chat().ID
	b.fsm.Delete(chatID)

	return b.handleStart(c)
}
