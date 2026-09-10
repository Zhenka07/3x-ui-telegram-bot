package vless

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/zhenya/3x-ui-admin/internal/xui"
)

// BuildUniversalLink constructs a connection link based on inbound protocol and client configuration.
// Supports VLESS (Reality and standard TLS/TCP), VMess, Trojan, and Shadowsocks.
func BuildUniversalLink(serverHost string, ib *xui.Inbound, client *xui.Client) (string, error) {
	if ib == nil {
		return "", fmt.Errorf("inbound is nil")
	}
	if client == nil {
		return "", fmt.Errorf("client is nil")
	}

	trimmedHost := strings.TrimSpace(serverHost)
	if trimmedHost == "" {
		return "", fmt.Errorf("не указан хост сервера")
	}

	protocol := strings.ToLower(strings.TrimSpace(ib.Protocol))
	switch protocol {
	case "vless":
		return buildVLESSLink(trimmedHost, ib, client)
	case "vmess":
		return buildVMessLink(trimmedHost, ib, client)
	case "trojan":
		return buildTrojanLink(trimmedHost, ib, client)
	case "shadowsocks":
		return buildShadowsocksLink(trimmedHost, ib, client)
	default:
		return "", fmt.Errorf("протокол %q не поддерживается локальным генератором", ib.Protocol)
	}
}

func buildVLESSLink(serverHost string, ib *xui.Inbound, client *xui.Client) (string, error) {
	if ib.IsReality {
		params, err := ParseInboundReality(serverHost, ib, client.Flow)
		if err != nil {
			return "", err
		}
		builder, err := NewBuilder(*params)
		if err != nil {
			return "", fmt.Errorf("создание билдера vless: %w", err)
		}
		return builder.BuildURI(client.ID, ib.Remark)
	}

	// Standard VLESS (e.g. TLS or none)
	network := "tcp"
	security := "none"
	sni := ""

	if ib.StreamSettings != "" {
		var ss struct {
			Network     string `json:"network"`
			Security    string `json:"security"`
			TLSSettings *struct {
				ServerName string `json:"serverName"`
			} `json:"tlsSettings"`
		}
		if err := json.Unmarshal([]byte(ib.StreamSettings), &ss); err == nil {
			if ss.Network != "" {
				network = ss.Network
			}
			if ss.Security != "" {
				security = ss.Security
			}
			if ss.TLSSettings != nil && ss.TLSSettings.ServerName != "" {
				sni = ss.TLSSettings.ServerName
			}
		}
	}

	query := make([]string, 0, 6)
	query = append(query, "type="+url.QueryEscape(network))
	query = append(query, "security="+url.QueryEscape(security))
	query = append(query, "encryption=none")
	if client.Flow != "" {
		query = append(query, "flow="+url.QueryEscape(client.Flow))
	}
	if sni != "" {
		query = append(query, "sni="+url.QueryEscape(sni))
	}

	targetHost := hostPort(serverHost, ib.Port)
	uri := fmt.Sprintf("vless://%s@%s?%s#%s",
		url.PathEscape(client.ID),
		targetHost,
		strings.Join(query, "&"),
		url.PathEscape(ib.Remark),
	)
	return uri, nil
}

func buildVMessLink(serverHost string, ib *xui.Inbound, client *xui.Client) (string, error) {
	payload := map[string]any{
		"v":    "2",
		"ps":   ib.Remark,
		"add":  serverHost,
		"port": strconv.Itoa(ib.Port),
		"id":   client.ID,
		"aid":  "0",
		"scy":  "auto",
		"net":  "tcp",
		"type": "none",
		"tls":  "",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("маршалинг vmess payload: %w", err)
	}

	return "vmess://" + base64.StdEncoding.EncodeToString(data), nil
}

func buildTrojanLink(serverHost string, ib *xui.Inbound, client *xui.Client) (string, error) {
	password := client.Password
	if password == "" {
		password = client.ID
	}
	if password == "" {
		return "", fmt.Errorf("пароль trojan клиента не задан")
	}

	targetHost := hostPort(serverHost, ib.Port)
	return fmt.Sprintf("trojan://%s@%s?security=tls&type=tcp#%s",
		url.PathEscape(password),
		targetHost,
		url.PathEscape(ib.Remark),
	), nil
}

func buildShadowsocksLink(serverHost string, ib *xui.Inbound, client *xui.Client) (string, error) {
	password := client.Password
	if password == "" {
		password = client.ID
	}
	if password == "" {
		return "", fmt.Errorf("пароль shadowsocks клиента не задан")
	}

	userInfo := "aes-256-gcm:" + password
	encodedUserInfo := base64.URLEncoding.EncodeToString([]byte(userInfo))
	targetHost := hostPort(serverHost, ib.Port)

	return fmt.Sprintf("ss://%s@%s#%s",
		encodedUserInfo,
		targetHost,
		url.PathEscape(ib.Remark),
	), nil
}
