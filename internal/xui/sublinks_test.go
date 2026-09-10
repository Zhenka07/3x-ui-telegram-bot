package xui

import (
	"encoding/json"
	"testing"
)

func TestExtractConnectionLinks_String(t *testing.T) {
	raw := json.RawMessage(`"vless://uuid@host:443?type=tcp#remark\nvmess://eyJ2IjoiMiJ9"`)
	links := extractConnectionLinks(raw)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d: %v", len(links), links)
	}
	if links[0] != "vless://uuid@host:443?type=tcp#remark" {
		t.Errorf("link[0] mismatch: %s", links[0])
	}
	if links[1] != "vmess://eyJ2IjoiMiJ9" {
		t.Errorf("link[1] mismatch: %s", links[1])
	}
}

func TestExtractConnectionLinks_Slice(t *testing.T) {
	raw := json.RawMessage(`["trojan://pass@host:443#tag", "ss://YWVzOjEyMw@host:1000#ss"]`)
	links := extractConnectionLinks(raw)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d: %v", len(links), links)
	}
}
