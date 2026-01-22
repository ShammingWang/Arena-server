# Protocol

Transport: WebSocket (binary). Payloads are protobuf-like messages encoded with `gogo/protobuf` tags.
Schema reference: `api/arena.proto`.

## Envelope

All messages are wrapped in an `Envelope`:

- `type` (enum MsgType)
- `seq` (uint64)
- `body` (bytes)
- `version` (int32, current = 1)

## MsgType

- 1 PING / 2 PONG
- 10 LOGIN_REQ / 11 LOGIN_RESP
- 12 RECONNECT_REQ / 13 RECONNECT_RESP
- 20 MATCH_REQ / 21 MATCH_RESP
- 30 PLAYER_INPUT / 31 SKILL_CAST
- 40 ROOM_SNAPSHOT / 41 ROOM_OVER
- 90 ERROR_RESP

## Login

- `LoginReq { username }`
- `LoginResp { player_id, access_token, reconnect_token }`

## Match

- `MatchReq { mode }`
- `MatchResp { match_id, room_id, players[] }`

## Gameplay

- `PlayerInput { dx, dy }`
- `SkillCast { skill_id, target_id }`
- `RoomSnapshot { room_id, tick, players[] }`
  - `PlayerSnapshot { player_id, x, y, hp, skill_cd }`
- `RoomOver { room_id, winner_id }`

## Error

- `ErrorResp { code, message }`
