package sshtunnel

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Host     string
	Port     int
	User     string
	KeyPath  string
	Password string
	Timeout  time.Duration
}

type Tunnel struct {
	cfg       Config
	sshConfig *ssh.ClientConfig
	client    *ssh.Client
	mu        sync.Mutex
}

// New creates and prepares a new SSH tunnel.
func New(cfg Config) (*Tunnel, error) {
	if strings.TrimSpace(cfg.Host) == "" {
		return nil, fmt.Errorf("не задан хост для SSH подключения")
	}
	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	if cfg.User == "" {
		cfg.User = "root"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	var authMethods []ssh.AuthMethod
	if cfg.KeyPath != "" {
		keyBytes, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("чтение приватного ключа SSH %q: %w", cfg.KeyPath, err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("разбор приватного ключа SSH: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("не указан метод авторизации SSH (требуется ключ или пароль)")
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         cfg.Timeout,
	}

	return &Tunnel{
		cfg:       cfg,
		sshConfig: sshConfig,
	}, nil
}

// DialContext establishes a TCP connection to the remote address through the SSH tunnel.
func (t *Tunnel) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.client == nil {
		if err := t.connectLocked(); err != nil {
			return nil, err
		}
	}

	conn, err := t.client.Dial(network, addr)
	if err != nil {
		_ = t.client.Close()
		t.client = nil
		if recErr := t.connectLocked(); recErr != nil {
			return nil, fmt.Errorf("переподключение SSH туннеля: %w (первичная ошибка: %v)", recErr, err)
		}
		conn, err = t.client.Dial(network, addr)
		if err != nil {
			return nil, fmt.Errorf("dial через SSH туннель к %s: %w", addr, err)
		}
	}

	return conn, nil
}

func (t *Tunnel) connectLocked() error {
	target := net.JoinHostPort(t.cfg.Host, strconv.Itoa(t.cfg.Port))
	client, err := ssh.Dial("tcp", target, t.sshConfig)
	if err != nil {
		return fmt.Errorf("установка соединения SSH с %s: %w", target, err)
	}
	t.client = client
	return nil
}

// Close closes the underlying SSH connection if active.
func (t *Tunnel) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.client != nil {
		err := t.client.Close()
		t.client = nil
		return err
	}
	return nil
}

// ExecuteCommand executes a remote shell command via SSH and returns its combined output.
func (t *Tunnel) ExecuteCommand(ctx context.Context, cmd string) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.client == nil {
		if err := t.connectLocked(); err != nil {
			return "", err
		}
	}

	session, err := t.client.NewSession()
	if err != nil {
		_ = t.client.Close()
		t.client = nil
		if recErr := t.connectLocked(); recErr != nil {
			return "", fmt.Errorf("переподключение SSH: %w (первичная ошибка: %v)", recErr, err)
		}
		session, err = t.client.NewSession()
		if err != nil {
			return "", fmt.Errorf("создание сессии SSH: %w", err)
		}
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(out), fmt.Errorf("команда завершилась с ошибкой: %w", err)
	}
	return string(out), nil
}

