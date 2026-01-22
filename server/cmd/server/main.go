package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"miniarena/server/internal/app"
	"miniarena/server/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	application, err := app.New(cfg)
	if err != nil {
		panic(err)
	}

	go func() {
		if err := application.Run(); err != nil {
			panic(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = application.Shutdown(ctx)
}
