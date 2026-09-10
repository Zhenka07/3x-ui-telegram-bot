package config

import (
	"time"
)

type Config struct {
	Telegram   TelegramConfig
	XUI        XUIConfig
	Storage    StorageConfig
	SSH        SSHConfig
	ServerHost string
	LogLevel   string
}

type SSHConfig struct {
	Enabled  bool
	Host     string
	Port     int
	User     string
	KeyPath  string
	Password string
}

type StorageConfig struct {
	DBPath string
}

type TelegramConfig struct {
	Token             string
	AdminIDs          []int64
	LongPollerTimeout time.Duration
}

type XUIConfig struct {
	BaseURL            string
	Username           string
	Password           string
	Timeout            time.Duration
	InsecureSkipVerify bool
}

// IsAdmin reports whether the given Telegram ID is in the admin list.
func (t TelegramConfig) IsAdmin(id int64) bool {
	for _, adminID := range t.AdminIDs {
		if adminID == id {
			return true
		}
	}
	return false
}
