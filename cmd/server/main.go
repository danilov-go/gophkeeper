package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/danilov-go/gophkeeper/internal/config"
	"github.com/danilov-go/gophkeeper/internal/logger"
	"github.com/danilov-go/gophkeeper/internal/server/handler"
	server "github.com/danilov-go/gophkeeper/internal/server/http"
	"github.com/danilov-go/gophkeeper/internal/server/repository/memory"
	"github.com/go-chi/chi"
	"golang.org/x/sync/errgroup"
)

func main() {
	cfg := &config.ConfigServer{}
	if err := logger.Initialize("info"); err != nil {
		panic(err)
	}
	if err := cfg.Get(); err != nil {
		logger.Log.Sugar().Fatal("ошибка загрузки конфигурации сервера")
	}
	storage := memory.InitMemStorage()
	h := handler.NewHandlers(storage, logger.Log.Sugar())
	r := chi.NewRouter()
	jwt := "jwtsecrettoken"
	r.Route("/user", func(r chi.Router) {
		r.Post("/register", h.RegisterUser(jwt))
		r.Post("/login", h.LoginUser(jwt))
	})
	r.Route("/api", func(r chi.Router) {
		r.Post("/pull", handler.AuthMiddleware(jwt, h.PullHandler()))
		r.Post("/push", handler.AuthMiddleware(jwt, h.PushHandler()))
	})
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
