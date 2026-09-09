package bot

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zhenya/3x-ui-admin/internal/xui"
)

var errPanicRecovered = errors.New("внутренняя ошибка обработчика")

// ── Главное меню ────────────────────────────────────────────────────

func welcomeText() string {
	var sb strings.Builder
	sb.WriteString("🛡 <b>Админ-панель 3x-ui</b>\n\n")
	sb.WriteString("Управление прокси-сервером через Telegram.\n\n")
	sb.WriteString("Выберите действие:")
	return sb.String()
}

// ── Системный статус ────────────────────────────────────────────────

func serverStatusText(status *xui.ServerStatus) string {
	var sb strings.Builder
	sb.WriteString("🖥 <b>Системный статус</b>\n\n")

	xrayStatus := "🟢 работает"
	if status.Xray.State != "running" {
		xrayStatus = "🔴 " + status.Xray.State
	}
	sb.WriteString(fmt.Sprintf("• <b>Xray:</b> %s (v%s)\n", xrayStatus, status.Xray.Version))
	sb.WriteString(fmt.Sprintf("• <b>CPU:</b> %.1f%%\n", status.CPU))

	if status.Mem.Total > 0 {
		memUsed := status.Mem.Current
		memTotal := status.Mem.Total
		memPercent := float64(memUsed) / float64(memTotal) * 100
		sb.WriteString(fmt.Sprintf("• <b>RAM:</b> %s / %s (%.1f%%)\n",
			formatBytes(memUsed), formatBytes(memTotal), memPercent))
	}

	if status.Disk.Total > 0 {
		diskUsed := status.Disk.Current
		diskTotal := status.Disk.Total
		diskPercent := float64(diskUsed) / float64(diskTotal) * 100
		sb.WriteString(fmt.Sprintf("• <b>Диск:</b> %s / %s (%.1f%%)\n",
			formatBytes(diskUsed), formatBytes(diskTotal), diskPercent))
	}

	if status.Uptime > 0 {
		d := time.Duration(status.Uptime) * time.Second
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		mins := int(d.Minutes()) % 60
		sb.WriteString(fmt.Sprintf("• <b>Аптайм:</b> %d д. %d ч. %d мин.\n", days, hours, mins))
	}

	sb.WriteString(fmt.Sprintf("• <b>Соединения:</b> TCP: %d, UDP: %d\n", status.TCPCount, status.UDPCount))

	return sb.String()
}

// ── Инбаунды ────────────────────────────────────────────────────────

func inboundsListText(inbounds []xui.Inbound) string {
	var sb strings.Builder
	sb.WriteString("📡 <b>Прокси (Inbounds)</b>\n\n")

	if len(inbounds) == 0 {
		sb.WriteString("Нет доступных инбаундов.")
		return sb.String()
	}

	for _, ib := range inbounds {
		status := "🟢"
		if !ib.Enable {
			status = "🔴"
		}
		sb.WriteString(fmt.Sprintf("%s <b>%s</b> :%d [%s]\n",
			status, escapeHTML(ib.Remark), ib.Port, ib.Protocol))
		sb.WriteString(fmt.Sprintf("   📊 Трафик: %s ↑ / %s ↓ • Клиенты: %d\n\n",
			formatBytes(ib.Up), formatBytes(ib.Down), len(ib.Clients)))
	}
	return sb.String()
}

// ── Список клиентов ─────────────────────────────────────────────────

func clientsListText(ib *xui.Inbound, page, totalPages int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👥 <b>Клиенты — %s</b> (:%d)\n",
		escapeHTML(ib.Remark), ib.Port))
	sb.WriteString(fmt.Sprintf("Страница %d/%d\n\n", page+1, totalPages))
	return sb.String()
}

// ── Карточка клиента ────────────────────────────────────────────────

func clientCardText(client *xui.Client, traffic *xui.ClientTraffic, ib *xui.Inbound) string {
	var sb strings.Builder

	status := "🟢 активен"
	if !client.Enable {
		status = "🔴 выключен"
	}

	sb.WriteString(fmt.Sprintf("👤 <b>%s</b>\n", escapeHTML(client.Email)))
	sb.WriteString(fmt.Sprintf("Инбаунд: %s (:%d)\n", escapeHTML(ib.Remark), ib.Port))
	sb.WriteString(fmt.Sprintf("Статус: %s\n", status))
	sb.WriteString(fmt.Sprintf("UUID: <code>%s</code>\n\n", escapeHTML(client.ID)))

	// Блок трафика.
	if traffic != nil {
		sb.WriteString("📈 <b>Трафик</b>\n")
		sb.WriteString(fmt.Sprintf("• Использовано: <b>%s</b>\n", formatBytes(traffic.Used())))
		sb.WriteString(fmt.Sprintf("• Отправлено: %s\n", formatBytes(traffic.Up)))
		sb.WriteString(fmt.Sprintf("• Получено: %s\n", formatBytes(traffic.Down)))

		if remaining, limited := traffic.Remaining(); limited {
			sb.WriteString(fmt.Sprintf("• Лимит: %s\n", formatBytes(traffic.Total)))
			sb.WriteString(fmt.Sprintf("• Осталось: <b>%s</b>\n", formatBytes(remaining)))
		} else {
			sb.WriteString("• Лимит: <b>без ограничений</b>\n")
		}
	} else {
		// Нет данных из API — показываем то, что есть в клиенте.
		if client.TotalGB > 0 {
			sb.WriteString(fmt.Sprintf("📊 Лимит трафика: %s\n", formatBytes(client.TotalGB)))
		} else {
			sb.WriteString("📊 Лимит трафика: <b>безлимит</b>\n")
		}
	}

	// Блок срока действия.
	expiry := client.ExpiryAt()
	if expiry.IsZero() {
		sb.WriteString("\n🕒 Срок действия: <b>бессрочно</b>\n")
	} else {
		sb.WriteString(fmt.Sprintf("\n🕒 Действует до: <b>%s</b>\n", formatTime(expiry)))
		now := time.Now().UTC()
		if now.After(expiry) {
			sb.WriteString("❌ Срок действия <b>истёк</b>\n")
		} else {
			sb.WriteString(fmt.Sprintf("⏳ Осталось: <b>%s</b>\n", formatDuration(expiry.Sub(now))))
		}
	}

	if client.LimitIP > 0 {
		sb.WriteString(fmt.Sprintf("\n📱 Лимит устройств: <b>%d</b>\n", client.LimitIP))
	}

	return sb.String()
}

// ── Конструктор (Wizard) ────────────────────────────────────────────

func wizardSelectInboundText() string {
	return "🔧 <b>Конструктор нового подключения</b>\n\nШаг 1/4: Выберите Reality-инбаунд:"
}

func wizardInputEmailText() string {
	return "🔧 <b>Шаг 2/4: Имя клиента</b>\n\nВведите email (имя) нового клиента.\n" +
		"Допустимы: латиница, цифры, <code>_</code>, <code>-</code> (3–32 символа)."
}

func wizardSelectTrafficText() string {
	return "🔧 <b>Шаг 3/4: Лимит трафика</b>\n\nВыберите лимит трафика для клиента:"
}

func wizardInputTrafficText() string {
	return "Введите лимит трафика в ГБ (целое число):"
}

func wizardSelectExpiryText() string {
	return "🔧 <b>Шаг 4/4: Срок действия</b>\n\nВыберите срок действия подключения:"
}

func wizardInputExpiryText() string {
	return "Введите срок действия в днях (целое число):"
}

func wizardConfirmText(sess *wizardSession, ibRemark string) string {
	var sb strings.Builder
	sb.WriteString("✅ <b>Подтверждение создания</b>\n\n")
	sb.WriteString(fmt.Sprintf("• Инбаунд: <b>%s</b>\n", escapeHTML(ibRemark)))
	sb.WriteString(fmt.Sprintf("• Имя: <b>%s</b>\n", escapeHTML(sess.Email)))

	if sess.TrafficGB > 0 {
		sb.WriteString(fmt.Sprintf("• Трафик: <b>%d GB</b>\n", sess.TrafficGB))
	} else {
		sb.WriteString("• Трафик: <b>безлимит</b>\n")
	}

	if sess.ExpiryDays > 0 {
		sb.WriteString(fmt.Sprintf("• Срок: <b>%d дн.</b>\n", sess.ExpiryDays))
	} else {
		sb.WriteString("• Срок: <b>бессрочно</b>\n")
	}

	sb.WriteString("\nВсё верно?")
	return sb.String()
}

func wizardSuccessText(email, uri string) string {
	var sb strings.Builder
	sb.WriteString("🎉 <b>Клиент успешно создан!</b>\n\n")
	sb.WriteString(fmt.Sprintf("Email: <b>%s</b>\n\n", escapeHTML(email)))
	sb.WriteString("<code>")
	sb.WriteString(escapeHTML(uri))
	sb.WriteString("</code>\n\n")
	sb.WriteString("👆 Нажмите на ссылку, чтобы скопировать.")
	return sb.String()
}

// ── Подтверждения ───────────────────────────────────────────────────

func confirmDeleteText(email string) string {
	return fmt.Sprintf("⚠️ Вы уверены, что хотите <b>удалить</b> клиента <b>%s</b>?\n\n"+
		"Это действие необратимо.", escapeHTML(email))
}

func confirmResetText(email string) string {
	return fmt.Sprintf("⚠️ Сбросить счётчик трафика для клиента <b>%s</b>?", escapeHTML(email))
}

// ── Утилиты форматирования ──────────────────────────────────────────

func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(s)
}

func formatBytes(b int64) string {
	if b <= 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	value := float64(b)
	idx := -1
	for value >= unit && idx < len(units)-1 {
		value /= unit
		idx++
	}
	if value >= 100 {
		return fmt.Sprintf("%.0f %s", value, units[idx])
	}
	return fmt.Sprintf("%.2f %s", value, units[idx])
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.UTC().Format("02.01.2006 15:04") + " UTC"
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "истёк"
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24

	switch {
	case days > 0:
		return fmt.Sprintf("%d д. %d ч.", days, hours)
	case hours > 0:
		return fmt.Sprintf("%d ч. %d мин.", hours, int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%d мин.", int(d.Minutes()))
	}
}
