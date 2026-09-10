package bot

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v3"

	"github.com/zhenya/3x-ui-admin/internal/sshtunnel"
)

// SetTunnel sets the optional active SSH tunnel for server management.
func (b *Bot) SetTunnel(t *sshtunnel.Tunnel) {
	b.tunnel = t
}

// handleLogs displays or sends the recent Xray / 3x-ui server logs.
func (b *Bot) handleLogs(c tele.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
	}

	lines := 50
	args := c.Args()
	if len(args) > 0 {
		if val, err := strconv.Atoi(args[0]); err == nil && val >= 10 && val <= 300 {
			lines = val
		}
	}

	ctx, cancel := withTimeout()
	defer cancel()

	var output string
	var fetchErr error

	// 1. Try SSH log extraction if tunnel is available
	if b.tunnel != nil {
		cmd := fmt.Sprintf(`{
			if command -v journalctl >/dev/null 2>&1; then
				journalctl -u x-ui -n %d --no-pager -o short-iso
			elif [ -r /var/log/x-ui.log ]; then
				tail -n %d /var/log/x-ui.log
			elif [ -r /var/log/x-ui/x-ui.log ]; then
				tail -n %d /var/log/x-ui/x-ui.log
			fi
		}`, lines, lines, lines)
		output, fetchErr = b.tunnel.ExecuteCommand(ctx, cmd)
	}

	// 2. Fallback to 3x-ui panel API logs
	if strings.TrimSpace(output) == "" {
		output, fetchErr = b.xui.GetLogs(ctx, lines)
	}

	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput == "" {
		if fetchErr != nil {
			return c.Send("⚠️ Не удалось получить логи: " + escapeHTML(fetchErr.Error()))
		}
		return c.Send("⚠️ Журнал логов пуст.")
	}

	// If logs are very long, send as attachment and short preview
	if len(trimmedOutput) > 3500 {
		preview := trimmedOutput
		if len(preview) > 1500 {
			preview = "..." + preview[len(preview)-1500:]
		}

		doc := &tele.Document{
			File:     tele.FromReader(bytes.NewReader([]byte(trimmedOutput))),
			FileName: "server-logs.txt",
			Caption:  fmt.Sprintf("📋 <b>Журнал работы сервера (%d строк)</b>\n\n<pre>%s</pre>", lines, escapeHTML(preview)),
		}
		return c.Send(doc, &tele.SendOptions{
			ParseMode: tele.ModeHTML,
		})
	}

	return c.Send(fmt.Sprintf("📋 <b>Журнал работы сервера (%d строк):</b>\n\n<pre>%s</pre>", lines, escapeHTML(trimmedOutput)), &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	})
}
