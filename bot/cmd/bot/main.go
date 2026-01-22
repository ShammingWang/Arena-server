package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"

	"miniarena/pkg/protocol"
)

type Stats struct {
	connected int64
	matched   int64
	errors    int64
	snaps     int64
}

type RoomTracker struct {
	limit  int
	mu     sync.Mutex
	active map[string]struct{}
	cond   *sync.Cond
}

func NewRoomTracker(limit int) *RoomTracker {
	t := &RoomTracker{
		limit:  limit,
		active: make(map[string]struct{}),
	}
	t.cond = sync.NewCond(&t.mu)
	return t
}

func (t *RoomTracker) WaitForSlot() {
	if t.limit <= 0 {
		return
	}
	t.mu.Lock()
	for len(t.active) >= t.limit {
		t.cond.Wait()
	}
	t.mu.Unlock()
}

func (t *RoomTracker) OnRoomStart(roomID string) {
	if t.limit <= 0 {
		return
	}
	t.mu.Lock()
	if _, ok := t.active[roomID]; !ok {
		t.active[roomID] = struct{}{}
	}
	t.mu.Unlock()
}

func (t *RoomTracker) OnRoomEnd(roomID string) {
	if t.limit <= 0 {
		return
	}
	t.mu.Lock()
	if _, ok := t.active[roomID]; ok {
		delete(t.active, roomID)
		t.cond.Broadcast()
	}
	t.mu.Unlock()
}

type Bot struct {
	id         int
	addr       string
	conn       *websocket.Conn
	send       chan []byte
	stats      *Stats
	tracker    *RoomTracker
	mode       string
	mu         sync.RWMutex
	playerID   string
	roomID     string
	players    []string
	matchStart time.Time
	rng        *rand.Rand
}

func main() {
	addr := flag.String("addr", "ws://127.0.0.1:8080/ws", "websocket address")
	bots := flag.Int("bots", 100, "number of bots")
	rooms := flag.Int("rooms", 50, "expected rooms")
	mode := flag.String("mode", "mixed", "mode: move|skillspam|mixed")
	flag.Parse()
	tracker := NewRoomTracker(*rooms)

	stats := &Stats{}

	for i := 0; i < *bots; i++ {
		go func(id int) {
			bot := NewBot(id, *addr, *mode, stats, tracker)
			if err := bot.Run(); err != nil {
				atomic.AddInt64(&stats.errors, 1)
			}
		}(i)
		time.Sleep(5 * time.Millisecond)
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		fmt.Printf("connected=%d matched=%d snapshots=%d errors=%d\n",
			atomic.LoadInt64(&stats.connected),
			atomic.LoadInt64(&stats.matched),
			atomic.LoadInt64(&stats.snaps),
			atomic.LoadInt64(&stats.errors),
		)
	}
}

func NewBot(id int, addr string, mode string, stats *Stats, tracker *RoomTracker) *Bot {
	return &Bot{
		id:      id,
		addr:    addr,
		send:    make(chan []byte, 128),
		stats:   stats,
		tracker: tracker,
		mode:    mode,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano() + int64(id))),
	}
}

func (b *Bot) Run() error {
	headers := http.Header{}
	conn, _, err := websocket.DefaultDialer.Dial(os.ExpandEnv(b.addr), headers)
	if err != nil {
		return err
	}
	b.conn = conn
	atomic.AddInt64(&b.stats.connected, 1)

	go b.writeLoop()
	go b.actionLoop()

	if err := b.sendLogin(); err != nil {
		return err
	}

	for {
		_, data, err := b.conn.ReadMessage()
		if err != nil {
			return err
		}
		b.handleMessage(data)
	}
}

func (b *Bot) sendLogin() error {
	name := fmt.Sprintf("bot-%d", b.id)
	return b.send(protocol.MsgLoginReq, &protocol.LoginReq{Username: name})
}

func (b *Bot) sendMatch() error {
	if b.tracker != nil {
		b.tracker.WaitForSlot()
	}
	return b.send(protocol.MsgMatchReq, &protocol.MatchReq{Mode: "default"})
}

func (b *Bot) send(msgType protocol.MsgType, msg proto.Message) error {
	payload, err := protocol.Encode(msgType, msg, 0)
	if err != nil {
		return err
	}
	select {
	case b.send <- payload:
		return nil
	default:
		return nil
	}
}

func (b *Bot) handleMessage(data []byte) {
	msgType, _, msg, err := protocol.DecodeMessage(data)
	if err != nil {
		atomic.AddInt64(&b.stats.errors, 1)
		return
	}

	switch msgType {
	case protocol.MsgLoginResp:
		resp := msg.(*protocol.LoginResp)
		b.mu.Lock()
		b.playerID = resp.PlayerId
		b.mu.Unlock()
		_ = b.sendMatch()
	case protocol.MsgMatchResp:
		atomic.AddInt64(&b.stats.matched, 1)
		resp := msg.(*protocol.MatchResp)
		b.mu.Lock()
		b.roomID = resp.RoomId
		b.matchStart = time.Now()
		b.players = resp.Players
		b.mu.Unlock()
		if b.tracker != nil {
			b.tracker.OnRoomStart(resp.RoomId)
		}
	case protocol.MsgRoomSnapshot:
		atomic.AddInt64(&b.stats.snaps, 1)
		snap := msg.(*protocol.RoomSnapshot)
		ids := make([]string, 0, len(snap.Players))
		for _, p := range snap.Players {
			ids = append(ids, p.PlayerId)
		}
		b.mu.Lock()
		b.players = ids
		b.mu.Unlock()
	case protocol.MsgRoomOver:
		var roomID string
		b.mu.RLock()
		roomID = b.roomID
		b.mu.RUnlock()
		if b.tracker != nil {
			b.tracker.OnRoomEnd(roomID)
		}
		b.mu.Lock()
		b.roomID = ""
		b.mu.Unlock()
		_ = b.sendMatch()
	case protocol.MsgErrorResp:
		atomic.AddInt64(&b.stats.errors, 1)
	}
}

func (b *Bot) writeLoop() {
	for data := range b.send {
		_ = b.conn.WriteMessage(websocket.BinaryMessage, data)
	}
}

func (b *Bot) actionLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		b.mu.RLock()
		roomID := b.roomID
		players := append([]string(nil), b.players...)
		self := b.playerID
		b.mu.RUnlock()

		if roomID == "" || self == "" {
			continue
		}

		switch b.mode {
		case "move":
			_ = b.send(protocol.MsgPlayerInput, &protocol.PlayerInput{Dx: randRange(b.rng, -2, 2), Dy: randRange(b.rng, -2, 2)})
		case "skillspam":
			if target := pickTarget(players, self, b.rng); target != "" {
				_ = b.send(protocol.MsgSkillCast, &protocol.SkillCast{SkillId: 1, TargetId: target})
			}
		default:
			_ = b.send(protocol.MsgPlayerInput, &protocol.PlayerInput{Dx: randRange(b.rng, -2, 2), Dy: randRange(b.rng, -2, 2)})
			if b.rng.Intn(4) == 0 {
				if target := pickTarget(players, self, b.rng); target != "" {
					_ = b.send(protocol.MsgSkillCast, &protocol.SkillCast{SkillId: 1, TargetId: target})
				}
			}
		}
	}
}

func pickTarget(players []string, self string, rng *rand.Rand) string {
	if len(players) == 0 {
		return ""
	}
	for i := 0; i < 3; i++ {
		p := players[rng.Intn(len(players))]
		if p != self {
			return p
		}
	}
	return ""
}

func randRange(rng *rand.Rand, min, max float32) float32 {
	return min + rng.Float32()*(max-min)
}
