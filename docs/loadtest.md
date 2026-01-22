# Load Test

## Local run

1) Start server

```
go run ./server/cmd/server
```

2) Start bots

```
go run ./bot/cmd/bot --addr ws://127.0.0.1:8080/ws --bots 1000 --rooms 500 --mode mixed
```

`--rooms` limits the number of concurrent active rooms (useful for stable pressure).

## Metrics

Prometheus endpoint: `http://localhost:8080/metrics`

Key metrics:

- `arena_sessions_online_total`
- `arena_match_queue_total`
- `arena_match_duration_ms_bucket`
- `arena_room_tick_delay_ms_bucket`
- `arena_net_send_bytes_total`
- `arena_net_recv_bytes_total`
- `arena_net_dropped_messages_total`

## Example (placeholder)

- 1k bots: P99 tick delay < 10ms on laptop
- 5k bots: P99 tick delay < 30ms on laptop

Replace with your own machine measurements.
