package vless

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhenya/3x-ui-admin/internal/xui"
)

// streamSettingsRaw представляет структуру streamSettings из 3x-ui.
type streamSettingsRaw struct {
	Network         string               `json:"network"`
	Security        string               `json:"security"`
	RealitySettings *realitySettingsRaw `json:"realitySettings"`
}

type realitySettingsRaw struct {
	Show        bool               `json:"show"`
	Dest        string             `json:"dest"`
	ServerNames []string           `json:"serverNames"`
	PrivateKey  string             `json:"privateKey"`
	ShortIds    []string           `json:"shortIds"`
	Settings    realitySubSettings `json:"settings"`
}

type realitySubSettings struct {
	PublicKey   string `json:"publicKey"`
	Fingerprint string `json:"fingerprint"`
	SpiderX     string `json:"spiderX"`
}

// ParseRealityStreamSettings извлекает параметры Reality из сырой строки streamSettings.
func ParseRealityStreamSettings(rawJSON, serverAddr string, port int, tag, flow string) (*Params, error) {
	if strings.TrimSpace(rawJSON) == "" {
		return nil, fmt.Errorf("пустые streamSettings")
	}

	var ss streamSettingsRaw
	if err := json.Unmarshal([]byte(rawJSON), &ss); err != nil {
		return nil, fmt.Errorf("разбор streamSettings: %w", err)
	}

	if ss.Security != "reality" || ss.RealitySettings == nil {
		return nil, fmt.Errorf("инбаунд не использует протокол reality (security=%q)", ss.Security)
	}

	rs := ss.RealitySettings
	sni := ""
	if len(rs.ServerNames) > 0 {
		sni = rs.ServerNames[0]
	}

	shortID := ""
	if len(rs.ShortIds) > 0 {
		shortID = rs.ShortIds[0]
	}

	fp := rs.Settings.Fingerprint
	if fp == "" {
		fp = "chrome"
	}

	spx := rs.Settings.SpiderX
	if spx == "" {
		spx = "/"
	}

	if flow == "" {
		flow = "xtls-rprx-vision"
	}

	params := &Params{
		ServerAddr:  serverAddr,
		Port:        port,
		PublicKey:   rs.Settings.PublicKey,
		Fingerprint: fp,
		SNI:         sni,
		ShortID:     shortID,
		SpiderX:     spx,
		Flow:        flow,
		Tag:         tag,
	}

	return params, nil
}

// ParseInboundReality извлекает параметры Reality из структуры xui.Inbound.
func ParseInboundReality(serverAddr string, ib *xui.Inbound, flow string) (*Params, error) {
	if ib == nil {
		return nil, fmt.Errorf("inbound is nil")
	}
	if !ib.IsReality {
		return nil, fmt.Errorf("инбаунд %d (%s) не является Reality", ib.ID, ib.Remark)
	}

	sni := ""
	if len(ib.Reality.ServerNames) > 0 {
		sni = ib.Reality.ServerNames[0]
	}

	shortID := ""
	if len(ib.Reality.ShortIDs) > 0 {
		shortID = ib.Reality.ShortIDs[0]
	}

	fp := ib.Reality.Fingerprint
	if fp == "" {
		fp = "chrome"
	}

	spx := ib.Reality.SpiderX
	if spx == "" {
		spx = "/"
	}

	if flow == "" {
		flow = "xtls-rprx-vision"
	}

	params := &Params{
		ServerAddr:  serverAddr,
		Port:        ib.Port,
		PublicKey:   ib.Reality.PublicKey,
		Fingerprint: fp,
		SNI:         sni,
		ShortID:     shortID,
		SpiderX:     spx,
		Flow:        flow,
		Tag:         ib.Remark,
	}

	return params, nil
}
