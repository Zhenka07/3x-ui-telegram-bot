package xui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const maxErrorBodySize = 2 << 10 // 2 KiB

var csrfRegex = regexp.MustCompile(`<meta\s+name=["']csrf-token["']\s+content=["']([^"']+)["']`)

// Login выполняет авторизацию в панели и сохраняет сессионную куку в CookieJar.
func (c *APIClient) Login(ctx context.Context) error {
	c.loginMu.Lock()
	defer c.loginMu.Unlock()
	return c.loginLocked(ctx)
}

func (c *APIClient) loginLocked(ctx context.Context) error {
	// Предварительный GET для получения сессионных кук и CSRF-токена.
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/", nil)
	if err == nil {
		if getResp, err := c.httpClient.Do(getReq); err == nil {
			body, _ := io.ReadAll(io.LimitReader(getResp.Body, 256<<10))
			drainAndClose(getResp.Body)
			match := csrfRegex.FindSubmatch(body)
			if len(match) > 1 {
				c.csrfToken = string(match[1])
			}
		}
	}

	form := url.Values{}
	form.Set("username", c.username)
	form.Set("password", c.password)

	endpoint := c.baseURL + "/login"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("формирование запроса логина: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if c.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", c.csrfToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.loggedIn = false
		return fmt.Errorf("запрос логина в 3x-ui: %w", err)
	}
	defer drainAndClose(resp.Body)

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		c.loggedIn = false
		return fmt.Errorf("чтение ответа логина: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.loggedIn = false
		return fmt.Errorf("логин в 3x-ui: неожиданный HTTP-статус %d: %s", resp.StatusCode, truncate(body))
	}

	var parsed apiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		c.loggedIn = false
		return fmt.Errorf("разбор ответа логина (%s): %w", truncate(body), err)
	}
	if !parsed.Success {
		c.loggedIn = false
		return fmt.Errorf("%w: логин отклонён: %s", ErrAPIFailure, parsed.Msg)
	}

	if len(resp.Cookies()) == 0 && !c.hasSessionCookie() {
		c.loggedIn = false
		return fmt.Errorf("логин в 3x-ui: сессионная кука не получена")
	}

	c.loggedIn = true
	c.log.Debug("3x-ui: успешная авторизация", "user", c.username)
	return nil
}

func (c *APIClient) hasSessionCookie() bool {
	u := c.baseURL
	if !strings.HasSuffix(u, "/") {
		u += "/"
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	if len(c.httpClient.Jar.Cookies(parsed)) > 0 {
		return true
	}
	if parsedRoot, err := url.Parse(parsed.Scheme + "://" + parsed.Host + "/"); err == nil {
		if len(c.httpClient.Jar.Cookies(parsedRoot)) > 0 {
			return true
		}
	}
	return false
}

func (c *APIClient) ensureSession(ctx context.Context) error {
	c.loginMu.Lock()
	defer c.loginMu.Unlock()

	if c.loggedIn && c.hasSessionCookie() {
		return nil
	}
	return c.loginLocked(ctx)
}

func (c *APIClient) invalidateSession() {
	c.loginMu.Lock()
	c.loggedIn = false
	c.loginMu.Unlock()
}

func drainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 4<<10))
	_ = body.Close()
}

func truncate(body []byte) string {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > maxErrorBodySize {
		return string(trimmed[:maxErrorBodySize]) + "…(обрезано)"
	}
	return string(trimmed)
}
