# MiniArena: Multiplayer Room Battle Server

MiniArena is a Go-based multiplayer room battle service with matchmaking, room actor loops, snapshot sync, and bot load testing.

## Layout

```
/ server
  /cmd/server          Entry point
  /internal/...        Server packages
/ bot
  /cmd/bot             Bot load tester
/ pkg/protocol         Shared protobuf-like protocol
/ deploy               docker-compose + prometheus
/ docs                 architecture, protocol, loadtest
```

## Quick start

1) Start dependencies

```
cd deploy

docker compose up -d
```

2) Run server

```
go run ./server/cmd/server
```

3) Run bots (100 clients, 50 rooms)

```
go run ./bot/cmd/bot --addr ws://127.0.0.1:8080/ws --bots 100 --rooms 50 --mode mixed
```

## Environment

Server reads env with prefix `ARENA_`:

- `ARENA_HTTP_ADDR` (default `:8080`)
- `ARENA_JWT_SECRET` (default `dev-secret`)
- `ARENA_REDIS_ADDR` (default `127.0.0.1:6379`)
- `ARENA_MYSQL_DSN` (default empty)
- `ARENA_TICK_MS` (default `50`)
- `ARENA_PLAYERS_PER_ROOM` (default `2`)
- `ARENA_RECONNECT_TTL_SEC` (default `30`)

## Docs

- `docs/architecture.md`
- `docs/protocol.md`
- `docs/loadtest.md`
- Protocol schema: `api/arena.proto`, runtime codec: `pkg/protocol`
