// Package main — точка входа сервиса Telegram-админ-панели 3x-ui.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/zhenya/3x-ui-admin/internal/bot"
	"github.com/zhenya/3x-ui-admin/internal/config"
	"github.com/zhenya/3x-ui-admin/internal/storage"
	"github.com/zhenya/3x-ui-admin/internal/xui"
)

func main() {
	if err := run(); err != nil {
		slog.Error("критическая ошибка при работе сервиса", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Загрузка конфигурации.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("загрузка конфигурации: %w", err)
	}

	// 2. Инициализация логгера.
	logger := setupLogger(cfg.LogLevel)
	logger.Info("запуск сервиса 3x-ui Admin Bot")

	// 3. Graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. Клиент 3x-ui API.
	logger.Info("инициализация клиента 3x-ui API",
		"base_url", cfg.XUI.BaseURL,
	)
	xuiClient, err := xui.New(xui.Config{
		BaseURL:            cfg.XUI.BaseURL,
		Username:           cfg.XUI.Username,
		Password:           cfg.XUI.Password,
		Timeout:            cfg.XUI.Timeout,
		InsecureSkipVerify: cfg.XUI.InsecureSkipVerify,
	}, logger)
	if err != nil {
		return fmt.Errorf("создание клиента 3x-ui: %w", err)
	}

	// 5. Инициализация хранилища аудита (SQLite без CGO).
	logger.Info("инициализация хранилища аудита", "db_path", cfg.Storage.DBPath)
	sqlDB, err := storage.Open(ctx, cfg.Storage.DBPath)
	if err != nil {
		return fmt.Errorf("открытие базы данных: %w", err)
	}
	defer sqlDB.Close()
	auditRepo := storage.NewAuditRepo(sqlDB)

	// 6. Инициализация Telegram-бота.
	logger.Info("инициализация Telegram-бота")
	tgBot, err := bot.New(*cfg, xuiClient, auditRepo, logger)
	if err != nil {
		return fmt.Errorf("создание Telegram-бота: %w", err)
	}

	// 7. Логирование сигнала остановки.
	go func() {
		<-ctx.Done()
		logger.Info("получен сигнал остановки (SIGINT/SIGTERM), завершаем работу...")
	}()

	// 7. Запуск (блокирующий).
	if err := tgBot.Start(ctx); err != nil {
		return fmt.Errorf("работа бота завершилась с ошибкой: %w", err)
	}

	logger.Info("сервис 3x-ui Admin Bot штатно остановлен")
	return nil
}

func setupLogger(levelStr string) *slog.Logger {
	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}
