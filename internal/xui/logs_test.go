package xui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/login" {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "test"})
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true}`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/panel/api/server/logs") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true, "msg": "", "obj": "2026/09/10 xray started\n2026/09/10 accepted connection"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL:  server.URL,
		Username: "admin",
		Password: "password",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	logs, err := client.GetLogs(context.Background(), 50)
	if err != nil {
		t.Fatalf("expected logs, got error: %v", err)
	}
	if !strings.Contains(logs, "xray started") {
		t.Errorf("unexpected logs content: %s", logs)
	}
}
