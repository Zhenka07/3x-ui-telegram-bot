package xui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// AddClient добавляет нового клиента в указанный инбаунд.
func (c *APIClient) AddClient(ctx context.Context, inboundID int, spec ClientSpec) error {
	if err := spec.validate(); err != nil {
		return fmt.Errorf("добавление клиента в инбаунд %d: %w", inboundID, err)
	}

	// 1. Пробуем v2 API (3x-ui v2.4+).
	type v2Req struct {
		Client struct {
			ID         string `json:"id"`
			Email      string `json:"email"`
			Flow       string `json:"flow"`
			LimitIP    int    `json:"limitIp"`
			TotalGB    int64  `json:"totalGB"`
			ExpiryTime int64  `json:"expiryTime"`
			Enable     bool   `json:"enable"`
			TgID       int64  `json:"tgId,omitempty"`
		} `json:"client"`
		InboundIDs []int `json:"inboundIds"`
	}
	v2 := v2Req{InboundIDs: []int{inboundID}}
	v2.Client.ID = spec.UUID
	v2.Client.Email = spec.Email
	v2.Client.Flow = spec.Flow
	v2.Client.LimitIP = spec.LimitIP
	v2.Client.TotalGB = spec.TrafficLimitBytes
	v2.Client.ExpiryTime = timeToMS(spec.ExpiryAt)
	v2.Client.Enable = spec.Enable
	v2.Client.TgID = spec.TelegramID

	_, err := c.doJSON(ctx, "POST", "/panel/api/clients/add", v2)
	if err == nil {
		c.log.Debug("3x-ui: клиент добавлен (v2 clients/add)", "email", spec.Email, "inbound", inboundID)
		return nil
	}
	if !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("добавление клиента %q в инбаунд %d: %w", spec.Email, inboundID, err)
	}

	// 2. Фолбэк на legacy API.
	payload, err := newAddClientRequest(inboundID, spec.toClient())
	if err != nil {
		return fmt.Errorf("добавление клиента %q: %w", spec.Email, err)
	}
	if _, err := c.doJSON(ctx, "POST", "/panel/api/inbounds/addClient", payload); err != nil {
		return fmt.Errorf("добавление клиента %q в инбаунд %d: %w", spec.Email, inboundID, err)
	}

	c.log.Debug("3x-ui: клиент добавлен (legacy)", "email", spec.Email, "inbound", inboundID)
	return nil
}

// GetClientTraffics возвращает статистику трафика клиента по его email.
func (c *APIClient) GetClientTraffics(ctx context.Context, email string) (*ClientTraffic, error) {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return nil, fmt.Errorf("получение трафика: не задан email клиента")
	}

	pathV2 := "/panel/api/clients/traffic/" + url.PathEscape(trimmed)
	resp, err := c.doJSON(ctx, "GET", pathV2, nil)
	if err != nil && errors.Is(err, ErrNotFound) {
		pathLegacy := "/panel/api/inbounds/getClientTraffics/" + url.PathEscape(trimmed)
		resp, err = c.doJSON(ctx, "GET", pathLegacy, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("получение трафика клиента %q: %w", trimmed, err)
	}

	trimmedObj := bytes.TrimSpace(resp.Obj)
	if len(trimmedObj) == 0 || string(trimmedObj) == "null" {
		return nil, fmt.Errorf("получение трафика клиента %q: %w", trimmed, ErrClientNotFound)
	}

	var dto clientTrafficDTO
	if len(trimmedObj) > 0 && trimmedObj[0] == '[' {
		var list []clientTrafficDTO
		if err := json.Unmarshal(trimmedObj, &list); err != nil {
			return nil, fmt.Errorf("разбор статистики клиента %q: %w", trimmed, err)
		}
		if len(list) == 0 {
			return nil, fmt.Errorf("получение трафика клиента %q: %w", trimmed, ErrClientNotFound)
		}
		dto = list[0]
	} else {
		if err := json.Unmarshal(trimmedObj, &dto); err != nil {
			return nil, fmt.Errorf("разбор статистики клиента %q: %w", trimmed, err)
		}
	}

	return &ClientTraffic{
		Email:    dto.Email,
		Up:       dto.Up,
		Down:     dto.Down,
		Total:    dto.Total,
		Enable:   dto.Enable,
		ExpiryAt: msToTime(dto.ExpiryTime),
	}, nil
}

// DeleteClient удаляет клиента из инбаунда по его UUID и email.
//
// Эндпоинты различаются между версиями панели:
//   - новые (3.x): POST /panel/api/clients/del/{email}?keepTraffic=0
//   - старые (v2.x): POST /panel/api/clients/del/{uuid}?keepTraffic=0,
//     а также legacy POST /panel/api/inbounds/{id}/delClient/{uuid}.
func (c *APIClient) DeleteClient(ctx context.Context, inboundID int, uuid, email string) error {
	trimmedUUID := strings.TrimSpace(uuid)
	trimmedEmail := strings.TrimSpace(email)
	if trimmedUUID == "" && trimmedEmail == "" {
		return fmt.Errorf("удаление клиента: не задан идентификатор")
	}

	var lastErr error
	attempt := func(id string) error {
		pathV2 := fmt.Sprintf("/panel/api/clients/del/%s?keepTraffic=0", url.PathEscape(id))
		_, err := c.doJSON(ctx, "POST", pathV2, nil)
		return err
	}

	// 1. Пробуем v2 API по UUID (старые панели 2.x).
	if trimmedUUID != "" {
		if err := attempt(trimmedUUID); err == nil {
			c.log.Debug("3x-ui: клиент удалён (v2 uuid)", "uuid", trimmedUUID)
			return nil
		} else {
			lastErr = err
		}
	}

	// 2. Пробуем v2 API по email (новые панели 3.x).
	if trimmedEmail != "" {
		if err := attempt(trimmedEmail); err == nil {
			c.log.Debug("3x-ui: клиент удалён (v2 email)", "email", trimmedEmail)
			return nil
		} else {
			lastErr = err
		}
	}

	// 3. Фолбэк на legacy API по UUID.
	path := fmt.Sprintf("/panel/api/inbounds/%d/delClient/%s", inboundID, url.PathEscape(trimmedUUID))
	if _, err := c.doJSON(ctx, "POST", path, nil); err != nil {
		fallbackPath := fmt.Sprintf("/panel/api/inbounds/delClient/%s", url.PathEscape(trimmedUUID))
		if _, fallbackErr := c.doJSON(ctx, "POST", fallbackPath, nil); fallbackErr != nil {
			if lastErr != nil {
				return fmt.Errorf("удаление клиента %q из инбаунда %d: %w", trimmedEmail, inboundID, lastErr)
			}
			return fmt.Errorf("удаление клиента %q из инбаунда %d: %w", trimmedUUID, inboundID, err)
		}
	}

	c.log.Debug("3x-ui: клиент удалён (legacy)", "uuid", trimmedUUID, "inbound", inboundID)
	return nil
}

// UpdateClient обновляет параметры существующего клиента.
//
// Эндпоинт и тела запроса различаются между версиями панели:
//   - новые (3.x): POST /panel/api/clients/update/{email} с полями клиента
//     в теле на верхнем уровне (JSON).
//   - старые (v2.x): POST /panel/api/inbounds/updateClient/{uuid} с телом
//     {"id": <inboundId>, "settings": "<JSON-строка>"}.
//
// Пробуем v2 первым, при 404/несовпадении контракта — фолбэк на legacy.
func (c *APIClient) UpdateClient(ctx context.Context, inboundID int, spec ClientSpec) error {
	if err := spec.validate(); err != nil {
		return fmt.Errorf("обновление клиента в инбаунде %d: %w", inboundID, err)
	}

	// 1. Пробуем v2 API панелей 3.x.
	type v2Body struct {
		ID         string `json:"id"`
		Email      string `json:"email"`
		Flow       string `json:"flow"`
		LimitIP    int    `json:"limitIp"`
		TotalGB    int64  `json:"totalGB"`
		ExpiryTime int64  `json:"expiryTime"`
		Enable     bool   `json:"enable"`
		TgID       int64  `json:"tgId,omitempty"`
		SubID      string `json:"subId,omitempty"`
	}
	body := v2Body{
		ID:         spec.UUID,
		Email:      spec.Email,
		Flow:       spec.Flow,
		LimitIP:    spec.LimitIP,
		TotalGB:    spec.TrafficLimitBytes,
		ExpiryTime: timeToMS(spec.ExpiryAt),
		Enable:     spec.Enable,
		TgID:       spec.TelegramID,
		SubID:      spec.SubID,
	}
	pathV2 := "/panel/api/clients/update/" + url.PathEscape(spec.Email) + "?inboundIds=" + strconv.Itoa(inboundID)
	if _, err := c.doJSON(ctx, "POST", pathV2, body); err == nil {
		c.log.Debug("3x-ui: клиент обновлён (v2)", "email", spec.Email, "inbound", inboundID)
		return nil
	}

	// 2. Фолбэк на legacy API.
	payload, err := newAddClientRequest(inboundID, spec.toClient())
	if err != nil {
		return fmt.Errorf("обновление клиента %q: %w", spec.Email, err)
	}

	pathLegacy := "/panel/api/inbounds/updateClient/" + url.PathEscape(spec.UUID)
	if _, err := c.doJSON(ctx, "POST", pathLegacy, payload); err != nil {
		return fmt.Errorf("обновление клиента %q в инбаунде %d: %w", spec.Email, inboundID, err)
	}

	c.log.Debug("3x-ui: клиент обновлён (legacy)", "email", spec.Email, "inbound", inboundID)
	return nil
}

// ResetClientTraffic сбрасывает счётчик трафика клиента.
//
// В новых версиях панели (3.x) сброс выполняется по v2-эндпоинту
// POST /panel/api/clients/resetTraffic/{email}, в старых —
// POST /panel/api/inbounds/{id}/resetClientTraffic/{email}.
func (c *APIClient) ResetClientTraffic(ctx context.Context, inboundID int, email string) error {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return fmt.Errorf("сброс трафика: не задан email клиента")
	}

	// 1. Пробуем v2 API панелей 3.x.
	pathV2 := "/panel/api/clients/resetTraffic/" + url.PathEscape(trimmed)
	if _, err := c.doJSON(ctx, "POST", pathV2, nil); err == nil {
		c.log.Debug("3x-ui: трафик сброшен (v2)", "email", trimmed)
		return nil
	}

	// 2. Фолбэк на legacy API.
	path := fmt.Sprintf("/panel/api/inbounds/%d/resetClientTraffic/%s", inboundID, url.PathEscape(trimmed))
	if _, err := c.doJSON(ctx, "POST", path, nil); err != nil {
		return fmt.Errorf("сброс трафика клиента %q: %w", trimmed, err)
	}
	return nil
}
