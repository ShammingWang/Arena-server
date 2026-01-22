package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	OnlineGauge      prometheus.Gauge
	MatchQueueGauge  prometheus.Gauge
	MatchDuration    prometheus.Histogram
	RoomTickDelay    prometheus.Histogram
	SendBytes        prometheus.Counter
	RecvBytes        prometheus.Counter
	DroppedMessages  prometheus.Counter
}

func NewMetrics() *Metrics {
	m := &Metrics{
		OnlineGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "arena",
			Subsystem: "sessions",
			Name:      "online_total",
			Help:      "Online sessions",
		}),
		MatchQueueGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "arena",
			Subsystem: "match",
			Name:      "queue_total",
			Help:      "Players waiting in match queue",
		}),
		MatchDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "arena",
			Subsystem: "match",
			Name:      "duration_ms",
			Help:      "Matchmaking duration in ms",
			Buckets:   []float64{50, 100, 200, 500, 1000, 2000, 5000},
		}),
		RoomTickDelay: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "arena",
			Subsystem: "room",
			Name:      "tick_delay_ms",
			Help:      "Room tick delay in ms",
			Buckets:   []float64{1, 2, 5, 10, 20, 50, 100},
		}),
		SendBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "arena",
			Subsystem: "net",
			Name:      "send_bytes_total",
			Help:      "Total outbound bytes",
		}),
		RecvBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "arena",
			Subsystem: "net",
			Name:      "recv_bytes_total",
			Help:      "Total inbound bytes",
		}),
		DroppedMessages: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "arena",
			Subsystem: "net",
			Name:      "dropped_messages_total",
			Help:      "Dropped outbound messages due to backpressure",
		}),
	}

	prometheus.MustRegister(
		m.OnlineGauge,
		m.MatchQueueGauge,
		m.MatchDuration,
		m.RoomTickDelay,
		m.SendBytes,
		m.RecvBytes,
		m.DroppedMessages,
	)

	return m
}

func Handler() http.Handler {
	return promhttp.Handler()
}
