package protocol

import (
	"fmt"
)

// MsgType defines top-level envelope message types.
type MsgType int32

const (
	MsgUnknown       MsgType = 0
	MsgPing          MsgType = 1
	MsgPong          MsgType = 2
	MsgLoginReq      MsgType = 10
	MsgLoginResp     MsgType = 11
	MsgReconnectReq  MsgType = 12
	MsgReconnectResp MsgType = 13
	MsgMatchReq      MsgType = 20
	MsgMatchResp     MsgType = 21
	MsgPlayerInput   MsgType = 30
	MsgSkillCast     MsgType = 31
	MsgRoomSnapshot  MsgType = 40
	MsgRoomOver      MsgType = 41
	MsgErrorResp     MsgType = 90
)

const CurrentVersion = 1

func (t MsgType) String() string {
	switch t {
	case MsgPing:
		return "PING"
	case MsgPong:
		return "PONG"
	case MsgLoginReq:
		return "LOGIN_REQ"
	case MsgLoginResp:
		return "LOGIN_RESP"
	case MsgReconnectReq:
		return "RECONNECT_REQ"
	case MsgReconnectResp:
		return "RECONNECT_RESP"
	case MsgMatchReq:
		return "MATCH_REQ"
	case MsgMatchResp:
		return "MATCH_RESP"
	case MsgPlayerInput:
		return "PLAYER_INPUT"
	case MsgSkillCast:
		return "SKILL_CAST"
	case MsgRoomSnapshot:
		return "ROOM_SNAPSHOT"
	case MsgRoomOver:
		return "ROOM_OVER"
	case MsgErrorResp:
		return "ERROR_RESP"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", t)
	}
}

// Envelope wraps all payloads to allow a single decoder.
type Envelope struct {
	Type    MsgType `protobuf:"varint,1,opt,name=type,proto3,enum=protocol.MsgType" json:"type,omitempty"`
	Seq     uint64  `protobuf:"varint,2,opt,name=seq,proto3" json:"seq,omitempty"`
	Body    []byte  `protobuf:"bytes,3,opt,name=body,proto3" json:"body,omitempty"`
	Version int32   `protobuf:"varint,4,opt,name=version,proto3" json:"version,omitempty"`
}

func (m *Envelope) Reset()         { *m = Envelope{} }
func (m *Envelope) String() string { return "Envelope" }
func (*Envelope) ProtoMessage()    {}

// Ping/Pong

type Ping struct {
	ClientTs int64 `protobuf:"varint,1,opt,name=client_ts,json=clientTs,proto3" json:"client_ts,omitempty"`
}

func (m *Ping) Reset()         { *m = Ping{} }
func (m *Ping) String() string { return "Ping" }
func (*Ping) ProtoMessage()    {}

type Pong struct {
	ClientTs int64 `protobuf:"varint,1,opt,name=client_ts,json=clientTs,proto3" json:"client_ts,omitempty"`
	ServerTs int64 `protobuf:"varint,2,opt,name=server_ts,json=serverTs,proto3" json:"server_ts,omitempty"`
}

func (m *Pong) Reset()         { *m = Pong{} }
func (m *Pong) String() string { return "Pong" }
func (*Pong) ProtoMessage()    {}

// Login

type LoginReq struct {
	Username string `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
}

func (m *LoginReq) Reset()         { *m = LoginReq{} }
func (m *LoginReq) String() string { return "LoginReq" }
func (*LoginReq) ProtoMessage()    {}

type LoginResp struct {
	PlayerId       string `protobuf:"bytes,1,opt,name=player_id,json=playerId,proto3" json:"player_id,omitempty"`
	AccessToken    string `protobuf:"bytes,2,opt,name=access_token,json=accessToken,proto3" json:"access_token,omitempty"`
	ReconnectToken string `protobuf:"bytes,3,opt,name=reconnect_token,json=reconnectToken,proto3" json:"reconnect_token,omitempty"`
}

func (m *LoginResp) Reset()         { *m = LoginResp{} }
func (m *LoginResp) String() string { return "LoginResp" }
func (*LoginResp) ProtoMessage()    {}

type ReconnectReq struct {
	ReconnectToken string `protobuf:"bytes,1,opt,name=reconnect_token,json=reconnectToken,proto3" json:"reconnect_token,omitempty"`
}

func (m *ReconnectReq) Reset()         { *m = ReconnectReq{} }
func (m *ReconnectReq) String() string { return "ReconnectReq" }
func (*ReconnectReq) ProtoMessage()    {}

type ReconnectResp struct {
	PlayerId string `protobuf:"bytes,1,opt,name=player_id,json=playerId,proto3" json:"player_id,omitempty"`
	RoomId   string `protobuf:"bytes,2,opt,name=room_id,json=roomId,proto3" json:"room_id,omitempty"`
	Ok       bool   `protobuf:"varint,3,opt,name=ok,proto3" json:"ok,omitempty"`
	Reason   string `protobuf:"bytes,4,opt,name=reason,proto3" json:"reason,omitempty"`
}

func (m *ReconnectResp) Reset()         { *m = ReconnectResp{} }
func (m *ReconnectResp) String() string { return "ReconnectResp" }
func (*ReconnectResp) ProtoMessage()    {}

// Match

type MatchReq struct {
	Mode string `protobuf:"bytes,1,opt,name=mode,proto3" json:"mode,omitempty"`
}

func (m *MatchReq) Reset()         { *m = MatchReq{} }
func (m *MatchReq) String() string { return "MatchReq" }
func (*MatchReq) ProtoMessage()    {}

type MatchResp struct {
	MatchId string   `protobuf:"bytes,1,opt,name=match_id,json=matchId,proto3" json:"match_id,omitempty"`
	RoomId  string   `protobuf:"bytes,2,opt,name=room_id,json=roomId,proto3" json:"room_id,omitempty"`
	Players []string `protobuf:"bytes,3,rep,name=players,proto3" json:"players,omitempty"`
}

func (m *MatchResp) Reset()         { *m = MatchResp{} }
func (m *MatchResp) String() string { return "MatchResp" }
func (*MatchResp) ProtoMessage()    {}

// Player input

type PlayerInput struct {
	Dx float32 `protobuf:"fixed32,1,opt,name=dx,proto3" json:"dx,omitempty"`
	Dy float32 `protobuf:"fixed32,2,opt,name=dy,proto3" json:"dy,omitempty"`
}

func (m *PlayerInput) Reset()         { *m = PlayerInput{} }
func (m *PlayerInput) String() string { return "PlayerInput" }
func (*PlayerInput) ProtoMessage()    {}

// Skill cast

type SkillCast struct {
	SkillId  int32  `protobuf:"varint,1,opt,name=skill_id,json=skillId,proto3" json:"skill_id,omitempty"`
	TargetId string `protobuf:"bytes,2,opt,name=target_id,json=targetId,proto3" json:"target_id,omitempty"`
}

func (m *SkillCast) Reset()         { *m = SkillCast{} }
func (m *SkillCast) String() string { return "SkillCast" }
func (*SkillCast) ProtoMessage()    {}

// Snapshot

type PlayerSnapshot struct {
	PlayerId string  `protobuf:"bytes,1,opt,name=player_id,json=playerId,proto3" json:"player_id,omitempty"`
	X        float32 `protobuf:"fixed32,2,opt,name=x,proto3" json:"x,omitempty"`
	Y        float32 `protobuf:"fixed32,3,opt,name=y,proto3" json:"y,omitempty"`
	Hp       int32   `protobuf:"varint,4,opt,name=hp,proto3" json:"hp,omitempty"`
	SkillCd  int32   `protobuf:"varint,5,opt,name=skill_cd,json=skillCd,proto3" json:"skill_cd,omitempty"`
}

func (m *PlayerSnapshot) Reset()         { *m = PlayerSnapshot{} }
func (m *PlayerSnapshot) String() string { return "PlayerSnapshot" }
func (*PlayerSnapshot) ProtoMessage()    {}

type RoomSnapshot struct {
	RoomId  string             `protobuf:"bytes,1,opt,name=room_id,json=roomId,proto3" json:"room_id,omitempty"`
	Tick    int64              `protobuf:"varint,2,opt,name=tick,proto3" json:"tick,omitempty"`
	Players []*PlayerSnapshot  `protobuf:"bytes,3,rep,name=players,proto3" json:"players,omitempty"`
}

func (m *RoomSnapshot) Reset()         { *m = RoomSnapshot{} }
func (m *RoomSnapshot) String() string { return "RoomSnapshot" }
func (*RoomSnapshot) ProtoMessage()    {}

// Room over

type RoomOver struct {
	RoomId   string `protobuf:"bytes,1,opt,name=room_id,json=roomId,proto3" json:"room_id,omitempty"`
	WinnerId string `protobuf:"bytes,2,opt,name=winner_id,json=winnerId,proto3" json:"winner_id,omitempty"`
}

func (m *RoomOver) Reset()         { *m = RoomOver{} }
func (m *RoomOver) String() string { return "RoomOver" }
func (*RoomOver) ProtoMessage()    {}

// Error

type ErrorResp struct {
	Code    int32  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

func (m *ErrorResp) Reset()         { *m = ErrorResp{} }
func (m *ErrorResp) String() string { return "ErrorResp" }
func (*ErrorResp) ProtoMessage()    {}
