package battle

import (
	"math"

	"miniarena/pkg/protocol"
)

const (
	defaultHP      = 100
	arenaMin       = -100.0
	arenaMax       = 100.0
	skillDamage    = 10
	skillCooldown  = 20
	skillRange     = 20.0
	maxMovePerTick = 5.0
)

// State holds the mutable battle state.
type State struct {
	Players map[string]*PlayerState
	Tick    int64
}

// PlayerState is the authoritative server state for a player.
type PlayerState struct {
	ID      string
	X       float32
	Y       float32
	HP      int32
	SkillCD int32
}

func NewState(playerIDs []string) *State {
	players := make(map[string]*PlayerState, len(playerIDs))
	for i, id := range playerIDs {
		players[id] = &PlayerState{
			ID: id,
			X:  float32(-50 + i*100),
			Y:  0,
			HP: defaultHP,
		}
	}
	return &State{Players: players}
}

func (s *State) ApplyInput(playerID string, input *protocol.PlayerInput) {
	p := s.Players[playerID]
	if p == nil || p.HP <= 0 || input == nil {
		return
	}

	dx := clampFloat32(input.Dx, -maxMovePerTick, maxMovePerTick)
	dy := clampFloat32(input.Dy, -maxMovePerTick, maxMovePerTick)

	p.X = clampFloat32(p.X+dx, arenaMin, arenaMax)
	p.Y = clampFloat32(p.Y+dy, arenaMin, arenaMax)
}

func (s *State) ApplySkill(casterID string, skill *protocol.SkillCast) {
	caster := s.Players[casterID]
	if caster == nil || caster.HP <= 0 || skill == nil {
		return
	}
	if caster.SkillCD > 0 {
		return
	}
	if skill.TargetId == "" || skill.TargetId == casterID {
		return
	}

	target := s.Players[skill.TargetId]
	if target == nil || target.HP <= 0 {
		return
	}

	if distance(caster.X, caster.Y, target.X, target.Y) > skillRange {
		return
	}

	target.HP -= skillDamage
	if target.HP < 0 {
		target.HP = 0
	}
	caster.SkillCD = skillCooldown
}

func (s *State) TickForward() {
	s.Tick++
	for _, p := range s.Players {
		if p.SkillCD > 0 {
			p.SkillCD--
		}
	}
}

func (s *State) Snapshot(roomID string) *protocol.RoomSnapshot {
	players := make([]*protocol.PlayerSnapshot, 0, len(s.Players))
	for _, p := range s.Players {
		players = append(players, &protocol.PlayerSnapshot{
			PlayerId: p.ID,
			X:        p.X,
			Y:        p.Y,
			Hp:       p.HP,
			SkillCd:  p.SkillCD,
		})
	}
	return &protocol.RoomSnapshot{
		RoomId:  roomID,
		Tick:    s.Tick,
		Players: players,
	}
}

func (s *State) Winner() (string, bool) {
	alive := ""
	count := 0
	for _, p := range s.Players {
		if p.HP > 0 {
			count++
			alive = p.ID
			if count > 1 {
				return "", false
			}
		}
	}
	if count == 0 {
		return "", true
	}
	return alive, true
}

func distance(x1, y1, x2, y2 float32) float64 {
	dx := float64(x1 - x2)
	dy := float64(y1 - y2)
	return math.Sqrt(dx*dx + dy*dy)
}

func clampFloat32(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
