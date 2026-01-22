package store

import (
	"context"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"miniarena/server/internal/config"
)

type Store struct {
	Redis *redis.Client
	MySQL *sqlx.DB
	Idem  Idempotency
	log   *zap.Logger
}

func NewStore(cfg config.Config, log *zap.Logger) (*Store, error) {
	s := &Store{log: log}
	if cfg.RedisAddr != "" {
		rdb := redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Warn("redis ping failed", zap.Error(err))
		} else {
			s.Redis = rdb
		}
	}

	if cfg.MySQLDSN != "" {
		db, err := sqlx.Connect("mysql", cfg.MySQLDSN)
		if err != nil {
			log.Warn("mysql connect failed", zap.Error(err))
		} else {
			s.MySQL = db
		}
	}

	if s.Redis != nil {
		s.Idem = NewRedisIdem(s.Redis)
	} else {
		s.Idem = NewMemoryIdem()
	}

	return s, nil
}

func (s *Store) Close() {
	if s.Redis != nil {
		_ = s.Redis.Close()
	}
	if s.MySQL != nil {
		_ = s.MySQL.Close()
	}
}
