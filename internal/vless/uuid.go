package vless

import (
	"fmt"

	"github.com/google/uuid"
)

// NewUUID генерирует случайный UUID версии 4 (RFC 4122).
func NewUUID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("генерация UUID: %w", err)
	}
	return id.String(), nil
}
