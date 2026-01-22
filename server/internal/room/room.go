package room

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"go.uber.org/zap"

	"miniarena/pkg/protocol"
	"miniarena/server/internal/battle"
	"miniarena/server/internal/metrics"
	"miniarena/server/internal/store"
)

type Sender interface {
	Send(playerID string, msgType protocol.MsgType, msg proto.Message) error
}

type Room struct {
	id        string
	matchID   string
	players   []string
	events    chan Event
	state     *battle.State
	tick      time.Duration
	sender    Sender
	idem      store.Idempotency
	metrics   *metrics.Metrics
	log       *zap.Logger
	done      chan struct{}
	onClose   func(roomID string, players []string)
}

func NewRoom(id, matchID string, players []string, tick time.Duration, sender Sender, idem store.Idempotency, metrics *metrics.Metrics, log *zap.Logger, onClose func(roomID string, players []string)) *Room {
	return &Room{
		id:      id,
		matchID: matchID,
		players: players,
		events:  make(chan Event, 128),
		state:   battle.NewState(players),
		tick:    tick,
		sender:  sender,
		idem:    idem,
		metrics: metrics,
		log:     log,
		done:    make(chan struct{}),
		onClose: onClose,
	}
}

func (r *Room) ID() string { return r.id }

func (r *Room) Start() {
	go r.loop()
}

func (r *Room) Stop() {
	select {
	case <-r.done:
		return
	default:
		close(r.done)
	}
}

func (r *Room) SendEvent(ev Event) {
	select {
	case r.events <- ev:
	default:
		r.log.Warn("room event dropped", zap.String("room", r.id))
	}
}

func (r *Room) loop() {
	ticker := time.NewTicker(r.tick)
	defer ticker.Stop()
	defer r.closeRoom()

	for {
		select {
		case ev := <-r.events:
			r.handleEvent(ev)
		case <-ticker.C:
			start := time.Now()
			r.state.TickForward()
			r.broadcastSnapshot()
			if r.metrics != nil {
				r.metrics.RoomTickDelay.Observe(float64(time.Since(start).Milliseconds()))
			}
			if winner, ok := r.state.Winner(); ok {
				r.broadcastRoomOver(winner)
				return
			}
		case <-r.done:
			return
		}
	}
}

func (r *Room) handleEvent(ev Event) {
	switch ev.Type {
	case EventJoin:
		// Reserved for future use.
	case EventLeave:
		p := r.state.Players[ev.PlayerID]
		if p != nil {
			p.HP = 0
		}
	case EventInput:
		r.state.ApplyInput(ev.PlayerID, ev.Input)
	case EventSkill:
		r.state.ApplySkill(ev.PlayerID, ev.Skill)
	}
}

func (r *Room) broadcastSnapshot() {
	snap := r.state.Snapshot(r.id)
	for _, pid := range r.players {
		_ = r.sender.Send(pid, protocol.MsgRoomSnapshot, snap)
	}
}

func (r *Room) broadcastRoomOver(winner string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := "settle:" + r.matchID
	ok, err := r.idem.SetIfNotExists(ctx, key, 5*time.Minute)
	if err != nil {
		r.log.Warn("idempotent settle failed", zap.Error(err), zap.String("room", r.id))
		ok = true
	}
	if !ok {
		return
	}

	over := &protocol.RoomOver{
		RoomId:   r.id,
		WinnerId: winner,
	}
	for _, pid := range r.players {
		_ = r.sender.Send(pid, protocol.MsgRoomOver, over)
	}
}

func (r *Room) closeRoom() {
	if r.onClose != nil {
		r.onClose(r.id, r.players)
	}
}
