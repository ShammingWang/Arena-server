package netws

import (
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"miniarena/server/internal/metrics"
)

var ErrSendQueueFull = errors.New("send queue full")

const (
	writeWait  = 5 * time.Second
	pongWait   = 20 * time.Second
	pingPeriod = 10 * time.Second
)

type Client struct {
	conn       *websocket.Conn
	send       chan []byte
	metrics    *metrics.Metrics
	log        *zap.Logger
	readLimit  int64
	mu         sync.RWMutex
	playerID   string
	winMu      sync.Mutex
	winStart   time.Time
	winCount   int
	closeOnce  sync.Once
}

func NewClient(conn *websocket.Conn, sendQueue int, readLimit int64, metrics *metrics.Metrics, log *zap.Logger) *Client {
	return &Client{
		conn:      conn,
		send:      make(chan []byte, sendQueue),
		metrics:   metrics,
		log:       log,
		readLimit: readLimit,
		winStart:  time.Now(),
	}
}

func (c *Client) SetPlayerID(id string) {
	c.mu.Lock()
	c.playerID = id
	c.mu.Unlock()
}

func (c *Client) PlayerID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.playerID
}

func (c *Client) AllowMessage(maxPerSecond int) bool {
	if maxPerSecond <= 0 {
		return true
	}
	c.winMu.Lock()
	defer c.winMu.Unlock()

	now := time.Now()
	if now.Sub(c.winStart) >= time.Second {
		c.winStart = now
		c.winCount = 0
	}
	c.winCount++
	return c.winCount <= maxPerSecond
}

func (c *Client) Send(data []byte) error {
	select {
	case c.send <- data:
		if c.metrics != nil {
			c.metrics.SendBytes.Add(float64(len(data)))
		}
		return nil
	default:
		if c.metrics != nil {
			c.metrics.DroppedMessages.Inc()
		}
		return ErrSendQueueFull
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) CloseSend() {
	c.closeOnce.Do(func() {
		close(c.send)
	})
}

func (c *Client) ReadLoop(handle func([]byte)) {
	c.conn.SetReadLimit(c.readLimit)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		if msgType != websocket.BinaryMessage {
			continue
		}
		if c.metrics != nil {
			c.metrics.RecvBytes.Add(float64(len(data)))
		}
		handle(data)
	}
}

func (c *Client) WriteLoop() {
	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

	for {
		select {
		case data, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return
			}
		case <-pingTicker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
