package bot

import (
	"sync"
	"time"
)

type wizardState int

const (
	stateIdle wizardState = iota
	stateSelectInbound
	stateInputEmail
	stateSelectTraffic
	stateInputTraffic
	stateSelectExpiry
	stateInputExpiry
	stateConfirm

	stateInboundWaitRemark
	stateInboundWaitPort
	stateInboundWaitSNI
	stateInboundWaitSNICustom
	stateInboundConfirm
)

const sessionTimeout = 5 * time.Minute

type wizardSession struct {
	State wizardState

	InboundID  int
	Email      string
	TrafficGB  int64
	ExpiryDays int

	InboundRemark     string
	InboundPort       int
	InboundSNI        string
	InboundPrivateKey string
	InboundPublicKey  string
	InboundShortID    string

	UpdatedAt time.Time
}

// isExpired reports whether the session has expired due to timeout.
func (s *wizardSession) isExpired() bool {
	return time.Since(s.UpdatedAt) > sessionTimeout
}

// touch updates the last activity timestamp.
func (s *wizardSession) touch() {
	s.UpdatedAt = time.Now()
}

// reset resets the session to its initial idle state.
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

type FSM struct {
	mu       sync.RWMutex
	sessions map[int64]*wizardSession
}

// newFSM creates a new state manager.
func newFSM() *FSM {
	return &FSM{
		sessions: make(map[int64]*wizardSession),
	}
}

// Get returns the session for the specified chat ID, or nil if not found or expired.
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

// GetOrCreate returns an existing active session or creates a new one.
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

// Set saves the session for the specified chat ID.
func (f *FSM) Set(chatID int64, sess *wizardSession) {
	f.mu.Lock()
	sess.touch()
	f.sessions[chatID] = sess
	f.mu.Unlock()
}

// Delete removes the session for the specified chat ID.
func (f *FSM) Delete(chatID int64) {
	f.mu.Lock()
	delete(f.sessions, chatID)
	f.mu.Unlock()
}

// IsActive reports whether the specified chat has an active wizard session.
func (f *FSM) IsActive(chatID int64) bool {
	sess := f.Get(chatID)
	return sess != nil && sess.State != stateIdle
}

// Cleanup removes all expired sessions.
func (f *FSM) Cleanup() {
	f.mu.Lock()
	defer f.mu.Unlock()

	for id, sess := range f.sessions {
		if sess.isExpired() {
			delete(f.sessions, id)
		}
	}
}
