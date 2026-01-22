package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	HTTPAddr         string
	JWTSecret        string
	RedisAddr        string
	RedisPassword    string
	RedisDB          int
	MySQLDSN         string
	TickMS           int
	PlayersPerRoom   int
	ReconnectTTL     time.Duration
	LogLevel         string
	SendQueueSize    int
	ReadLimitBytes   int64
	MatchQueueSize   int
	MaxMsgPerSecond  int
}

func Load() (Config, error) {
	v := viper.New()
	v.SetEnvPrefix("ARENA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("HTTP_ADDR", ":8080")
	v.SetDefault("JWT_SECRET", "dev-secret")
	v.SetDefault("REDIS_ADDR", "127.0.0.1:6379")
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("MYSQL_DSN", "")
	v.SetDefault("TICK_MS", 50)
	v.SetDefault("PLAYERS_PER_ROOM", 2)
	v.SetDefault("RECONNECT_TTL_SEC", 30)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("SEND_QUEUE_SIZE", 256)
	v.SetDefault("READ_LIMIT_BYTES", 1048576)
	v.SetDefault("MATCH_QUEUE_SIZE", 10240)
	v.SetDefault("MAX_MSG_PER_SECOND", 60)

	cfg := Config{
		HTTPAddr:        v.GetString("HTTP_ADDR"),
		JWTSecret:       v.GetString("JWT_SECRET"),
		RedisAddr:       v.GetString("REDIS_ADDR"),
		RedisPassword:   v.GetString("REDIS_PASSWORD"),
		RedisDB:         v.GetInt("REDIS_DB"),
		MySQLDSN:        v.GetString("MYSQL_DSN"),
		TickMS:          v.GetInt("TICK_MS"),
		PlayersPerRoom:  v.GetInt("PLAYERS_PER_ROOM"),
		ReconnectTTL:    time.Duration(v.GetInt("RECONNECT_TTL_SEC")) * time.Second,
		LogLevel:        v.GetString("LOG_LEVEL"),
		SendQueueSize:   v.GetInt("SEND_QUEUE_SIZE"),
		ReadLimitBytes:  v.GetInt64("READ_LIMIT_BYTES"),
		MatchQueueSize:  v.GetInt("MATCH_QUEUE_SIZE"),
		MaxMsgPerSecond: v.GetInt("MAX_MSG_PER_SECOND"),
	}

	return cfg, nil
}
