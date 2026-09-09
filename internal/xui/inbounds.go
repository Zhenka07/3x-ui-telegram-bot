package xui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ListInbounds retrieves all inbounds with parsed clients and Reality parameters from 3x-ui.
func (c *APIClient) ListInbounds(ctx context.Context) ([]Inbound, error) {
	resp, err := c.doJSON(ctx, "GET", "/panel/api/inbounds/list", nil)
	if err != nil {
		return nil, fmt.Errorf("получение списка инбаундов: %w", err)
	}

	var raw []rawInbound
	if err := json.Unmarshal(resp.Obj, &raw); err != nil {
		return nil, fmt.Errorf("разбор списка инбаундов: %w", err)
	}

	inbounds := make([]Inbound, 0, len(raw))
	for _, r := range raw {
		ib := parseInbound(r)
		inbounds = append(inbounds, ib)
	}
	return inbounds, nil
}

// GetInbound retrieves a single inbound by its ID.
func (c *APIClient) GetInbound(ctx context.Context, id int) (*Inbound, error) {
	path := fmt.Sprintf("/panel/api/inbounds/get/%d", id)
	resp, err := c.doJSON(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("получение инбаунда %d: %w", id, err)
	}

	var raw rawInbound
	if err := json.Unmarshal(resp.Obj, &raw); err != nil {
		return nil, fmt.Errorf("разбор инбаунда %d: %w", id, err)
	}

	ib := parseInbound(raw)
	return &ib, nil
}

// GetServerStatus retrieves system load metrics and Xray core status.
func (c *APIClient) GetServerStatus(ctx context.Context) (*ServerStatus, error) {
	endpoints := [][2]string{
		{"GET", "/panel/api/server/status"},
		{"POST", "/server/status"},
		{"GET", "/server/status"},
	}

	var resp *apiResponse
	var err error
	for _, ep := range endpoints {
		resp, err = c.doJSON(ctx, ep[0], ep[1], nil)
		if err == nil {
			break
		}
		if errors.Is(err, ErrNotFound) {
			err = nil
			continue
		}
		break
	}
	if err != nil {
		return nil, fmt.Errorf("получение статуса сервера: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("получение статуса сервера: %w", ErrNotFound)
	}

	var status ServerStatus
	if err := json.Unmarshal(resp.Obj, &status); err != nil {
		return nil, fmt.Errorf("разбор статуса сервера: %w", err)
	}
	return &status, nil
}

// parseInbound converts a raw inbound DTO into an Inbound struct.
func parseInbound(raw rawInbound) Inbound {
	ib := Inbound{
		ID:       raw.ID,
		Remark:   raw.Remark,
		Port:     raw.Port,
		Protocol: raw.Protocol,
		Enable:   raw.Enable,
		Up:       raw.Up,
		Down:     raw.Down,
		Total:    raw.Total,
		Tag:      raw.Tag,
	}

	if len(raw.Settings) > 0 {
		if data, ok := decodeObjectJSON(raw.Settings); ok {
			var settings inboundSettings
			if err := json.Unmarshal(data, &settings); err == nil {
				ib.Clients = settings.Clients
			}
		}
	}

	if len(raw.StreamSettings) > 0 {
		if data, ok := decodeObjectJSON(raw.StreamSettings); ok {
			var ss streamSettingsJSON
			if err := json.Unmarshal(data, &ss); err == nil {
				if ss.Security == "reality" && ss.RealitySettings != nil {
					ib.IsReality = true
					ib.Reality = RealityInfo{
						PublicKey:   ss.RealitySettings.Settings.PublicKey,
						ServerNames: ss.RealitySettings.ServerNames,
						ShortIDs:    ss.RealitySettings.ShortIds,
						Fingerprint: ss.RealitySettings.Settings.Fingerprint,
						SpiderX:     ss.RealitySettings.Settings.SpiderX,
						Dest:        ss.RealitySettings.Dest,
					}
				}
			}
		}
	}

	return ib
}

// decodeObjectJSON unmarshals raw JSON into a byte slice, handling both direct objects and nested JSON strings.
func decodeObjectJSON(raw json.RawMessage) ([]byte, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, false
	}

	if trimmed[0] == '{' || trimmed[0] == '[' {
		return trimmed, true
	}

	var nested string
	if err := json.Unmarshal(trimmed, &nested); err != nil || strings.TrimSpace(nested) == "" {
		return nil, false
	}
	return []byte(nested), true
}

// FindClient searches for a client by email within the inbound.
func (ib *Inbound) FindClient(email string) *Client {
	for i := range ib.Clients {
		if ib.Clients[i].Email == email {
			return &ib.Clients[i]
		}
	}
	return nil
}

// FindClientByUUID searches for a client by UUID within the inbound.
func (ib *Inbound) FindClientByUUID(uuid string) *Client {
	for i := range ib.Clients {
		if ib.Clients[i].ID == uuid {
			return &ib.Clients[i]
		}
	}
	return nil
}

// GetNewX25519Cert retrieves a new X25519 Reality key pair from 3x-ui or generates one locally.
func (c *APIClient) GetNewX25519Cert(ctx context.Context) (*X25519Cert, error) {
	endpoints := [][2]string{
		{"GET", "/panel/api/server/getNewX25519Cert"},
		{"GET", "/server/getNewX25519Cert"},
		{"POST", "/server/getNewX25519Cert"},
	}

	for _, ep := range endpoints {
		resp, err := c.doJSON(ctx, ep[0], ep[1], nil)
		if err == nil && resp != nil && len(resp.Obj) > 0 {
			var cert X25519Cert
			if err := json.Unmarshal(resp.Obj, &cert); err == nil && cert.PrivateKey != "" && cert.PublicKey != "" {
				c.log.Debug("3x-ui: получены X25519 ключи от панели", "endpoint", ep[1])
				return &cert, nil
			}
		}
	}

	c.log.Warn("3x-ui: API генерации X25519 недоступен, генерируем ключи локально")
	return GenerateX25519Keys()
}

// AddInbound creates a new VLESS-Reality inbound in the 3x-ui panel.
func (c *APIClient) AddInbound(ctx context.Context, spec CreateInboundSpec) (*Inbound, error) {
	payload, err := newAddInboundRequest(spec)
	if err != nil {
		return nil, fmt.Errorf("подготовка запроса создания инбаунда: %w", err)
	}

	resp, err := c.doJSON(ctx, "POST", "/panel/api/inbounds/add", payload)
	if err != nil {
		return nil, fmt.Errorf("создание инбаунда %q: %w", spec.Remark, err)
	}

	c.log.Debug("3x-ui: инбаунд успешно создан", "remark", spec.Remark, "port", spec.Port)

	var raw rawInbound
	if err := json.Unmarshal(resp.Obj, &raw); err == nil && raw.ID > 0 {
		ib := parseInbound(raw)
		return &ib, nil
	}

	inbounds, err := c.ListInbounds(ctx)
	if err == nil {
		for _, ib := range inbounds {
			if ib.Port == spec.Port && ib.Remark == spec.Remark {
				return &ib, nil
			}
		}
	}

	return &Inbound{
		Remark:    spec.Remark,
		Port:      spec.Port,
		Protocol:  "vless",
		Enable:    true,
		IsReality: true,
	}, nil
}
