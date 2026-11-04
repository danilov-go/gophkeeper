package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/danilov-go/gophkeeper/docs"
	"github.com/danilov-go/gophkeeper/internal/config"
	"github.com/danilov-go/gophkeeper/internal/logger"
	"github.com/danilov-go/gophkeeper/internal/proto"
	"github.com/danilov-go/gophkeeper/internal/server/handler"
	"github.com/danilov-go/gophkeeper/internal/server/repository/db"
	"github.com/danilov-go/gophkeeper/internal/server/repository/memory"
	"github.com/danilov-go/gophkeeper/internal/server/service"
	"github.com/danilov-go/gophkeeper/internal/server/sync"
	"github.com/danilov-go/gophkeeper/internal/server/transport"
	"github.com/danilov-go/gophkeeper/internal/strategy"
	"github.com/go-chi/chi"
	httpSwagger "github.com/swaggo/http-swagger"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

// @title           GophKeeper API
// @version         1.0.0
// @description     API для управления пользователями и синхронизации событий.
// @securityDefinitions.apiKey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Введите токен в формате: Bearer <jwt_token>
// @host            localhost:8080
// @basePath
func main() {
	config.PrintBuild(os.Stdout, buildVersion, buildDate, buildCommit)
	if err := logger.Initialize("info"); err != nil {
		panic(err)
	}
	cfg := &config.ConfigServer{}
	if err := cfg.Get(); err != nil {
		logger.Log.Sugar().Fatal("ошибка парсинга конфигурации сервера")
	}
	if err := cfg.Validate(); err != nil {
		logger.Log.Sugar().Fatal(err)
	}
	conflict := strategy.NewServerStrategy()
	var storage service.Storage
	if cfg.DatabaseDSN != "" {
		db, err := db.InitDB(cfg.DatabaseDSN, conflict)
		if err != nil {
			logger.Log.Sugar().Fatalw("ошибка инициализации базы данных", "error", err)
		}
		storage = db
	} else {
		storage = memory.NewMemStorage(conflict)
	}
	notifier := sync.NewSyncNotifier()
	srv := service.NewServerService(storage, cfg.Token, notifier)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()
	g, gCtx := errgroup.WithContext(ctx)
	if cfg.Protocol == "grpc" {
		gServ := grpc.NewServer(grpc.UnaryInterceptor(handler.AuthInterceptor(srv)))
		h := handler.NewGRPCHandler(srv, logger.Log.Sugar())
		proto.RegisterSyncServiceServer(gServ, h)
		serv := transport.NewGRPC(cfg.Net.String(), logger.Log.Sugar(), gServ, cfg.PathTLSCert, cfg.PathTLSKey)
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
		r.Get("/swagger/*", httpSwagger.WrapHandler)
		r.Get("/api-docs", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
		}))
		r.Route("/user", func(r chi.Router) {
			r.Post("/register", h.RegisterUser())
			r.Post("/login", h.LoginUser())
			r.Get("/salt", h.GetSalt())
		})
		r.Route("/api", func(r chi.Router) {
			r.Get("/event", h.AuthMiddleware(h.StreamHandler()))
			r.With(handler.GzipMiddleware).Group(func(r chi.Router) {
				r.Post("/pull", h.AuthMiddleware(h.PullHandler()))
				r.Post("/push", h.AuthMiddleware(h.PushHandler()))
			})
		})
		serv := transport.New(cfg.Net.String(), logger.Log.Sugar(), r, ctx, cfg.PathTLSCert, cfg.PathTLSKey)
		g.Go(func() error {
			logger.Log.Sugar().Info("Сервер запущен по протоколу HTTPS на ", cfg.Net.String())
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
