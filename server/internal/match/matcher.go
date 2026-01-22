package match

import (
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"miniarena/pkg/protocol"
	"miniarena/server/internal/metrics"
	"miniarena/server/internal/room"
	"miniarena/server/internal/session"
)

type Matcher struct {
	enqueueCh      chan string
	playersPerRoom int
	enqueuedAt     map[string]time.Time
	roomMgr        *room.Manager
	sessionMgr     *session.Manager
	metrics        *metrics.Metrics
	log            *zap.Logger
}

func NewMatcher(playersPerRoom int, queueSize int, roomMgr *room.Manager, sessionMgr *session.Manager, metrics *metrics.Metrics, log *zap.Logger) *Matcher {
	m := &Matcher{
		enqueueCh:      make(chan string, queueSize),
		playersPerRoom: playersPerRoom,
		enqueuedAt:     make(map[string]time.Time),
		roomMgr:        roomMgr,
		sessionMgr:     sessionMgr,
		metrics:        metrics,
		log:            log,
	}
	go m.loop()
	return m
}

func (m *Matcher) Enqueue(playerID string) bool {
	select {
	case m.enqueueCh <- playerID:
		return true
	default:
		return false
	}
}

func (m *Matcher) loop() {
	queue := make([]string, 0, m.playersPerRoom*2)
	for pid := range m.enqueueCh {
		if _, ok := m.enqueuedAt[pid]; ok {
			continue
		}
		m.enqueuedAt[pid] = time.Now()
		queue = append(queue, pid)
		if m.metrics != nil {
			m.metrics.MatchQueueGauge.Set(float64(len(queue)))
		}

		for len(queue) >= m.playersPerRoom {
			players := make([]string, 0, m.playersPerRoom)
			for len(queue) > 0 && len(players) < m.playersPerRoom {
				p := queue[0]
				queue = queue[1:]
				if m.sessionMgr.IsOnline(p) {
					players = append(players, p)
				} else {
					delete(m.enqueuedAt, p)
				}
			}

			if len(players) < m.playersPerRoom {
				queue = append(players, queue...)
				if m.metrics != nil {
					m.metrics.MatchQueueGauge.Set(float64(len(queue)))
				}
				break
			}

			if m.metrics != nil {
				m.metrics.MatchQueueGauge.Set(float64(len(queue)))
			}

			matchID := uuid.NewString()
			roomID := m.roomMgr.CreateRoom(matchID, players)
			for _, p := range players {
				m.sessionMgr.SetRoom(p, roomID)
				if ts, ok := m.enqueuedAt[p]; ok {
					if m.metrics != nil {
						m.metrics.MatchDuration.Observe(float64(time.Since(ts).Milliseconds()))
					}
					delete(m.enqueuedAt, p)
				}
				resp := &protocol.MatchResp{
					MatchId: matchID,
					RoomId:  roomID,
					Players: players,
				}
				_ = m.sessionMgr.Send(p, protocol.MsgMatchResp, resp)
			}
		}
	}
}
