package xui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// doJSON sends a JSON request to the 3x-ui API and unmarshals the response envelope.
func (c *APIClient) doJSON(ctx context.Context, method, path string, payload any) (*apiResponse, error) {
	var rawBody []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("сериализация тела запроса %s: %w", path, err)
		}
		rawBody = encoded
	}

	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 1; attempt <= maxLoginAttempts; attempt++ {
		resp, body, err := c.send(ctx, method, path, rawBody)
		if err != nil {
			return nil, err
		}

		if isSessionExpired(resp, body) {
			drainAndClose(resp.Body)
			c.invalidateSession()
			lastErr = fmt.Errorf("%w (запрос %s %s)", ErrUnauthorized, method, path)

			if attempt == maxLoginAttempts {
				break
			}

			c.log.Warn("3x-ui: сессия истекла, выполняется повторный логин",
				"method", method, "path", path, "attempt", attempt)
			if err := c.ensureSession(ctx); err != nil {
				return nil, fmt.Errorf("повторная авторизация после истечения сессии: %w", err)
			}
			continue
		}
		drainAndClose(resp.Body)

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: запрос %s %s: HTTP-статус %d: %s",
				ErrNotFound, method, path, resp.StatusCode, truncate(body))
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("запрос %s %s: HTTP-статус %d: %s",
				method, path, resp.StatusCode, truncate(body))
		}

		var parsed apiResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, fmt.Errorf("разбор ответа %s %s (%s): %w", method, path, truncate(body), err)
		}
		if !parsed.Success {
			return nil, fmt.Errorf("%w: %s %s: %s", ErrAPIFailure, method, path, parsed.Msg)
		}
		return &parsed, nil
	}
	return nil, lastErr
}

// send executes a single HTTP request and returns the response with its body.
func (c *APIClient) send(ctx context.Context, method, path string, rawBody []byte) (*http.Response, []byte, error) {
	endpoint := c.baseURL + path

	var reader io.Reader
	if rawBody != nil {
		reader = bytes.NewReader(rawBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, nil, fmt.Errorf("формирование запроса %s %s: %w", method, path, err)
	}
	req.Header.Set("Accept", "application/json")
	if rawBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", c.csrfToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("выполнение запроса %s %s: %w", method, path, err)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		drainAndClose(resp.Body)
		return nil, nil, fmt.Errorf("чтение ответа %s %s: %w", method, path, err)
	}
	return resp, body, nil
}

// isSessionExpired checks whether the HTTP response indicates an expired session.
func isSessionExpired(resp *http.Response, body []byte) bool {
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return true
	}

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		if location == "" {
			return false
		}
		return strings.Contains(strings.ToLower(location), "login")
	}

	if resp.StatusCode != http.StatusOK {
		return false
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "text/html") {
		return true
	}

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		lower := strings.ToLower(string(trimmed))
		return strings.Contains(lower, "login")
	}
	return false
}
