package room

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"miniarena/server/internal/metrics"
	"miniarena/server/internal/store"
)

type Manager struct {
	mu      sync.RWMutex
	rooms   map[string]*Room
	tick    time.Duration
	sender  Sender
	idem    store.Idempotency
	metrics *metrics.Metrics
	log     *zap.Logger
	onRoomClosed func(roomID string, players []string)
}

func NewManager(tick time.Duration, sender Sender, idem store.Idempotency, metrics *metrics.Metrics, log *zap.Logger, onRoomClosed func(roomID string, players []string)) *Manager {
	return &Manager{
		rooms:   make(map[string]*Room),
		tick:    tick,
		sender:  sender,
		idem:    idem,
		metrics: metrics,
		log:     log,
		onRoomClosed: onRoomClosed,
	}
}

func (m *Manager) CreateRoom(matchID string, players []string) string {
	roomID := uuid.NewString()
	room := NewRoom(roomID, matchID, players, m.tick, m.sender, m.idem, m.metrics, m.log, m.closeRoom)

	m.mu.Lock()
	m.rooms[roomID] = room
	m.mu.Unlock()

	room.Start()
	return roomID
}

func (m *Manager) SendEvent(roomID string, ev Event) {
	m.mu.RLock()
	room := m.rooms[roomID]
	m.mu.RUnlock()
	if room == nil {
		return
	}
	room.SendEvent(ev)
}

func (m *Manager) remove(roomID string) {
	m.mu.Lock()
	delete(m.rooms, roomID)
	m.mu.Unlock()
}

func (m *Manager) closeRoom(roomID string, players []string) {
	m.remove(roomID)
	if m.onRoomClosed != nil {
		m.onRoomClosed(roomID, players)
	}
}
