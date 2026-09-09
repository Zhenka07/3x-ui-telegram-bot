package bot

import (
	"sync"
	"time"
)

// Состояния FSM-конструктора нового подключения.
type wizardState int

const (
	stateIdle          wizardState = iota
	stateSelectInbound             // Ожидание выбора инбаунда
	stateInputEmail                // Ожидание ввода имени клиента
	stateSelectTraffic             // Выбор лимита трафика
	stateInputTraffic              // Ручной ввод лимита трафика
	stateSelectExpiry              // Выбор срока действия
	stateInputExpiry               // Ручной ввод срока действия
	stateConfirm                   // Подтверждение создания

	// Состояния FSM для создания входящего подключения (Inbound):
	stateInboundWaitRemark    // Ожидание ввода названия (Remark)
	stateInboundWaitPort      // Ожидание ввода или выбора порта
	stateInboundWaitSNI       // Выбор домена маскировки
	stateInboundWaitSNICustom // Ручной ввод домена маскировки
	stateInboundConfirm       // Подтверждение создания Inbound
)

// sessionTimeout — время жизни неактивной сессии конструктора.
const sessionTimeout = 5 * time.Minute

// wizardSession — данные одной FSM-сессии конструктора.
type wizardSession struct {
	State wizardState

	// Данные для мастера клиента:
	InboundID  int
	Email      string
	TrafficGB  int64 // 0 = безлимит
	ExpiryDays int   // 0 = бессрочно

	// Данные для мастера создания Inbound:
	InboundRemark     string
	InboundPort       int
	InboundSNI        string
	InboundPrivateKey string
	InboundPublicKey  string
	InboundShortID    string

	UpdatedAt time.Time
}

// isExpired сообщает, истекла ли сессия по таймауту.
func (s *wizardSession) isExpired() bool {
	return time.Since(s.UpdatedAt) > sessionTimeout
}

// touch обновляет метку времени последней активности.
func (s *wizardSession) touch() {
	s.UpdatedAt = time.Now()
}

// reset сбрасывает сессию в начальное состояние.
func (s *wizardSession) reset() {
	s.State = stateIdle
	s.InboundID = 0
	s.Email = ""
	s.TrafficGB = 0
	s.ExpiryDays = 0
	s.InboundRemark = ""
	s.InboundPort = 0
	s.InboundSNI = ""
	s.InboundPrivateKey = ""
	s.InboundPublicKey = ""
	s.InboundShortID = ""
	s.UpdatedAt = time.Now()
}

// FSM — потокобезопасный менеджер состояний конструктора (по chat_id).
type FSM struct {
	mu       sync.RWMutex
	sessions map[int64]*wizardSession
}

// newFSM создаёт менеджер состояний.
func newFSM() *FSM {
	return &FSM{
		sessions: make(map[int64]*wizardSession),
	}
}

// Get возвращает сессию для указанного чата.
// Если сессия не существует или истекла — возвращает nil.
func (f *FSM) Get(chatID int64) *wizardSession {
	f.mu.RLock()
	sess, ok := f.sessions[chatID]
	f.mu.RUnlock()

	if !ok {
		return nil
	}
	if sess.isExpired() {
		f.Delete(chatID)
		return nil
	}
	return sess
}

// GetOrCreate возвращает существующую или создаёт новую сессию.
func (f *FSM) GetOrCreate(chatID int64) *wizardSession {
	f.mu.Lock()
	defer f.mu.Unlock()

	sess, ok := f.sessions[chatID]
	if ok && !sess.isExpired() {
		return sess
	}

	sess = &wizardSession{
		State:     stateIdle,
		UpdatedAt: time.Now(),
	}
	f.sessions[chatID] = sess
	return sess
}

// Set сохраняет сессию для чата.
func (f *FSM) Set(chatID int64, sess *wizardSession) {
	f.mu.Lock()
	sess.touch()
	f.sessions[chatID] = sess
	f.mu.Unlock()
}

// Delete удаляет сессию.
func (f *FSM) Delete(chatID int64) {
	f.mu.Lock()
	delete(f.sessions, chatID)
	f.mu.Unlock()
}

// IsActive сообщает, находится ли чат в активной сессии конструктора.
func (f *FSM) IsActive(chatID int64) bool {
	sess := f.Get(chatID)
	return sess != nil && sess.State != stateIdle
}

// Cleanup удаляет все истекшие сессии (вызывать периодически).
func (f *FSM) Cleanup() {
	f.mu.Lock()
	defer f.mu.Unlock()

	for id, sess := range f.sessions {
		if sess.isExpired() {
			delete(f.sessions, id)
		}
	}
}
