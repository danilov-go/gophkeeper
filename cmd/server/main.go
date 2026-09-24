package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/danilov-go/gophkeeper/internal/config"
	"github.com/danilov-go/gophkeeper/internal/logger"
	"github.com/danilov-go/gophkeeper/internal/server"
	"github.com/go-chi/chi"
	"golang.org/x/sync/errgroup"
)

func main() {
	cfg := &config.ConfigServer{}
	cfg.Net.Set("localhost:8080")
	if err := logger.Initialize("info"); err != nil {
		panic(err)
	}
	if err := cfg.Get(); err != nil {
		logger.Log.Sugar().Fatal("ошибка загрузки конфигурации сервера", "error", err)
	}
	r := chi.NewRouter()
	serv := server.New(cfg.Net.String(), logger.Log.Sugar(), r)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return serv.Run()
	})
	g.Go(func() error {
		<-gCtx.Done()
		return serv.Stop()
	})
	if err := g.Wait(); err != nil {
		logger.Log.Sugar().Fatal("сервер аварийно завершил работу", "err", err)
	}
}
