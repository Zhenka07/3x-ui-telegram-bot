package xui

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// GenerateX25519Keys генерирует пару ключей x25519 локально на Go через crypto/ecdh.
// Возвращает ключи в unpadded base64 (RawURLEncoding), совместимом с Xray Reality.
func GenerateX25519Keys() (*X25519Cert, error) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("генерация ключей X25519: %w", err)
	}

	privBytes := priv.Bytes()
	pubBytes := priv.PublicKey().Bytes()

	return &X25519Cert{
		PrivateKey: base64.RawURLEncoding.EncodeToString(privBytes),
		PublicKey:  base64.RawURLEncoding.EncodeToString(pubBytes),
	}, nil
}

// GenerateShortID генерирует случайный shortId длиной 8 байт (16 hex-символов).
func GenerateShortID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback в маловероятном случае сбоя crypto/rand
		return "0123456789abcdef"
	}
	return hex.EncodeToString(b)
}
