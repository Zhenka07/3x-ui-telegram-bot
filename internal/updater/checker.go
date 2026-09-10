package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	latestReleaseURL = "https://api.github.com/repos/MHSanaei/3x-ui/releases/latest"
	defaultCacheTTL  = 1 * time.Hour
)

type ReleaseInfo struct {
	Version     string    `json:"version"`
	Name        string    `json:"name"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
}

type Checker struct {
	client   *http.Client
	cacheTTL time.Duration

	mu        sync.RWMutex
	cached    *ReleaseInfo
	expiresAt time.Time
}

func NewChecker(client *http.Client, cacheTTL time.Duration) *Checker {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if cacheTTL <= 0 {
		cacheTTL = defaultCacheTTL
	}
	return &Checker{
		client:   client,
		cacheTTL: cacheTTL,
	}
}

// GetLatestRelease fetches the latest 3x-ui release info from GitHub API, with in-memory caching.
func (c *Checker) GetLatestRelease(ctx context.Context) (*ReleaseInfo, error) {
	c.mu.RLock()
	if c.cached != nil && time.Now().Before(c.expiresAt) {
		defer c.mu.RUnlock()
		return c.cached, nil
	}
	c.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, "GET", latestReleaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("создание запроса к GitHub API: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "3x-ui-admin-bot")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("запрос релизов GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API вернул статус: %d", resp.StatusCode)
	}

	var dto struct {
		TagName     string    `json:"tag_name"`
		Name        string    `json:"name"`
		HTMLURL     string    `json:"html_url"`
		PublishedAt time.Time `json:"published_at"`
		Body        string    `json:"body"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, fmt.Errorf("разбор ответа GitHub API: %w", err)
	}

	info := &ReleaseInfo{
		Version:     strings.TrimSpace(dto.TagName),
		Name:        strings.TrimSpace(dto.Name),
		HTMLURL:     strings.TrimSpace(dto.HTMLURL),
		PublishedAt: dto.PublishedAt,
		Body:        dto.Body,
	}

	c.mu.Lock()
	c.cached = info
	c.expiresAt = time.Now().Add(c.cacheTTL)
	c.mu.Unlock()

	return info, nil
}

var versionRe = regexp.MustCompile(`(\d+)`)

// ParseVersion extracts integer parts from a version string (e.g. "v2.5.1" -> [2, 5, 1]).
func ParseVersion(v string) []int {
	matches := versionRe.FindAllString(v, -1)
	parts := make([]int, 0, len(matches))
	for _, m := range matches {
		if n, err := strconv.Atoi(m); err == nil {
			parts = append(parts, n)
		}
	}
	return parts
}

// IsNewer reports whether latestVersion is strictly newer than currentVersion.
func IsNewer(currentVersion, latestVersion string) bool {
	cur := ParseVersion(currentVersion)
	latest := ParseVersion(latestVersion)

	for i := 0; i < len(cur) && i < len(latest); i++ {
		if latest[i] > cur[i] {
			return true
		}
		if latest[i] < cur[i] {
			return false
		}
	}
	return len(latest) > len(cur)
}
