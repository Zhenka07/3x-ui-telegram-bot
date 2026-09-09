package xui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type apiResponse struct {
	Success bool            `json:"success"`
	Msg     string          `json:"msg"`
	Obj     json.RawMessage `json:"obj"`
}

type Inbound struct {
	ID       int
	Remark   string
	Port     int
	Protocol string
	Enable   bool
	Up       int64
	Down     int64
	Total    int64
	Tag      string

	Clients []Client

	IsReality bool
	Reality   RealityInfo
}

// TotalTraffic returns the total traffic transferred by the inbound in bytes.
func (i Inbound) TotalTraffic() int64 {
	return i.Up + i.Down
}

type RealityInfo struct {
	PublicKey   string
	ServerNames []string
	ShortIDs    []string
	Fingerprint string
	SpiderX     string
	Dest        string
}

type X25519Cert struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type CreateInboundSpec struct {
	Remark     string
	Port       int
	DestDomain string
	PrivateKey string
	PublicKey  string
	ShortID    string
}

type ServerStatus struct {
	CPU float64 `json:"cpu"`
	Mem struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"mem"`
	Swap struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"swap"`
	Disk struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"disk"`
	Xray struct {
		State    string `json:"state"`
		Version  string `json:"version"`
		ErrorMsg string `json:"errorMsg"`
	} `json:"xray"`
	Uptime   int64     `json:"uptime"`
	Loads    []float64 `json:"loads"`
	TCPCount int       `json:"tcpCount"`
	UDPCount int       `json:"udpCount"`
}

type Client struct {
	ID         string `json:"id"`
	Flow       string `json:"flow"`
	Email      string `json:"email"`
	LimitIP    int    `json:"limitIp"`
	TotalGB    int64  `json:"totalGB"`
	ExpiryTime int64  `json:"expiryTime"`
	Enable     bool   `json:"enable"`
	TgID       string `json:"tgId"`
	SubID      string `json:"subId"`
	Reset      int    `json:"reset"`
}

// UnmarshalJSON unmarshals a client from 3x-ui JSON data.
func (c *Client) UnmarshalJSON(data []byte) error {
	type alias Client

	var raw struct {
		*alias
		TgID json.RawMessage `json:"tgId"`
	}
	raw.alias = (*alias)(c)

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if len(raw.TgID) == 0 || string(raw.TgID) == "null" {
		c.TgID = ""
		return nil
	}

	c.TgID = string(bytes.Trim(raw.TgID, `"`))
	return nil
}

// ExpiryAt returns the expiration time of the client as a time.Time in UTC.
func (c Client) ExpiryAt() time.Time {
	return msToTime(c.ExpiryTime)
}

// IsExpired reports whether the client configuration has expired.
func (c Client) IsExpired() bool {
	exp := c.ExpiryAt()
	if exp.IsZero() {
		return false
	}
	return time.Now().UTC().After(exp)
}

type ClientTraffic struct {
	Email    string
	Up       int64
	Down     int64
	Total    int64
	Enable   bool
	ExpiryAt time.Time
}

// Used returns the total used traffic in bytes.
func (t ClientTraffic) Used() int64 {
	return t.Up + t.Down
}

// Remaining returns the remaining traffic in bytes and whether a limit is set.
func (t ClientTraffic) Remaining() (int64, bool) {
	if t.Total <= 0 {
		return 0, false
	}
	remaining := t.Total - t.Used()
	if remaining < 0 {
		remaining = 0
	}
	return remaining, true
}

type ClientSpec struct {
	UUID              string
	Email             string
	Flow              string
	LimitIP           int
	TrafficLimitBytes int64
	ExpiryAt          time.Time
	TelegramID        int64
	SubID             string
	Enable            bool
}

// toClient converts a ClientSpec into a Client DTO.
func (s ClientSpec) toClient() Client {
	tgID := ""
	if s.TelegramID != 0 {
		tgID = fmt.Sprintf("%d", s.TelegramID)
	}
	return Client{
		ID:         s.UUID,
		Flow:       s.Flow,
		Email:      s.Email,
		LimitIP:    s.LimitIP,
		TotalGB:    s.TrafficLimitBytes,
		ExpiryTime: timeToMS(s.ExpiryAt),
		Enable:     s.Enable,
		TgID:       tgID,
		SubID:      s.SubID,
		Reset:      0,
	}
}

// validate checks that the required fields of ClientSpec are valid.
func (s ClientSpec) validate() error {
	if s.UUID == "" {
		return fmt.Errorf("не задан UUID клиента")
	}
	if s.Email == "" {
		return fmt.Errorf("не задан email (имя) клиента")
	}
	if s.TrafficLimitBytes < 0 {
		return fmt.Errorf("лимит трафика не может быть отрицательным: %d", s.TrafficLimitBytes)
	}
	return nil
}

type clientSettings struct {
	Clients []Client `json:"clients"`
}

type addClientRequest struct {
	ID       int    `json:"id"`
	Settings string `json:"settings"`
}

// newAddClientRequest formats an add client request by packaging clients into a JSON string.
func newAddClientRequest(inboundID int, clients ...Client) (*addClientRequest, error) {
	if inboundID <= 0 {
		return nil, fmt.Errorf("некорректный inbound id: %d", inboundID)
	}
	if len(clients) == 0 {
		return nil, fmt.Errorf("список клиентов пуст")
	}

	encoded, err := json.Marshal(clientSettings{Clients: clients})
	if err != nil {
		return nil, fmt.Errorf("сериализация settings: %w", err)
	}

	return &addClientRequest{
		ID:       inboundID,
		Settings: string(encoded),
	}, nil
}

type addInboundRequest struct {
	Remark         string `json:"remark"`
	Port           int    `json:"port"`
	Protocol       string `json:"protocol"`
	Enable         bool   `json:"enable"`
	Listen         string `json:"listen"`
	Total          int64  `json:"total"`
	ExpiryTime     int64  `json:"expiryTime"`
	Settings       string `json:"settings"`
	Sniffing       string `json:"sniffing"`
	StreamSettings string `json:"streamSettings"`
}

type inboundStreamPayload struct {
	Network         string                `json:"network"`
	Security        string                `json:"security"`
	RealitySettings inboundRealityPayload `json:"realitySettings"`
}

type inboundRealityPayload struct {
	Show        bool                      `json:"show"`
	Xver        int                       `json:"xver"`
	Target      string                    `json:"target"`
	Dest        string                    `json:"dest"`
	ServerNames []string                  `json:"serverNames"`
	PrivateKey  string                    `json:"privateKey"`
	ShortIds    []string                  `json:"shortIds"`
	Settings    inboundRealitySubSettings `json:"settings"`
}

type inboundRealitySubSettings struct {
	PublicKey   string `json:"publicKey"`
	Fingerprint string `json:"fingerprint"`
	ServerName  string `json:"serverName"`
	SpiderX     string `json:"spiderX"`
}

// newAddInboundRequest constructs an add inbound request for a VLESS-Reality inbound.
func newAddInboundRequest(spec CreateInboundSpec) (*addInboundRequest, error) {
	if strings.TrimSpace(spec.Remark) == "" {
		return nil, fmt.Errorf("не указано название инбаунда (remark)")
	}
	if spec.Port < 1 || spec.Port > 65535 {
		return nil, fmt.Errorf("некорректный порт: %d", spec.Port)
	}
	trimmedDomain := strings.TrimSpace(spec.DestDomain)
	if trimmedDomain == "" {
		return nil, fmt.Errorf("не указан домен маскировки (dest)")
	}

	destWithPort := trimmedDomain
	if !strings.Contains(destWithPort, ":") {
		destWithPort += ":443"
	}

	serverNames := []string{trimmedDomain}
	if !strings.HasPrefix(trimmedDomain, "www.") {
		serverNames = append(serverNames, "www."+trimmedDomain)
	}

	stream := inboundStreamPayload{
		Network:  "tcp",
		Security: "reality",
		RealitySettings: inboundRealityPayload{
			Show:        false,
			Xver:        0,
			Target:      destWithPort,
			Dest:        destWithPort,
			ServerNames: serverNames,
			PrivateKey:  spec.PrivateKey,
			ShortIds:    []string{spec.ShortID},
			Settings: inboundRealitySubSettings{
				PublicKey:   spec.PublicKey,
				Fingerprint: "chrome",
				ServerName:  "",
				SpiderX:     "/",
			},
		},
	}

	streamEncoded, err := json.Marshal(stream)
	if err != nil {
		return nil, fmt.Errorf("сериализация streamSettings: %w", err)
	}

	return &addInboundRequest{
		Remark:         spec.Remark,
		Port:           spec.Port,
		Protocol:       "vless",
		Enable:         true,
		Listen:         "",
		Total:          0,
		ExpiryTime:     0,
		Settings:       `{"clients":[],"decryption":"none","fallbacks":[]}`,
		Sniffing:       `{"enabled":true,"destOverride":["http","tls","quic","fakedns"]}`,
		StreamSettings: string(streamEncoded),
	}, nil
}

type clientTrafficDTO struct {
	ID         int64  `json:"id"`
	InboundID  int    `json:"inboundId"`
	Enable     bool   `json:"enable"`
	Email      string `json:"email"`
	Up         int64  `json:"up"`
	Down       int64  `json:"down"`
	Total      int64  `json:"total"`
	ExpiryTime int64  `json:"expiryTime"`
	Reset      int    `json:"reset"`
}

type rawInbound struct {
	ID             int             `json:"id"`
	Up             int64           `json:"up"`
	Down           int64           `json:"down"`
	Total          int64           `json:"total"`
	Remark         string          `json:"remark"`
	Enable         bool            `json:"enable"`
	ExpiryTime     int64           `json:"expiryTime"`
	Listen         string          `json:"listen"`
	Port           int             `json:"port"`
	Protocol       string          `json:"protocol"`
	Settings       json.RawMessage `json:"settings"`
	StreamSettings json.RawMessage `json:"streamSettings"`
	Tag            string          `json:"tag"`
}

type inboundSettings struct {
	Clients    []Client `json:"clients"`
	Decryption string   `json:"decryption"`
}

type streamSettingsJSON struct {
	Network         string              `json:"network"`
	Security        string              `json:"security"`
	RealitySettings *realitySettingsJSON `json:"realitySettings"`
}

type realitySettingsJSON struct {
	Show        bool               `json:"show"`
	Dest        string             `json:"dest"`
	ServerNames []string           `json:"serverNames"`
	PrivateKey  string             `json:"privateKey"`
	ShortIds    []string           `json:"shortIds"`
	Settings    realitySubSettings `json:"settings"`
}

type realitySubSettings struct {
	PublicKey   string `json:"publicKey"`
	Fingerprint string `json:"fingerprint"`
	SpiderX     string `json:"spiderX"`
}

// msToTime converts Unix time in milliseconds to time.Time in UTC.
func msToTime(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

// timeToMS converts a time.Time into Unix time in milliseconds.
func timeToMS(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
