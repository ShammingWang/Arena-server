package room

import "miniarena/pkg/protocol"

type EventType int

const (
	EventJoin EventType = iota
	EventLeave
	EventInput
	EventSkill
)

type Event struct {
	Type     EventType
	PlayerID string
	Input    *protocol.PlayerInput
	Skill    *protocol.SkillCast
}
