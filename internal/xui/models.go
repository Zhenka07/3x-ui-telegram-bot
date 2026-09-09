package xui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ── API envelope ────────────────────────────────────────────────────

// apiResponse — стандартный конверт ответа панели 3x-ui.
type apiResponse struct {
	Success bool            `json:"success"`
	Msg     string          `json:"msg"`
	Obj     json.RawMessage `json:"obj"`
}

// ── Inbound (публичные типы) ────────────────────────────────────────

// Inbound — разобранный инбаунд с распаршенными клиентами и Reality-параметрами.
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

	// Reality-специфичные поля (заполняются только для Reality-инбаундов).
	IsReality bool
	Reality   RealityInfo
}

// TotalTraffic возвращает суммарный трафик инбаунда (upload + download).
func (i Inbound) TotalTraffic() int64 {
	return i.Up + i.Down
}

// RealityInfo — извлечённые параметры Reality из streamSettings инбаунда.
type RealityInfo struct {
	PublicKey   string
	ServerNames []string
	ShortIDs    []string
	Fingerprint string
	SpiderX     string
	Dest        string
}

// X25519Cert — пара ключей x25519 для Reality.
type X25519Cert struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

// CreateInboundSpec — спецификация для создания нового инбаунда VLESS-Reality.
type CreateInboundSpec struct {
	Remark     string
	Port       int
	DestDomain string // например, "gateway.icloud.com" или "apple.com"
	PrivateKey string
	PublicKey  string
	ShortID    string
}

// ServerStatus — системный статус хоста и ядра Xray из 3x-ui.
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

// ── Client ──────────────────────────────────────────────────────────

// Client — описание VLESS-клиента внутри инбаунда 3x-ui.
type Client struct {
	ID         string `json:"id"`         // UUID клиента
	Flow       string `json:"flow"`       // xtls-rprx-vision для Reality
	Email      string `json:"email"`      // уникальное имя клиента в инбаунде
	LimitIP    int    `json:"limitIp"`    // лимит одновременных подключений, 0 = без лимита
	TotalGB    int64  `json:"totalGB"`    // лимит трафика в БАЙТАХ (несмотря на имя поля)
	ExpiryTime int64  `json:"expiryTime"` // Unix time в МИЛЛИСЕКУНДАХ, 0 = бессрочно
	Enable     bool   `json:"enable"`
	TgID       string `json:"tgId"`
	SubID      string `json:"subId"`
	Reset      int    `json:"reset"`
}

// UnmarshalJSON разбирает клиента из JSON панели.
//
// В разных версиях 3x-ui поле tgId приходит по-разному: в новых версиях (3.x)
// это число, в старых — строка или пусто. Обходим это несоответствие,
// допуская оба варианта, чтобы распарсить остальные поля клиента.
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

	// tgId может отсутствовать, быть строкой или числом.
	if len(raw.TgID) == 0 || string(raw.TgID) == "null" {
		c.TgID = ""
		return nil
	}

	c.TgID = string(bytes.Trim(raw.TgID, `"`))
	return nil
}

// ExpiryAt возвращает момент истечения в time.Time (UTC).
func (c Client) ExpiryAt() time.Time {
	return msToTime(c.ExpiryTime)
}

// IsExpired сообщает, истёк ли срок действия клиента.
func (c Client) IsExpired() bool {
	exp := c.ExpiryAt()
	if exp.IsZero() {
		return false
	}
	return time.Now().UTC().After(exp)
}

// ClientTraffic — статистика трафика клиента из панели.
type ClientTraffic struct {
	Email    string
	Up       int64
	Down     int64
	Total    int64
	Enable   bool
	ExpiryAt time.Time
}

// Used возвращает суммарный использованный трафик (upload + download).
func (t ClientTraffic) Used() int64 {
	return t.Up + t.Down
}

// Remaining возвращает остаток трафика в байтах и признак наличия лимита.
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

// ── ClientSpec (для записи) ─────────────────────────────────────────

// ClientSpec — параметры создаваемого/обновляемого VLESS-клиента.
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

// toClient конвертирует спецификацию в DTO панели.
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

// validate проверяет минимально необходимые поля спецификации.
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

// ── Внутренние DTO для сериализации ─────────────────────────────────

// clientSettings — контейнер для поля settings запроса addClient.
type clientSettings struct {
	Clients []Client `json:"clients"`
}

// addClientRequest — пейлоад POST /panel/api/inbounds/addClient.
//
// КРИТИЧНО: поле Settings — это строка с ЭКРАНИРОВАННЫМ JSON,
// а не вложенный объект. Панель 3x-ui парсит её отдельно.
type addClientRequest struct {
	ID       int    `json:"id"`       // ID инбаунда
	Settings string `json:"settings"` // JSON-строка вида {"clients":[...]}
}

// newAddClientRequest формирует запрос, упаковывая клиента в JSON-строку.
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

// addInboundRequest — пейлоад POST /panel/api/inbounds/add.
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

// newAddInboundRequest формирует запрос на создание Reality-инбаунда.
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

// clientTrafficDTO — ответ GET /panel/api/inbounds/getClientTraffics/{email}.
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

// ── Внутренние DTO для парсинга инбаундов ───────────────────────────

// rawInbound — представление инбаунда как его возвращает API 3x-ui.
//
// Поля Settings и StreamSettings в разных версиях панели приходят по-разному:
//   - старые версии (до ~2.x) возвращают ВЛОЖЕННЫЕ JSON-строки,
//   - новые версии (3.x, например 3.7.0) возвращают обычные JSON-объекты.
//
// Описываем их как json.RawMessage и разбираем в зависимости от типа значения.
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

// inboundSettings — содержимое JSON-строки поля settings инбаунда.
type inboundSettings struct {
	Clients    []Client `json:"clients"`
	Decryption string   `json:"decryption"`
}

// streamSettingsJSON — содержимое JSON-строки поля streamSettings.
type streamSettingsJSON struct {
	Network         string               `json:"network"`
	Security        string               `json:"security"`
	RealitySettings *realitySettingsJSON  `json:"realitySettings"`
}

// realitySettingsJSON — блок realitySettings внутри streamSettings.
type realitySettingsJSON struct {
	Show        bool               `json:"show"`
	Dest        string             `json:"dest"`
	ServerNames []string           `json:"serverNames"`
	PrivateKey  string             `json:"privateKey"`
	ShortIds    []string           `json:"shortIds"`
	Settings    realitySubSettings `json:"settings"`
}

// realitySubSettings — вложенный блок settings внутри realitySettings.
type realitySubSettings struct {
	PublicKey   string `json:"publicKey"`
	Fingerprint string `json:"fingerprint"`
	SpiderX     string `json:"spiderX"`
}

// ── Хелперы конверсии времени ───────────────────────────────────────

// msToTime переводит Unix-время в миллисекундах в time.Time (UTC).
func msToTime(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

// timeToMS переводит time.Time в Unix-время в миллисекундах.
func timeToMS(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
