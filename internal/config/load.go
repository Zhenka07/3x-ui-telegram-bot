package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultEnvFile       = ".env"
	defaultDBPath        = "data/admin.db"
	defaultXUITimeout    = 20 * time.Second
	defaultPollerTimeout = 10 * time.Second
	defaultLogLevel      = "info"
)

// Load reads configuration from environment variables and an optional .env file, then validates it.
func Load() (*Config, error) {
	envFile := defaultEnvFile
	if custom := strings.TrimSpace(os.Getenv("ENV_FILE")); custom != "" {
		envFile = custom
	}
	if err := loadEnvFile(envFile); err != nil {
		return nil, err
	}

	cfg := &Config{
		LogLevel: strings.ToLower(lookupString("LOG_LEVEL", defaultLogLevel)),
	}

	if err := cfg.loadTelegram(); err != nil {
		return nil, err
	}
	if err := cfg.loadXUI(); err != nil {
		return nil, err
	}
	if err := cfg.loadSSH(); err != nil {
		return nil, err
	}
	cfg.Storage.DBPath = lookupString("DB_PATH", defaultDBPath)
	cfg.ServerHost = lookupStringFirst([]string{"SERVER_HOST", "REALITY_SERVER_IP", "VLESS_SERVER_ADDR"}, "")

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("валидация конфигурации: %w", err)
	}
	return cfg, nil
}

// loadTelegram loads Telegram configuration parameters from environment variables.
func (c *Config) loadTelegram() error {
	c.Telegram.Token = lookupStringFirst([]string{"BOT_TOKEN", "TELEGRAM_BOT_TOKEN"}, "")

	adminIDs, err := lookupIDList("ADMIN_IDS")
	if err != nil {
		return err
	}
	c.Telegram.AdminIDs = adminIDs

	pollerTimeout, err := lookupDuration("TELEGRAM_POLLER_TIMEOUT", defaultPollerTimeout)
	if err != nil {
		return err
	}
	c.Telegram.LongPollerTimeout = pollerTimeout
	return nil
}

// loadXUI loads 3x-ui configuration parameters from environment variables.
func (c *Config) loadXUI() error {
	c.XUI.BaseURL = strings.TrimRight(lookupString("XUI_BASE_URL", ""), "/")
	c.XUI.Username = lookupString("XUI_USERNAME", "")
	c.XUI.Password = lookupString("XUI_PASSWORD", "")

	timeout, err := lookupDuration("XUI_TIMEOUT", defaultXUITimeout)
	if err != nil {
		return err
	}
	c.XUI.Timeout = timeout

	insecure, err := lookupBool("XUI_INSECURE_SKIP_VERIFY", false)
	if err != nil {
		return err
	}
	c.XUI.InsecureSkipVerify = insecure
	return nil
}

// validate checks required configuration fields.
func (c *Config) validate() error {
	var missing []string

	if c.Telegram.Token == "" {
		missing = append(missing, "BOT_TOKEN / TELEGRAM_BOT_TOKEN")
	}
	if len(c.Telegram.AdminIDs) == 0 {
		missing = append(missing, "ADMIN_IDS")
	}
	if c.XUI.BaseURL == "" {
		missing = append(missing, "XUI_BASE_URL")
	}
	if c.XUI.Username == "" {
		missing = append(missing, "XUI_USERNAME")
	}
	if c.XUI.Password == "" {
		missing = append(missing, "XUI_PASSWORD")
	}
	if c.ServerHost == "" {
		missing = append(missing, "SERVER_HOST / REALITY_SERVER_IP")
	}
	if len(missing) > 0 {
		return fmt.Errorf("не заданы обязательные переменные окружения: %s", strings.Join(missing, ", "))
	}

	parsed, err := url.Parse(c.XUI.BaseURL)
	if err != nil {
		return fmt.Errorf("XUI_BASE_URL содержит некорректный URL %q: %w", c.XUI.BaseURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("XUI_BASE_URL должен начинаться с http:// или https://, получено %q", c.XUI.BaseURL)
	}
	if parsed.Host == "" {
		return fmt.Errorf("XUI_BASE_URL не содержит хост: %q", c.XUI.BaseURL)
	}
	return nil
}

// loadSSH loads optional SSH tunneling configuration parameters from environment variables.
func (c *Config) loadSSH() error {
	enabled, err := lookupBool("SSH_ENABLED", false)
	if err != nil {
		return err
	}
	c.SSH.Enabled = enabled
	c.SSH.Host = lookupString("SSH_HOST", "")
	port, err := lookupInt("SSH_PORT", 22)
	if err != nil {
		return err
	}
	c.SSH.Port = port
	c.SSH.User = lookupString("SSH_USER", "root")
	c.SSH.KeyPath = lookupString("SSH_KEY_PATH", "")
	c.SSH.Password = lookupString("SSH_PASSWORD", "")
	return nil
}

