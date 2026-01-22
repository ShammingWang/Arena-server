package app

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	"miniarena/server/internal/auth"
	"miniarena/server/internal/config"
	"miniarena/server/internal/match"
	"miniarena/server/internal/metrics"
	"miniarena/server/internal/netws"
	"miniarena/server/internal/room"
	"miniarena/server/internal/session"
	"miniarena/server/internal/store"
)

type App struct {
	cfg        config.Config
	log        *zap.Logger
	store      *store.Store
	metrics    *metrics.Metrics
	sessions   *session.Manager
	rooms      *room.Manager
	matcher    *match.Matcher
	netServer  *netws.Server
	httpServer *http.Server
}

func New(cfg config.Config) (*App, error) {
	log, err := newLogger(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	metricsSrv := metrics.NewMetrics()
	storeSrv, err := store.NewStore(cfg, log)
	if err != nil {
		return nil, err
	}

	sessions := session.NewManager(cfg.ReconnectTTL, metricsSrv, log)
	authMgr := auth.NewManager(cfg.JWTSecret)
	rooms := room.NewManager(time.Duration(cfg.TickMS)*time.Millisecond, sessions, storeSrv.Idem, metricsSrv, log, func(roomID string, players []string) {
		for _, pid := range players {
			sessions.SetRoom(pid, "")
		}
	})
	matcher := match.NewMatcher(cfg.PlayersPerRoom, cfg.MatchQueueSize, rooms, sessions, metricsSrv, log)

	netServer := netws.NewServer(cfg, log, metricsSrv, authMgr, sessions, matcher, rooms)

	mux := http.NewServeMux()
	mux.Handle("/ws", netServer)
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	httpServer := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &App{
		cfg:        cfg,
		log:        log,
		store:      storeSrv,
		metrics:    metricsSrv,
		sessions:   sessions,
		rooms:      rooms,
		matcher:    matcher,
		netServer:  netServer,
		httpServer: httpServer,
	}, nil
}

func (a *App) Run() error {
	a.log.Info("server start", zap.String("addr", a.cfg.HTTPAddr))
	err := a.httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (a *App) Shutdown(ctx context.Context) error {
	a.store.Close()
	return a.httpServer.Shutdown(ctx)
}

func newLogger(level string) (*zap.Logger, error) {
	if level == "debug" {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}
