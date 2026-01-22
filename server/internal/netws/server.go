package netws

import (
	"net/http"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"miniarena/pkg/protocol"
	"miniarena/server/internal/auth"
	"miniarena/server/internal/config"
	"miniarena/server/internal/match"
	"miniarena/server/internal/metrics"
	"miniarena/server/internal/room"
	"miniarena/server/internal/session"
)

type Server struct {
	cfg      config.Config
	log      *zap.Logger
	metrics  *metrics.Metrics
	upgrader websocket.Upgrader
	auth     *auth.Manager
	sessions *session.Manager
	matcher  *match.Matcher
	rooms    *room.Manager
}

func NewServer(cfg config.Config, log *zap.Logger, metrics *metrics.Metrics, auth *auth.Manager, sessions *session.Manager, matcher *match.Matcher, rooms *room.Manager) *Server {
	return &Server{
		cfg:     cfg,
		log:     log,
		metrics: metrics,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		auth:     auth,
		sessions: sessions,
		matcher:  matcher,
		rooms:    rooms,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(conn, s.cfg.SendQueueSize, s.cfg.ReadLimitBytes, s.metrics, s.log)
	go client.WriteLoop()
	client.ReadLoop(func(data []byte) {
		s.handleMessage(client, data)
	})

	client.CloseSend()
	_ = client.Close()
	if pid := client.PlayerID(); pid != "" {
		if sess, ok := s.sessions.Get(pid); ok {
			if roomID := sess.GetRoomID(); roomID != "" {
				s.rooms.SendEvent(roomID, room.Event{Type: room.EventLeave, PlayerID: pid})
			}
		}
		s.sessions.MarkOffline(pid)
	}
}

func (s *Server) handleMessage(c *Client, data []byte) {
	if !c.AllowMessage(s.cfg.MaxMsgPerSecond) {
		s.sendError(c, 429, "rate limited")
		return
	}

	env, err := protocol.DecodeEnvelope(data)
	if err != nil {
		s.sendError(c, 400, "bad envelope")
		return
	}
	if env.Version != 0 && env.Version != protocol.CurrentVersion {
		s.sendError(c, 426, "protocol version mismatch")
		return
	}

	switch env.Type {
	case protocol.MsgPing:
		req := &protocol.Ping{}
		if err := proto.Unmarshal(env.Body, req); err != nil {
			s.sendError(c, 400, "bad ping")
			return
		}
		pong := &protocol.Pong{ClientTs: req.ClientTs, ServerTs: time.Now().UnixMilli()}
		s.sendDirect(c, protocol.MsgPong, pong)
		return
	case protocol.MsgLoginReq:
		s.handleLogin(c, env.Body)
		return
	case protocol.MsgReconnectReq:
		s.handleReconnect(c, env.Body)
		return
	}

	playerID := c.PlayerID()
	if playerID == "" {
		s.sendError(c, 401, "not logged in")
		return
	}

	switch env.Type {
	case protocol.MsgMatchReq:
		s.handleMatch(playerID)
	case protocol.MsgPlayerInput:
		var input protocol.PlayerInput
		if err := proto.Unmarshal(env.Body, &input); err != nil {
			s.sendError(c, 400, "bad input")
			return
		}
		s.forwardInput(playerID, &input)
	case protocol.MsgSkillCast:
		var skill protocol.SkillCast
		if err := proto.Unmarshal(env.Body, &skill); err != nil {
			s.sendError(c, 400, "bad skill")
			return
		}
		s.forwardSkill(playerID, &skill)
	default:
		s.sendError(c, 400, "unknown message")
	}
}

func (s *Server) handleLogin(c *Client, body []byte) {
	var req protocol.LoginReq
	if err := proto.Unmarshal(body, &req); err != nil {
		s.sendError(c, 400, "bad login")
		return
	}
	if req.Username == "" {
		req.Username = "player-" + uuid.NewString()[:8]
	}

	playerID := uuid.NewString()
	accessToken, _ := s.auth.GenerateAccessToken(playerID, req.Username, 10*time.Minute)
	reconnectToken, _ := s.auth.GenerateReconnectToken(playerID, s.cfg.ReconnectTTL)

	s.sessions.Create(playerID, req.Username, reconnectToken, c)
	c.SetPlayerID(playerID)

	resp := &protocol.LoginResp{
		PlayerId:       playerID,
		AccessToken:    accessToken,
		ReconnectToken: reconnectToken,
	}
	_ = s.sessions.Send(playerID, protocol.MsgLoginResp, resp)
}

func (s *Server) handleReconnect(c *Client, body []byte) {
	var req protocol.ReconnectReq
	if err := proto.Unmarshal(body, &req); err != nil {
		s.sendError(c, 400, "bad reconnect")
		return
	}
	playerID, err := s.auth.ParseReconnectToken(req.ReconnectToken)
	if err != nil {
		s.sendDirect(c, protocol.MsgReconnectResp, &protocol.ReconnectResp{Ok: false, Reason: "invalid token"})
		return
	}

	sess, ok := s.sessions.Bind(playerID, c)
	if !ok {
		s.sendDirect(c, protocol.MsgReconnectResp, &protocol.ReconnectResp{Ok: false, Reason: "session not found"})
		return
	}
	c.SetPlayerID(playerID)

	s.sendDirect(c, protocol.MsgReconnectResp, &protocol.ReconnectResp{
		PlayerId: playerID,
		RoomId:   sess.RoomID,
		Ok:       true,
	})
	if sess.RoomID != "" {
		s.rooms.SendEvent(sess.RoomID, room.Event{Type: room.EventJoin, PlayerID: playerID})
	}
}

func (s *Server) handleMatch(playerID string) {
	ok := s.matcher.Enqueue(playerID)
	if !ok {
		_ = s.sessions.Send(playerID, protocol.MsgErrorResp, &protocol.ErrorResp{Code: 429, Message: "match queue full"})
	}
}

func (s *Server) forwardInput(playerID string, input *protocol.PlayerInput) {
	sess, ok := s.sessions.Get(playerID)
	if !ok {
		return
	}
	roomID := sess.GetRoomID()
	if roomID == "" {
		return
	}
	s.rooms.SendEvent(roomID, room.Event{Type: room.EventInput, PlayerID: playerID, Input: input})
}

func (s *Server) forwardSkill(playerID string, skill *protocol.SkillCast) {
	sess, ok := s.sessions.Get(playerID)
	if !ok {
		return
	}
	roomID := sess.GetRoomID()
	if roomID == "" {
		return
	}
	s.rooms.SendEvent(roomID, room.Event{Type: room.EventSkill, PlayerID: playerID, Skill: skill})
}

func (s *Server) sendError(c *Client, code int32, message string) {
	_ = s.sendDirect(c, protocol.MsgErrorResp, &protocol.ErrorResp{Code: code, Message: message})
}

func (s *Server) sendDirect(c *Client, msgType protocol.MsgType, msg proto.Message) error {
	payload, err := protocol.Encode(msgType, msg, 0)
	if err != nil {
		return err
	}
	return c.Send(payload)
}
