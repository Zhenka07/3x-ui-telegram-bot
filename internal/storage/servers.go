package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type ServerRecord struct {
	ID         string
	Name       string
	BaseURL    string
	Username   string
	Password   string
	ServerHost string
	IsActive   bool
	CreatedAt  time.Time
}

type ServerRepo struct {
	db *DB
}

func NewServerRepo(db *DB) *ServerRepo {
	return &ServerRepo{db: db}
}

// EnsureDefaultServer inserts the initial server from configuration if no servers exist.
func (r *ServerRepo) EnsureDefaultServer(ctx context.Context, s ServerRecord) error {
	var count int
	err := r.db.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM servers").Scan(&count)
	if err != nil {
		return fmt.Errorf("проверка наличия серверов: %w", err)
	}

	if count == 0 {
		s.IsActive = true
		s.CreatedAt = time.Now()
		return r.Add(ctx, s)
	}
	return nil
}

// List returns all configured servers.
func (r *ServerRepo) List(ctx context.Context) ([]ServerRecord, error) {
	query := `SELECT id, name, base_url, username, password, server_host, is_active, created_at FROM servers ORDER BY created_at ASC`
	rows, err := r.db.sql.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("запрос списка серверов: %w", err)
	}
	defer rows.Close()

	var result []ServerRecord
	for rows.Next() {
		var s ServerRecord
		var createdAtUnix int64
		var isActiveInt int
		if err := rows.Scan(&s.ID, &s.Name, &s.BaseURL, &s.Username, &s.Password, &s.ServerHost, &isActiveInt, &createdAtUnix); err != nil {
			return nil, fmt.Errorf("сканирование сервера: %w", err)
		}
		s.IsActive = isActiveInt == 1
		s.CreatedAt = time.Unix(createdAtUnix, 0).UTC()
		result = append(result, s)
	}
	return result, rows.Err()
}

// Get retrieves a server by ID.
func (r *ServerRepo) Get(ctx context.Context, id string) (*ServerRecord, error) {
	query := `SELECT id, name, base_url, username, password, server_host, is_active, created_at FROM servers WHERE id = ?`
	row := r.db.sql.QueryRowContext(ctx, query, id)

	var s ServerRecord
	var createdAtUnix int64
	var isActiveInt int
	if err := row.Scan(&s.ID, &s.Name, &s.BaseURL, &s.Username, &s.Password, &s.ServerHost, &isActiveInt, &createdAtUnix); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("получение сервера %q: %w", id, err)
	}
	s.IsActive = isActiveInt == 1
	s.CreatedAt = time.Unix(createdAtUnix, 0).UTC()
	return &s, nil
}

// GetActive retrieves the currently active server.
func (r *ServerRepo) GetActive(ctx context.Context) (*ServerRecord, error) {
	query := `SELECT id, name, base_url, username, password, server_host, is_active, created_at FROM servers WHERE is_active = 1 LIMIT 1`
	row := r.db.sql.QueryRowContext(ctx, query)

	var s ServerRecord
	var createdAtUnix int64
	var isActiveInt int
	if err := row.Scan(&s.ID, &s.Name, &s.BaseURL, &s.Username, &s.Password, &s.ServerHost, &isActiveInt, &createdAtUnix); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("получение активного сервера: %w", err)
	}
	s.IsActive = isActiveInt == 1
	s.CreatedAt = time.Unix(createdAtUnix, 0).UTC()
	return &s, nil
}

// SetActive marks the specified server as active and all others as inactive.
func (r *ServerRepo) SetActive(ctx context.Context, id string) error {
	tx, err := r.db.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("начало транзакции переключения сервера: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "UPDATE servers SET is_active = 0"); err != nil {
		return fmt.Errorf("сброс активного сервера: %w", err)
	}

	res, err := tx.ExecContext(ctx, "UPDATE servers SET is_active = 1 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("активация сервера %q: %w", id, err)
	}

	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("сервер %q не найден", id)
	}

	return tx.Commit()
}

// Add adds a new server to the registry.
func (r *ServerRepo) Add(ctx context.Context, s ServerRecord) error {
	query := `INSERT INTO servers (id, name, base_url, username, password, server_host, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	isActiveInt := 0
	if s.IsActive {
		isActiveInt = 1
	}
	created := s.CreatedAt.Unix()
	if created == 0 {
		created = time.Now().Unix()
	}

	_, err := r.db.sql.ExecContext(ctx, query, s.ID, s.Name, s.BaseURL, s.Username, s.Password, s.ServerHost, isActiveInt, created)
	if err != nil {
		return fmt.Errorf("добавление сервера %q: %w", s.Name, err)
	}
	return nil
}

// Delete removes a server by ID.
func (r *ServerRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.sql.ExecContext(ctx, "DELETE FROM servers WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("удаление сервера %q: %w", id, err)
	}
	return nil
}
