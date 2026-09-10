package xui

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"
)

var (
	ErrUnauthorized   = errors.New("сессия 3x-ui недействительна")
	ErrClientNotFound = errors.New("клиент не найден в панели 3x-ui")
	ErrAPIFailure     = errors.New("панель 3x-ui вернула ошибку")
	ErrNotFound       = errors.New("эндпоинт или ресурс не найден (404)")
)

const (
	defaultTimeout   = 20 * time.Second
	maxLoginAttempts = 2
)

type Config struct {
	BaseURL            string
	Username           string
	Password           string
	Timeout            time.Duration
	InsecureSkipVerify bool
	DialContext        func(ctx context.Context, network, addr string) (net.Conn, error)
}

type Logger interface {
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
}

type nopLogger struct{}

func (nopLogger) Debug(string, ...any) {}
func (nopLogger) Warn(string, ...any)  {}

type APIClient struct {
	baseURL  string
	username string
	password string

	httpClient *http.Client
	log        Logger

	loginMu   sync.Mutex
	loggedIn  bool
	csrfToken string
}

// New creates a new 3x-ui API client with the given configuration and logger.
func New(cfg Config, log Logger) (*APIClient, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("создание клиента 3x-ui: не задан BaseURL")
	}
	if cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("создание клиента 3x-ui: не заданы учётные данные")
	}
	if log == nil {
		log = nopLogger{}
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("создание cookie jar: %w", err)
	}

	transport := &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DialContext:         cfg.DialContext,
	}
	if cfg.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	client := &APIClient{
		baseURL:  cfg.BaseURL,
		username: cfg.Username,
		password: cfg.Password,
		log:      log,
		httpClient: &http.Client{
			Timeout:   timeout,
			Jar:       jar,
			Transport: transport,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	return client, nil
}
