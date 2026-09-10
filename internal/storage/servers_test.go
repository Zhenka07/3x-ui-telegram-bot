package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestServerRepo(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	repo := NewServerRepo(db)

	defaultServer := ServerRecord{
		ID:         "srv-1",
		Name:       "Primary Server",
		BaseURL:    "http://127.0.0.1:2053",
		Username:   "admin",
		Password:   "admin",
		ServerHost: "127.0.0.1",
	}

	if err := repo.EnsureDefaultServer(ctx, defaultServer); err != nil {
		t.Fatalf("EnsureDefaultServer failed: %v", err)
	}

	active, err := repo.GetActive(ctx)
	if err != nil || active == nil {
		t.Fatalf("GetActive failed: %v, got %v", err, active)
	}
	if active.ID != "srv-1" {
		t.Errorf("expected active srv-1, got %s", active.ID)
	}

	srv2 := ServerRecord{
		ID:         "srv-2",
		Name:       "Secondary Server",
		BaseURL:    "http://1.2.3.4:2053",
		Username:   "admin2",
		Password:   "pass2",
		ServerHost: "1.2.3.4",
		CreatedAt:  time.Now(),
	}
	if err := repo.Add(ctx, srv2); err != nil {
		t.Fatalf("Add srv2 failed: %v", err)
	}

	list, err := repo.List(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 servers, got %d (err: %v)", len(list), err)
	}

	if err := repo.SetActive(ctx, "srv-2"); err != nil {
		t.Fatalf("SetActive srv-2 failed: %v", err)
	}

	newActive, err := repo.GetActive(ctx)
	if err != nil || newActive == nil {
		t.Fatalf("GetActive after switch failed: %v", err)
	}
	if newActive.ID != "srv-2" {
		t.Errorf("expected active srv-2, got %s", newActive.ID)
	}
}
