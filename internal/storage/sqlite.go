// Package storage реализует persistence на SQLite без CGO (драйвер modernc.org/sqlite).
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB — обёртка над *sql.DB с прикладными настройками пула.
type DB struct {
	sql *sql.DB
}

// Open открывает (или создаёт) файл базы данных и применяет миграции.
func Open(ctx context.Context, path string) (*DB, error) {
	if path == "" {
		return nil, fmt.Errorf("путь к базе данных не задан")
	}

	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("создание каталога %q для базы данных: %w", dir, err)
		}
	}

	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)",
		path,
	)

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("открытие базы данных %q: %w", path, err)
	}

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("проверка соединения с базой данных: %w", err)
	}

	db := &DB{sql: sqlDB}
	if err := db.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// SQL возвращает низкоуровневое соединение.
func (d *DB) SQL() *sql.DB {
	return d.sql
}

// Close закрывает базу данных.
func (d *DB) Close() error {
	if d == nil || d.sql == nil {
		return nil
	}
	return d.sql.Close()
}

func (d *DB) migrate(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		admin_id INTEGER NOT NULL,
		action TEXT NOT NULL,
		inbound_id INTEGER NOT NULL,
		client_email TEXT NOT NULL,
		details TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
	`
	_, err := d.sql.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("миграция audit_logs: %w", err)
	}
	return nil
}
