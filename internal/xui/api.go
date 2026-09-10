package xui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// AddClient adds a new client to the specified inbound.
func (c *APIClient) AddClient(ctx context.Context, inboundID int, spec ClientSpec) error {
	if err := spec.validate(); err != nil {
		return fmt.Errorf("добавление клиента в инбаунд %d: %w", inboundID, err)
	}

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

// GetClientTraffics retrieves client traffic statistics by email.
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

// DeleteClient deletes a client from the specified inbound by UUID and email.
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

	if trimmedUUID != "" {
		if err := attempt(trimmedUUID); err == nil {
			c.log.Debug("3x-ui: клиент удалён (v2 uuid)", "uuid", trimmedUUID)
			return nil
		} else {
			lastErr = err
		}
	}

	if trimmedEmail != "" {
		if err := attempt(trimmedEmail); err == nil {
			c.log.Debug("3x-ui: клиент удалён (v2 email)", "email", trimmedEmail)
			return nil
		} else {
			lastErr = err
		}
	}

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

// UpdateClient updates parameters of an existing client in the inbound.
func (c *APIClient) UpdateClient(ctx context.Context, inboundID int, spec ClientSpec) error {
	if err := spec.validate(); err != nil {
		return fmt.Errorf("обновление клиента в инбаунде %d: %w", inboundID, err)
	}

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

// ResetClientTraffic resets the traffic counter for a client.
func (c *APIClient) ResetClientTraffic(ctx context.Context, inboundID int, email string) error {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return fmt.Errorf("сброс трафика: не задан email клиента")
	}

	pathV2 := "/panel/api/clients/resetTraffic/" + url.PathEscape(trimmed)
	if _, err := c.doJSON(ctx, "POST", pathV2, nil); err == nil {
		c.log.Debug("3x-ui: трафик сброшен (v2)", "email", trimmed)
		return nil
	}

	path := fmt.Sprintf("/panel/api/inbounds/%d/resetClientTraffic/%s", inboundID, url.PathEscape(trimmed))
	if _, err := c.doJSON(ctx, "POST", path, nil); err != nil {
		return fmt.Errorf("сброс трафика клиента %q: %w", trimmed, err)
	}
	return nil
}

var connectionPrefixes = []string{
	"vless://",
	"vmess://",
	"trojan://",
	"ss://",
	"hysteria2://",
	"hy2://",
}

// GetClientSubLinks retrieves direct share links generated by 3x-ui for the specified client subscription ID.
func (c *APIClient) GetClientSubLinks(ctx context.Context, subID string) ([]string, error) {
	trimmed := strings.TrimSpace(subID)
	if trimmed == "" {
		return nil, fmt.Errorf("получение ссылок подписки: пустой subID")
	}

	path := "/panel/api/clients/subLinks/" + url.PathEscape(trimmed)
	resp, err := c.doJSON(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("запрос ссылок подписки %q: %w", trimmed, err)
	}

	links := extractConnectionLinks(resp.Obj)
	if len(links) == 0 {
		return nil, fmt.Errorf("панель 3x-ui не вернула ссылок подключения для subID %q", trimmed)
	}
	return links, nil
}

// extractConnectionLinks parses raw JSON message to extract connection links.
func extractConnectionLinks(raw json.RawMessage) []string {
	var result []string
	seen := make(map[string]bool)

	addLink := func(s string) {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			return
		}
		lower := strings.ToLower(trimmed)
		for _, prefix := range connectionPrefixes {
			if strings.HasPrefix(lower, prefix) {
				if !seen[trimmed] {
					seen[trimmed] = true
					result = append(result, trimmed)
				}
				break
			}
		}
	}

	// Try unmarshaling as string (e.g. newline-separated URLs)
	var strVal string
	if err := json.Unmarshal(raw, &strVal); err == nil {
		for _, line := range strings.Split(strVal, "\n") {
			addLink(line)
		}
		if len(result) > 0 {
			return result
		}
	}

	// Try unmarshaling as []string
	var strSlice []string
	if err := json.Unmarshal(raw, &strSlice); err == nil {
		for _, item := range strSlice {
			for _, line := range strings.Split(item, "\n") {
				addLink(line)
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	// Try unmarshaling as generic any (nested arrays/maps)
	var genericVal any
	if err := json.Unmarshal(raw, &genericVal); err == nil {
		var walk func(v any)
		walk = func(v any) {
			switch val := v.(type) {
			case string:
				for _, line := range strings.Split(val, "\n") {
					addLink(line)
				}
			case []any:
				for _, item := range val {
					walk(item)
				}
			case map[string]any:
				for _, item := range val {
					walk(item)
				}
			}
		}
		walk(genericVal)
	}

	return result
}

// GetOnlineClients retrieves a list of client emails that are currently online in 3x-ui.
func (c *APIClient) GetOnlineClients(ctx context.Context) ([]string, error) {
	endpoints := []string{
		"/panel/api/clients/onlines",
		"/panel/api/inbounds/onlines",
	}

	var lastErr error
	for _, ep := range endpoints {
		resp, err := c.doJSON(ctx, "POST", ep, nil)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			lastErr = err
			continue
		}

		emails := extractOnlineEmails(resp.Obj)
		return emails, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("запрос онлайн-клиентов: %w", lastErr)
	}
	return []string{}, nil
}

func extractOnlineEmails(raw json.RawMessage) []string {
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return cleanEmails(list)
	}

	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err == nil {
		for _, key := range []string{"emails", "online", "clients", "items", "obj"} {
			if val, ok := obj[key]; ok {
				if arr, ok := val.([]any); ok {
					var result []string
					for _, item := range arr {
						if s, ok := item.(string); ok && s != "" {
							result = append(result, s)
						}
					}
					return cleanEmails(result)
				}
			}
		}
	}

	return []string{}
}

func cleanEmails(items []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, it := range items {
		trimmed := strings.TrimSpace(it)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			out = append(out, trimmed)
		}
	}
	sort.Strings(out)
	return out
}

// GetLogs retrieves recent logs from the 3x-ui panel API.
func (c *APIClient) GetLogs(ctx context.Context, count int) (string, error) {
	if count <= 0 {
		count = 50
	}

	endpoints := []string{
		fmt.Sprintf("/panel/api/server/logs/%d", count),
		fmt.Sprintf("/server/getLogs/%d", count),
		"/panel/api/server/logs",
	}

	for _, ep := range endpoints {
		resp, err := c.doJSON(ctx, "GET", ep, nil)
		if err == nil && resp != nil && len(resp.Obj) > 0 {
			var strLog string
			if err := json.Unmarshal(resp.Obj, &strLog); err == nil && strLog != "" {
				return strLog, nil
			}
			var sliceLog []string
			if err := json.Unmarshal(resp.Obj, &sliceLog); err == nil && len(sliceLog) > 0 {
				return strings.Join(sliceLog, "\n"), nil
			}
			return string(resp.Obj), nil
		}
	}

	return "", fmt.Errorf("логи недоступны через API панели")
}



