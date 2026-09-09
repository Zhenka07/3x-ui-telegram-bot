package xui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ListInbounds возвращает все инбаунды панели с разобранными клиентами
// и Reality-параметрами.
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

// GetInbound возвращает один инбаунд по его ID.
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

// GetServerStatus возвращает информацию о нагрузке сервера и статусе Xray.
//
// Эндпоинт статуса различается между версиями панели:
//   - старые (v2.x и ниже): GET/POST /server/status
//   - новые (3.x, например 3.7.0): GET /panel/api/server/status
//
// Пробуем все варианты по очереди, игнорируя 404.
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
		// Иная ошибка (сеть, авторизация, панель) — прекращаем попытки.
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

// parseInbound конвертирует сырой DTO инбаунда в публичную структуру,
// парся вложенные JSON-строки settings и streamSettings.
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

	// Парсим клиентов из поля settings.
	if len(raw.Settings) > 0 {
		if data, ok := decodeObjectJSON(raw.Settings); ok {
			var settings inboundSettings
			if err := json.Unmarshal(data, &settings); err == nil {
				ib.Clients = settings.Clients
			}
		}
	}

	// Парсим Reality-параметры из поля streamSettings.
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

// decodeObjectJSON принимает json.RawMessage, который в разных версиях панели
// может быть либо вложенным JSON-объектом, либо JSON-строкой с экранированным
// JSON внутри. Возвращает байты, которые можно разобрать как объект.
func decodeObjectJSON(raw json.RawMessage) ([]byte, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, false
	}

	// Если прислали объект/массив напрямую — используем как есть.
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return trimmed, true
	}

	// Если прислали JSON-строку — разворачиваем её во вложенный JSON.
	var nested string
	if err := json.Unmarshal(trimmed, &nested); err != nil || strings.TrimSpace(nested) == "" {
		return nil, false
	}
	return []byte(nested), true
}

// FindClient ищет клиента по email в инбаунде.
// Возвращает nil, если клиент не найден.
func (ib *Inbound) FindClient(email string) *Client {
	for i := range ib.Clients {
		if ib.Clients[i].Email == email {
			return &ib.Clients[i]
		}
	}
	return nil
}

// FindClientByUUID ищет клиента по UUID в инбаунде.
func (ib *Inbound) FindClientByUUID(uuid string) *Client {
	for i := range ib.Clients {
		if ib.Clients[i].ID == uuid {
			return &ib.Clients[i]
		}
	}
	return nil
}
