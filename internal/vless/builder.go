package vless

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type Params struct {
	ServerAddr  string
	Port        int
	PublicKey   string
	Fingerprint string
	SNI         string
	ShortID     string
	SpiderX     string
	Flow        string
	Tag         string
}

type Builder struct {
	params Params
}

// NewBuilder creates a new Builder and validates required parameters.
func NewBuilder(params Params) (*Builder, error) {
	if strings.TrimSpace(params.ServerAddr) == "" {
		return nil, fmt.Errorf("создание билдера vless: не задан адрес сервера")
	}
	if params.Port <= 0 || params.Port > 65535 {
		return nil, fmt.Errorf("создание билдера vless: некорректный порт %d", params.Port)
	}
	if strings.TrimSpace(params.PublicKey) == "" {
		return nil, fmt.Errorf("создание билдера vless: не задан Reality public key")
	}
	if strings.TrimSpace(params.SNI) == "" {
		return nil, fmt.Errorf("создание билдера vless: не задан SNI")
	}
	return &Builder{params: params}, nil
}

// BuildURI generates a VLESS connection URI for the specified UUID and label.
func (b *Builder) BuildURI(uuid, label string) (string, error) {
	trimmedUUID := strings.TrimSpace(uuid)
	if trimmedUUID == "" {
		return "", fmt.Errorf("формирование vless-ссылки: не задан UUID")
	}

	query := make([]string, 0, 8)
	appendParam := func(key, value string) {
		if value == "" {
			return
		}
		query = append(query, key+"="+url.QueryEscape(value))
	}

	appendParam("type", "tcp")
	appendParam("security", "reality")
	appendParam("pbk", b.params.PublicKey)
	appendParam("fp", b.params.Fingerprint)
	appendParam("sni", b.params.SNI)
	appendParam("sid", b.params.ShortID)
	appendParam("spx", b.params.SpiderX)
	appendParam("flow", b.params.Flow)

	tag := strings.TrimSpace(label)
	if tag == "" {
		tag = b.params.Tag
	}

	host := hostPort(b.params.ServerAddr, b.params.Port)

	uri := "vless://" + url.PathEscape(trimmedUUID) + "@" + host
	if len(query) > 0 {
		uri += "?" + strings.Join(query, "&")
	}
	if tag != "" {
		uri += "#" + url.PathEscape(tag)
	}
	return uri, nil
}

// hostPort formats an address and port into a host:port string, wrapping IPv6 addresses in brackets.
func hostPort(addr string, port int) string {
	trimmed := strings.TrimSpace(addr)
	if strings.HasPrefix(trimmed, "[") {
		return trimmed + ":" + strconv.Itoa(port)
	}
	if ip := net.ParseIP(trimmed); ip != nil && ip.To4() == nil {
		return "[" + trimmed + "]:" + strconv.Itoa(port)
	}
	return trimmed + ":" + strconv.Itoa(port)
}
