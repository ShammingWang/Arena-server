package session

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/proto"
	"go.uber.org/zap"

	"miniarena/pkg/protocol"
	"miniarena/server/internal/metrics"
)

var ErrNotFound = errors.New("session not found")

// Sender is implemented by network connections.
type Sender interface {
	Send(data []byte) error
	Close() error
}

type Session struct {
	mu             sync.RWMutex
	PlayerID       string
	Username       string
	RoomID         string
	ReconnectToken string
	Online         bool
	LastSeen       time.Time
	sender         Sender
	seq            uint64
}

func (s *Session) SetSender(sender Sender) {
	var old Sender
	s.mu.Lock()
	old = s.sender
	s.sender = sender
	s.Online = true
	s.LastSeen = time.Now()
	s.mu.Unlock()
	if old != nil && old != sender {
		_ = old.Close()
	}
}

func (s *Session) ClearSender() {
	s.mu.Lock()
	s.sender = nil
	s.Online = false
	s.LastSeen = time.Now()
	s.mu.Unlock()
}

func (s *Session) Send(msgType protocol.MsgType, msg proto.Message) error {
	s.mu.RLock()
	sender := s.sender
	online := s.Online
	s.mu.RUnlock()
	if !online || sender == nil {
		return errors.New("session offline")
	}

	seq := atomic.AddUint64(&s.seq, 1)
	payload, err := protocol.Encode(msgType, msg, seq)
	if err != nil {
		return err
	}
	return sender.Send(payload)
}

func (s *Session) GetRoomID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.RoomID
}

type Manager struct {
	mu           sync.RWMutex
	sessions     map[string]*Session
	reconnectTTL time.Duration
	metrics      *metrics.Metrics
	log          *zap.Logger
}

func NewManager(reconnectTTL time.Duration, metrics *metrics.Metrics, log *zap.Logger) *Manager {
	m := &Manager{
		sessions:     make(map[string]*Session),
		reconnectTTL: reconnectTTL,
		metrics:      metrics,
		log:          log,
	}
	go m.cleanupLoop()
	return m
}

func (m *Manager) Create(playerID, username, reconnectToken string, sender Sender) *Session {
	s := &Session{
		PlayerID:       playerID,
		Username:       username,
		ReconnectToken: reconnectToken,
		Online:         true,
		LastSeen:       time.Now(),
		sender:         sender,
	}

	m.mu.Lock()
	m.sessions[playerID] = s
	m.mu.Unlock()
	m.updateOnlineGauge()
	return s
}

func (m *Manager) Bind(playerID string, sender Sender) (*Session, bool) {
	m.mu.RLock()
	s := m.sessions[playerID]
	m.mu.RUnlock()
	if s == nil {
		return nil, false
	}
	s.SetSender(sender)
	m.updateOnlineGauge()
	return s, true
}

func (m *Manager) SetRoom(playerID, roomID string) {
	m.mu.RLock()
	s := m.sessions[playerID]
	m.mu.RUnlock()
	if s == nil {
		return
	}
	s.mu.Lock()
	s.RoomID = roomID
	s.mu.Unlock()
}

func (m *Manager) MarkOffline(playerID string) {
	m.mu.RLock()
	s := m.sessions[playerID]
	m.mu.RUnlock()
	if s == nil {
		return
	}
	s.ClearSender()
	m.updateOnlineGauge()
}

func (m *Manager) Remove(playerID string) {
	m.mu.Lock()
	delete(m.sessions, playerID)
	m.mu.Unlock()
	m.updateOnlineGauge()
}

func (m *Manager) Get(playerID string) (*Session, bool) {
	m.mu.RLock()
	s := m.sessions[playerID]
	m.mu.RUnlock()
	if s == nil {
		return nil, false
	}
	return s, true
}

func (m *Manager) IsOnline(playerID string) bool {
	s, ok := m.Get(playerID)
	if !ok {
		return false
	}
	s.mu.RLock()
	online := s.Online
	s.mu.RUnlock()
	return online
}

func (m *Manager) Send(playerID string, msgType protocol.MsgType, msg proto.Message) error {
	s, ok := m.Get(playerID)
	if !ok {
		return ErrNotFound
	}
	return s.Send(msgType, msg)
}

func (m *Manager) Broadcast(playerIDs []string, msgType protocol.MsgType, msg proto.Message) {
	for _, pid := range playerIDs {
		_ = m.Send(pid, msgType, msg)
	}
}

func (m *Manager) updateOnlineGauge() {
	if m.metrics == nil {
		return
	}
	count := 0
	m.mu.RLock()
	for _, s := range m.sessions {
		s.mu.RLock()
		if s.Online {
			count++
		}
		s.mu.RUnlock()
	}
	m.mu.RUnlock()
	m.metrics.OnlineGauge.Set(float64(count))
}

func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var toRemove []string
		now := time.Now()
		m.mu.RLock()
		for id, s := range m.sessions {
			s.mu.RLock()
			offline := !s.Online
			last := s.LastSeen
			s.mu.RUnlock()
			if offline && now.Sub(last) > m.reconnectTTL {
				toRemove = append(toRemove, id)
			}
		}
		m.mu.RUnlock()
		if len(toRemove) == 0 {
			continue
		}
		m.mu.Lock()
		for _, id := range toRemove {
			delete(m.sessions, id)
		}
		m.mu.Unlock()
		m.updateOnlineGauge()
	}
}
