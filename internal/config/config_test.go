package config

import (
	"os"
	"testing"
	"time"
)

func clearEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"BOT_TOKEN", "TELEGRAM_BOT_TOKEN",
		"ADMIN_IDS",
		"XUI_BASE_URL", "XUI_USERNAME", "XUI_PASSWORD",
		"SERVER_HOST", "REALITY_SERVER_IP", "VLESS_SERVER_ADDR",
		"DB_PATH", "ENV_FILE", "LOG_LEVEL",
		"XUI_TIMEOUT", "TELEGRAM_POLLER_TIMEOUT", "XUI_INSECURE_SKIP_VERIFY",
	}
	for _, k := range keys {
		_ = os.Unsetenv(k)
	}
}

func TestConfigValidation_Missing(t *testing.T) {
	clearEnv(t)
	t.Setenv("ENV_FILE", "/nonexistent/.env")

	_, err := Load()
	if err == nil {
		t.Fatal("ожидалась ошибка валидации при отсутствии переменных")
	}
}

func TestConfigLoad_Success(t *testing.T) {
	clearEnv(t)
	t.Setenv("ENV_FILE", "/nonexistent/.env")
	t.Setenv("BOT_TOKEN", "123456:ABC-DEF")
	t.Setenv("ADMIN_IDS", "981784126, 12345")
	t.Setenv("XUI_BASE_URL", "http://127.0.0.1:2053")
	t.Setenv("XUI_USERNAME", "admin")
	t.Setenv("XUI_PASSWORD", "secret")
	t.Setenv("SERVER_HOST", "1.2.3.4")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("XUI_INSECURE_SKIP_VERIFY", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("неожиданная ошибка загрузки: %v", err)
	}

	if cfg.Telegram.Token != "123456:ABC-DEF" {
		t.Errorf("got token %q, want %q", cfg.Telegram.Token, "123456:ABC-DEF")
	}
	if len(cfg.Telegram.AdminIDs) != 2 || !cfg.Telegram.IsAdmin(981784126) || !cfg.Telegram.IsAdmin(12345) {
		t.Errorf("got admin IDs %v", cfg.Telegram.AdminIDs)
	}
	if cfg.Telegram.IsAdmin(99999) {
		t.Errorf("expected 99999 to not be admin")
	}
	if cfg.ServerHost != "1.2.3.4" {
		t.Errorf("got server host %q, want %q", cfg.ServerHost, "1.2.3.4")
	}
	if !cfg.XUI.InsecureSkipVerify {
		t.Errorf("expected InsecureSkipVerify=true")
	}
	if cfg.XUI.Timeout != 20*time.Second {
		t.Errorf("got timeout %v, want 20s", cfg.XUI.Timeout)
	}
}
