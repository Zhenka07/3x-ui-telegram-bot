package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhenya/3x-ui-admin/internal/bot"
	"github.com/zhenya/3x-ui-admin/internal/config"
	"github.com/zhenya/3x-ui-admin/internal/observability"
	"github.com/zhenya/3x-ui-admin/internal/sshtunnel"
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

	logger := observability.NewLogger(cfg.Observability)
	slog.SetDefault(logger)
	logger.Info("запуск сервиса 3x-ui Admin Bot")

	tracer, shutdownTracer, err := observability.InitTracer(context.Background(), cfg.Observability)
	if err != nil {
		return fmt.Errorf("инициализация трассировки: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracer(shutdownCtx); err != nil {
			logger.Error("ошибка при остановке провайдера трассировки", "error", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var dialContext func(ctx context.Context, network, addr string) (net.Conn, error)
	var tunnel *sshtunnel.Tunnel
	if cfg.SSH.Enabled {
		logger.Info("инициализация SSH-туннеля для 3x-ui", "host", cfg.SSH.Host, "user", cfg.SSH.User)
		var err error
		tunnel, err = sshtunnel.New(sshtunnel.Config{
			Host:     cfg.SSH.Host,
			Port:     cfg.SSH.Port,
			User:     cfg.SSH.User,
			KeyPath:  cfg.SSH.KeyPath,
			Password: cfg.SSH.Password,
			Timeout:  cfg.XUI.Timeout,
		})
		if err != nil {
			return fmt.Errorf("создание SSH-туннеля: %w", err)
		}
		defer tunnel.Close()
		dialContext = tunnel.DialContext
	}

	logger.Info("инициализация клиента 3x-ui API",
		"base_url", cfg.XUI.BaseURL,
	)
	xuiClient, err := xui.New(xui.Config{
		BaseURL:            cfg.XUI.BaseURL,
		Username:           cfg.XUI.Username,
		Password:           cfg.XUI.Password,
		Timeout:            cfg.XUI.Timeout,
		InsecureSkipVerify: cfg.XUI.InsecureSkipVerify,
		DialContext:        dialContext,
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
	serverRepo := storage.NewServerRepo(sqlDB)

	if err := serverRepo.EnsureDefaultServer(ctx, storage.ServerRecord{
		ID:         "default",
		Name:       "Основной сервер",
		BaseURL:    cfg.XUI.BaseURL,
		Username:   cfg.XUI.Username,
		Password:   cfg.XUI.Password,
		ServerHost: cfg.ServerHost,
	}); err != nil {
		return fmt.Errorf("инициализация списка серверов: %w", err)
	}

	logger.Info("инициализация Telegram-бота")
	tgBot, err := bot.New(*cfg, xuiClient, auditRepo, serverRepo, logger, tracer)
	if err != nil {
		return fmt.Errorf("создание Telegram-бота: %w", err)
	}
	if tunnel != nil {
		tgBot.SetTunnel(tunnel)
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
