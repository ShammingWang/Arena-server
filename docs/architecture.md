# Architecture

MiniArena is built around a room-actor model to reduce contention and make behavior predictable.

```
                   +-----------------------+
                   |  HTTP Server (Go)    |
                   |  /ws /metrics        |
                   +----------+------------+
                              |
                        WebSocket
                              |
+-----------+    +-------------v-------------+     +------------------+
|   Bots    |--> | netws.Server + SessionMgr | --> | Matchmaker Queue |
+-----------+    +-------------+-------------+     +---------+--------+
                              |                             |
                              |                             v
                              |                     +----------------+
                              |                     | Room Manager   |
                              |                     +--------+-------+
                              |                              |
                              v                              v
                     +-----------------+            +-------------------+
                     | Room Actor (1G) | <--------- | Battle State      |
                     | tick + events   |            | apply input/skill |
                     +-----------------+            +-------------------+
```

## Key points

- Room actor: each room runs in a single goroutine and serializes events (input/skill/leave) on a channel.
- Tick loop: 50ms ticker drives snapshot broadcast and cooldown updates.
- Network goroutines only parse messages and enqueue events; they do not mutate room state.
- Match queue is managed by a single goroutine to avoid shared-state locking.
- Idempotent settlement uses Redis SETNX (fallback to in-memory map for local runs).
- Redis/MySQL are wired and optional; the minimal demo runs without them.

## Data flow

1) Client connects to `/ws` and sends LoginReq.
2) Session is created, tokens are returned.
3) Client sends MatchReq; matcher groups players and creates a room.
4) Room actor ticks every 50ms, applies inputs, and broadcasts snapshots.
5) When only one (or zero) players remain alive, room ends and broadcasts RoomOver.
