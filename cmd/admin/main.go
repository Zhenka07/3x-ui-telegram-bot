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

// main is the entry point of the admin application.
func main() {
	if err := run(); err != nil {
		slog.Error("критическая ошибка при работе сервиса", "error", err)
		os.Exit(1)
	}
}

// run initializes and runs all components of the admin bot service.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("загрузка конфигурации: %w", err)
	}

	logger := setupLogger(cfg.LogLevel)
	logger.Info("запуск сервиса 3x-ui Admin Bot")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

	logger.Info("инициализация хранилища аудита", "db_path", cfg.Storage.DBPath)
	sqlDB, err := storage.Open(ctx, cfg.Storage.DBPath)
	if err != nil {
		return fmt.Errorf("открытие базы данных: %w", err)
	}
	defer sqlDB.Close()
	auditRepo := storage.NewAuditRepo(sqlDB)

	logger.Info("инициализация Telegram-бота")
	tgBot, err := bot.New(*cfg, xuiClient, auditRepo, logger)
	if err != nil {
		return fmt.Errorf("создание Telegram-бота: %w", err)
	}

	go func() {
		<-ctx.Done()
		logger.Info("получен сигнал остановки (SIGINT/SIGTERM), завершаем работу...")
	}()

	if err := tgBot.Start(ctx); err != nil {
		return fmt.Errorf("работа бота завершилась с ошибкой: %w", err)
	}

	logger.Info("сервис 3x-ui Admin Bot штатно остановлен")
	return nil
}

// setupLogger initializes and returns a slog.Logger configured with the given log level.
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
