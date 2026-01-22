package protocol

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
)

// Encode wraps a payload into an Envelope and marshals it.
func Encode(msgType MsgType, body proto.Message, seq uint64) ([]byte, error) {
	var raw []byte
	var err error
	if body != nil {
		raw, err = proto.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	env := &Envelope{
		Type:    msgType,
		Seq:     seq,
		Body:    raw,
		Version: CurrentVersion,
	}
	return proto.Marshal(env)
}

// DecodeEnvelope unmarshals an Envelope from raw bytes.
func DecodeEnvelope(data []byte) (*Envelope, error) {
	var env Envelope
	if err := proto.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// DecodeMessage unmarshals Envelope and payload into a concrete type.
func DecodeMessage(data []byte) (MsgType, uint64, proto.Message, error) {
	env, err := DecodeEnvelope(data)
	if err != nil {
		return MsgUnknown, 0, nil, err
	}
	msg, err := UnmarshalBody(env.Type, env.Body)
	if err != nil {
		return env.Type, env.Seq, nil, err
	}
	return env.Type, env.Seq, msg, nil
}

// UnmarshalBody decodes the body according to MsgType.
func UnmarshalBody(msgType MsgType, body []byte) (proto.Message, error) {
	switch msgType {
	case MsgPing:
		var m Ping
		return &m, proto.Unmarshal(body, &m)
	case MsgPong:
		var m Pong
		return &m, proto.Unmarshal(body, &m)
	case MsgLoginReq:
		var m LoginReq
		return &m, proto.Unmarshal(body, &m)
	case MsgLoginResp:
		var m LoginResp
		return &m, proto.Unmarshal(body, &m)
	case MsgReconnectReq:
		var m ReconnectReq
		return &m, proto.Unmarshal(body, &m)
	case MsgReconnectResp:
		var m ReconnectResp
		return &m, proto.Unmarshal(body, &m)
	case MsgMatchReq:
		var m MatchReq
		return &m, proto.Unmarshal(body, &m)
	case MsgMatchResp:
		var m MatchResp
		return &m, proto.Unmarshal(body, &m)
	case MsgPlayerInput:
		var m PlayerInput
		return &m, proto.Unmarshal(body, &m)
	case MsgSkillCast:
		var m SkillCast
		return &m, proto.Unmarshal(body, &m)
	case MsgRoomSnapshot:
		var m RoomSnapshot
		return &m, proto.Unmarshal(body, &m)
	case MsgRoomOver:
		var m RoomOver
		return &m, proto.Unmarshal(body, &m)
	case MsgErrorResp:
		var m ErrorResp
		return &m, proto.Unmarshal(body, &m)
	default:
		return nil, fmt.Errorf("unknown msg type: %d", msgType)
	}
}
