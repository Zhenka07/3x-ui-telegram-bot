package xui

import (
	"encoding/json"
	"testing"
)

func TestDecodeObjectJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    json.RawMessage
		wantOK   bool
		expected string
	}{
		{
			name:     "empty input",
			input:    nil,
			wantOK:   false,
			expected: "",
		},
		{
			name:     "direct JSON object (3x-ui v3.7+)",
			input:    json.RawMessage(`{"clients":[{"id":"uuid-1","email":"test"}]}`),
			wantOK:   true,
			expected: `{"clients":[{"id":"uuid-1","email":"test"}]}`,
		},
		{
			name:     "escaped JSON string (legacy 3x-ui)",
			input:    json.RawMessage(`"{\"clients\":[{\"id\":\"uuid-1\",\"email\":\"test\"}]}"`),
			wantOK:   true,
			expected: `{"clients":[{"id":"uuid-1","email":"test"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := decodeObjectJSON(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("decodeObjectJSON() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && string(got) != tt.expected {
				t.Errorf("decodeObjectJSON() = %s, want %s", string(got), tt.expected)
			}
		})
	}
}

func TestParseInbound_ObjectFormat(t *testing.T) {
	raw := rawInbound{
		ID:             1,
		Remark:         "VLESS-Reality",
		Port:           443,
		Protocol:       "vless",
		Enable:         true,
		Up:             100,
		Down:           200,
		Total:          300,
		Settings:       json.RawMessage(`{"clients":[{"id":"1111-2222","email":"user1","flow":"xtls-rprx-vision"}]}`),
		StreamSettings: json.RawMessage(`{"security":"reality","realitySettings":{"serverNames":["example.com"],"shortIds":["sid1"],"settings":{"publicKey":"pbk123","fingerprint":"chrome","spiderX":"/test"}}}`),
	}

	ib := parseInbound(raw)
	if ib.ID != 1 || ib.Remark != "VLESS-Reality" || ib.Port != 443 {
		t.Errorf("unexpected inbound header: %+v", ib)
	}
	if len(ib.Clients) != 1 || ib.Clients[0].Email != "user1" {
		t.Errorf("failed to parse clients: %+v", ib.Clients)
	}
	if !ib.IsReality {
		t.Errorf("expected IsReality=true")
	}
	if ib.Reality.PublicKey != "pbk123" || len(ib.Reality.ServerNames) == 0 || ib.Reality.ServerNames[0] != "example.com" {
		t.Errorf("failed to parse reality settings: %+v", ib.Reality)
	}
}

func TestParseInbound_LegacyStringFormat(t *testing.T) {
	raw := rawInbound{
		ID:             2,
		Remark:         "VLESS-Legacy",
		Port:           443,
		Protocol:       "vless",
		Enable:         true,
		Settings:       json.RawMessage(`"{\"clients\":[{\"id\":\"3333-4444\",\"email\":\"user2\"}]}"`),
		StreamSettings: json.RawMessage(`"{\"security\":\"reality\",\"realitySettings\":{\"serverNames\":[\"legacy.com\"],\"shortIds\":[\"sid2\"],\"settings\":{\"publicKey\":\"pbk456\"}}}"`),
	}

	ib := parseInbound(raw)
	if len(ib.Clients) != 1 || ib.Clients[0].Email != "user2" {
		t.Errorf("failed to parse legacy clients: %+v", ib.Clients)
	}
	if !ib.IsReality {
		t.Errorf("expected IsReality=true")
	}
	if ib.Reality.PublicKey != "pbk456" {
		t.Errorf("failed to parse legacy reality settings: %+v", ib.Reality)
	}
}
