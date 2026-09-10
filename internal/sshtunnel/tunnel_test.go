package sshtunnel

import (
	"testing"
)

func TestNew_Validation(t *testing.T) {
	// Missing host
	_, err := New(Config{})
	if err == nil {
		t.Error("expected error for empty host")
	}

	// Missing auth method
	_, err = New(Config{Host: "127.0.0.1"})
	if err == nil {
		t.Error("expected error for missing auth methods")
	}

	// Valid config with password
	tunnel, err := New(Config{
		Host:     "127.0.0.1",
		Password: "testpassword",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tunnel.cfg.Port != 22 {
		t.Errorf("expected default port 22, got %d", tunnel.cfg.Port)
	}
	if tunnel.cfg.User != "root" {
		t.Errorf("expected default user root, got %s", tunnel.cfg.User)
	}
}
