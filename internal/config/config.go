// Package config отвечает за загрузку и валидацию конфигурации админ-бота.
//
// Источники значений (в порядке приоритета):
//  1. Переменные окружения процесса.
//  2. Файл .env (если присутствует рядом с бинарником или указан через ENV_FILE).
//
// Загрузка .env реализована на стандартной библиотеке, чтобы не тянуть
// лишних зависимостей: значения из файла НЕ перетирают уже заданные
// переменные окружения.
package config

import (
	"time"
)

// Config — корневая структура конфигурации приложения.
type Config struct {
	Telegram   TelegramConfig
	XUI        XUIConfig
	Storage    StorageConfig
	ServerHost string // IP/hostname сервера для сборки vless-ссылок
	LogLevel   string
}

// StorageConfig — параметры базы данных.
type StorageConfig struct {
	DBPath string
}

// TelegramConfig — параметры Telegram-бота.
type TelegramConfig struct {
	Token string
	// AdminIDs — список Telegram ID администраторов.
	// Бот реагирует ТОЛЬКО на запросы от этих пользователей.
	AdminIDs []int64
	// LongPollerTimeout — таймаут long polling.
	LongPollerTimeout time.Duration
}

// XUIConfig — параметры доступа к панели 3x-ui.
type XUIConfig struct {
	BaseURL  string
	Username string
	Password string
	Timeout  time.Duration
	// InsecureSkipVerify отключает проверку TLS-сертификата панели
	// (актуально для самоподписанных сертификатов).
	InsecureSkipVerify bool
}

// IsAdmin сообщает, входит ли переданный Telegram ID в список администраторов.
func (t TelegramConfig) IsAdmin(id int64) bool {
	for _, adminID := range t.AdminIDs {
		if adminID == id {
			return true
		}
	}
	return false
}
