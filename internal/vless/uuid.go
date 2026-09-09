package vless

import (
	"fmt"

	"github.com/google/uuid"
)

// NewUUID generates a random UUID version 4 according to RFC 4122.
func NewUUID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("генерация UUID: %w", err)
	}
	return id.String(), nil
}
