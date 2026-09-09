package xui

import (
	"encoding/json"
	"testing"
)

// TestDecodeObjectJSON tests decoding both direct JSON objects and legacy escaped JSON strings.
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

// TestParseInbound_ObjectFormat tests parsing inbound configuration in modern JSON object format.
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

// TestParseInbound_LegacyStringFormat tests parsing inbound configuration in legacy JSON string format.
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

// TestNewAddInboundRequest tests serializing the request payload for adding a new inbound.
func TestNewAddInboundRequest(t *testing.T) {
	spec := CreateInboundSpec{
		Remark:     "My-Test-Inbound",
		Port:       443,
		DestDomain: "apple.com",
		PrivateKey: "priv123",
		PublicKey:  "pub123",
		ShortID:    "short123",
	}

	req, err := newAddInboundRequest(spec)
	if err != nil {
		t.Fatalf("newAddInboundRequest() error = %v", err)
	}

	if req.Remark != "My-Test-Inbound" || req.Port != 443 || req.Protocol != "vless" {
		t.Errorf("unexpected request header: %+v", req)
	}

	var stream inboundStreamPayload
	if err := json.Unmarshal([]byte(req.StreamSettings), &stream); err != nil {
		t.Fatalf("unmarshal streamSettings error = %v", err)
	}

	if stream.Security != "reality" {
		t.Errorf("expected security reality, got %s", stream.Security)
	}
	if stream.RealitySettings.Dest != "apple.com:443" {
		t.Errorf("expected dest apple.com:443, got %s", stream.RealitySettings.Dest)
	}
	if stream.RealitySettings.PrivateKey != "priv123" || stream.RealitySettings.Settings.PublicKey != "pub123" {
		t.Errorf("unexpected keys in stream: %+v", stream.RealitySettings)
	}
}

// TestGenerateX25519Keys tests local X25519 key pair generation and short ID generation.
func TestGenerateX25519Keys(t *testing.T) {
	cert, err := GenerateX25519Keys()
	if err != nil {
		t.Fatalf("GenerateX25519Keys() error = %v", err)
	}
	if len(cert.PrivateKey) != 43 {
		t.Errorf("expected privateKey length 43, got %d (%s)", len(cert.PrivateKey), cert.PrivateKey)
	}
	if len(cert.PublicKey) != 43 {
		t.Errorf("expected publicKey length 43, got %d (%s)", len(cert.PublicKey), cert.PublicKey)
	}

	shortID := GenerateShortID()
	if len(shortID) != 16 {
		t.Errorf("expected shortId length 16, got %d (%s)", len(shortID), shortID)
	}
}
