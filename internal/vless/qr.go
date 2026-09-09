package vless

import (
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

const (
	defaultQRSize = 512
	maxQRSize     = 2048
)

// GenerateQR generates a PNG image of a QR code for the provided content in memory.
func GenerateQR(content string, size int) ([]byte, error) {
	if content == "" {
		return nil, fmt.Errorf("генерация QR-кода: пустое содержимое")
	}
	if size <= 0 {
		size = defaultQRSize
	}
	if size > maxQRSize {
		size = maxQRSize
	}

	png, err := qrcode.Encode(content, qrcode.Medium, size)
	if err != nil {
		return nil, fmt.Errorf("генерация QR-кода: %w", err)
	}
	return png, nil
}
