package vless

import (
	"strings"
	"testing"

	"github.com/zhenya/3x-ui-admin/internal/xui"
)

func TestBuildUniversalLink_VLESS_NonReality(t *testing.T) {
	ib := &xui.Inbound{
		ID:             1,
		Remark:         "Test-VLESS-TLS",
		Port:           443,
		Protocol:       "vless",
		StreamSettings: `{"network":"tcp","security":"tls","tlsSettings":{"serverName":"example.com"}}`,
	}
	client := &xui.Client{
		ID:    "11111111-2222-3333-4444-555555555555",
		Email: "user1",
		Flow:  "xtls-rprx-vision",
	}

	link, err := BuildUniversalLink("1.2.3.4", ib, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(link, "vless://11111111-2222-3333-4444-555555555555@1.2.3.4:443?") {
		t.Errorf("unexpected prefix in link: %s", link)
	}
	if !strings.Contains(link, "security=tls") {
		t.Errorf("missing security=tls: %s", link)
	}
	if !strings.Contains(link, "sni=example.com") {
		t.Errorf("missing sni=example.com: %s", link)
	}
}

func TestBuildUniversalLink_VMess(t *testing.T) {
	ib := &xui.Inbound{
		ID:       2,
		Remark:   "Test-VMess",
		Port:     10086,
		Protocol: "vmess",
	}
	client := &xui.Client{
		ID:    "uuid-vmess-test",
		Email: "user2",
	}

	link, err := BuildUniversalLink("myhost.com", ib, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(link, "vmess://") {
		t.Errorf("expected vmess link, got: %s", link)
	}
}

func TestBuildUniversalLink_Trojan(t *testing.T) {
	ib := &xui.Inbound{
		ID:       3,
		Remark:   "Test-Trojan",
		Port:     443,
		Protocol: "trojan",
	}
	client := &xui.Client{
		Password: "secretpassword",
		Email:    "user3",
	}

	link, err := BuildUniversalLink("myhost.com", ib, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(link, "trojan://secretpassword@myhost.com:443?") {
		t.Errorf("unexpected trojan link: %s", link)
	}
}
