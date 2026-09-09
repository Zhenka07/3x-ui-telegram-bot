package vless

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/zhenya/3x-ui-admin/internal/xui"
)

func TestBuildURI_ExactFormat(t *testing.T) {
	builder, err := NewBuilder(Params{
		ServerAddr:  "198.51.100.1",
		Port:        443,
		PublicKey:   "W8Z3s9Y8cQW9...",
		Fingerprint: "chrome",
		SNI:         "gateway.icloud.com",
		ShortID:     "1234abcd",
		SpiderX:     "/",
		Flow:        "xtls-rprx-vision",
		Tag:         "MyVPN",
	})
	if err != nil {
		t.Fatalf("неожиданная ошибка создания билдера: %v", err)
	}

	uuid := "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"
	uri, err := builder.BuildURI(uuid, "")
	if err != nil {
		t.Fatalf("ошибка сборки URI: %v", err)
	}

	expected := "vless://a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d@198.51.100.1:443?type=tcp&security=reality&pbk=W8Z3s9Y8cQW9...&fp=chrome&sni=gateway.icloud.com&sid=1234abcd&spx=%2F&flow=xtls-rprx-vision#MyVPN"
	if uri != expected {
		t.Errorf("\ngot:  %s\nwant: %s", uri, expected)
	}
}

func TestParseInboundReality(t *testing.T) {
	ib := &xui.Inbound{
		ID:        1,
		Remark:    "TestReality",
		Port:      8443,
		IsReality: true,
		Reality: xui.RealityInfo{
			PublicKey:   "pubKeyVal",
			ServerNames: []string{"sni.example.com"},
			ShortIDs:    []string{"sid123"},
			Fingerprint: "chrome",
			SpiderX:     "/path",
		},
	}

	params, err := ParseInboundReality("1.2.3.4", ib, "")
	if err != nil {
		t.Fatalf("ParseInboundReality error: %v", err)
	}
	if params.ServerAddr != "1.2.3.4" || params.Port != 8443 {
		t.Errorf("unexpected addr/port: %s:%d", params.ServerAddr, params.Port)
	}
	if params.PublicKey != "pubKeyVal" || params.SNI != "sni.example.com" || params.ShortID != "sid123" {
		t.Errorf("unexpected reality params: %+v", params)
	}
	if params.Flow != "xtls-rprx-vision" {
		t.Errorf("expected default flow xtls-rprx-vision, got %s", params.Flow)
	}
}

func TestGenerateQR(t *testing.T) {
	pngData, err := GenerateQR("vless://test", 256)
	if err != nil {
		t.Fatalf("ошибка генерации QR: %v", err)
	}

	pngHeader := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	if !bytes.HasPrefix(pngData, pngHeader) {
		t.Errorf("результат не является валидным PNG изображением")
	}
}

func TestNewUUID(t *testing.T) {
	id, err := NewUUID()
	if err != nil {
		t.Fatalf("ошибка генерации UUID: %v", err)
	}

	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(id) {
		t.Errorf("некорректный UUIDv4: %s", id)
	}
}
