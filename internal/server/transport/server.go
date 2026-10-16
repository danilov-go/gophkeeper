// Package transport реализует HTTP-сервер приложения.
package transport

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

type log interface {
	Infow(msg string, keysAndValues ...any)
}

// Server определяет конфигурацию HTTP-сервера.
type Server struct {
	Server *http.Server
	Logger log
}

// New создает новый экземпляр Server.
func New(port string, l log, r http.Handler, ctx context.Context) *Server {
	server := &http.Server{
		Addr:    port,
		Handler: r,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
	return &Server{
		Server: server,
		Logger: l,
	}
}

// Run запускает HTTP-сервер.
func (serv *Server) Run() error {
	serv.Logger.Infow("HTTP server is starting", "address", serv.Server.Addr)
	if err := serv.Server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
	return nil
}

// Stop останавливает HTTP-сервер.
func (serv *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := serv.Server.Shutdown(ctx); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
	return nil
}
