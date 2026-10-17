package main

import (
	"context"
	"net"
	"os/signal"
	"syscall"

	"github.com/danilov-go/gophkeeper/internal/config"
	"github.com/danilov-go/gophkeeper/internal/logger"
	"github.com/danilov-go/gophkeeper/internal/proto"
	"github.com/danilov-go/gophkeeper/internal/server/handler"
	"github.com/danilov-go/gophkeeper/internal/server/repository/db"

	//"github.com/danilov-go/gophkeeper/internal/server/repository/memory"
	"github.com/danilov-go/gophkeeper/internal/server/service"
	"github.com/danilov-go/gophkeeper/internal/server/sync"
	"github.com/danilov-go/gophkeeper/internal/server/transport"
	"github.com/go-chi/chi"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

func main() {
	cfg := &config.ConfigServer{}
	if err := logger.Initialize("info"); err != nil {
		panic(err)
	}
	if err := cfg.Get(); err != nil {
		logger.Log.Sugar().Fatal("ошибка загрузки конфигурации сервера")
	}
	//storage := memory.InitMemStorage()
	db, err := db.InitDB(cfg.DatabaseDSN)
	if err != nil {
		logger.Log.Sugar().Fatalw("ошибка инициализации базы данных", "error", err)
	}
	notifier := sync.NewSyncNotifier()
	jwtKey := "jwtsecrettoken"
	srv := service.NewServerService(db, jwtKey, notifier)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()
	g, gCtx := errgroup.WithContext(ctx)
	if cfg.Protocol == "grpc" {
		listener, err := net.Listen("tcp", cfg.Net.String())
		if err != nil {
			logger.Log.Sugar().Fatal("ошибка запуска listener для gRPC", err)
		}
		gServ := grpc.NewServer(grpc.UnaryInterceptor(handler.AuthInterceptor(srv)))
		h := handler.NewGRPCHandler(srv, logger.Log.Sugar())
		proto.RegisterSyncServiceServer(gServ, h)
		serv := transport.NewGRPC(cfg.Net.String(), logger.Log.Sugar(), gServ, listener)
		g.Go(func() error {
			logger.Log.Sugar().Info("Сервер запущен по протоколу gRPC на ", cfg.Net.String())
			return serv.Run()
		})
		g.Go(func() error {
			<-gCtx.Done()
			logger.Log.Sugar().Info("Остановка gRPC сервера...")
			return serv.Stop()
		})
	} else {
		h := handler.NewHandlers(logger.Log.Sugar(), srv)
		r := chi.NewRouter()
		r.Route("/user", func(r chi.Router) {
			r.Post("/register", h.RegisterUser())
			r.Post("/login", h.LoginUser())
			r.Get("/salt", h.GetSalt())
		})
		r.Route("/api", func(r chi.Router) {
			r.Post("/pull", h.AuthMiddleware(h.PullHandler()))
			r.Post("/push", h.AuthMiddleware(h.PushHandler()))
			r.Get("/event", h.AuthMiddleware(h.StreamHandler()))
		})
		serv := transport.New(cfg.Net.String(), logger.Log.Sugar(), r, ctx)
		g.Go(func() error {
			logger.Log.Sugar().Info("Сервер запущен по протоколу HTTP/SSE на ", cfg.Net.String())
			return serv.Run()
		})
		g.Go(func() error {
			<-gCtx.Done()
			logger.Log.Sugar().Info("Остановка HTTP сервера...")
			return serv.Stop()
		})
	}
	if err := g.Wait(); err != nil {
		logger.Log.Sugar().Fatal("сервер аварийно завершил работу", "err", err)
	}
}
