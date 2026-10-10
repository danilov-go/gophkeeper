package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/client/repository/memory"
	"github.com/danilov-go/gophkeeper/internal/client/sender"
	"github.com/danilov-go/gophkeeper/internal/client/service"
	"github.com/danilov-go/gophkeeper/internal/client/sync"
	"github.com/danilov-go/gophkeeper/internal/client/tui"
	"github.com/danilov-go/gophkeeper/internal/config"
	"github.com/danilov-go/gophkeeper/internal/logger"
)

func main() {
	cfg := &config.ConfigClient{}
	if err := logger.Initialize("info"); err != nil {
		panic(err)
	}
	if err := cfg.Get(); err != nil {
		logger.Log.Sugar().Fatal("ошибка загрузки конфигурации клиента")
	}
	storage := memory.InitMemStorage()
	send := sender.NewSender(*cfg, logger.Log.Sugar(), storage)
	sendInterval := 5 * time.Second
	stepSend := 2 * time.Second
	syncer := sync.NewSyncWorker(send, logger.Log.Sugar(), sendInterval, stepSend)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()
	go syncer.Worker(ctx)
	svc := service.NewClientService(storage, send, syncer)
	appModel := tui.NewAppModel(svc)
	p := tea.NewProgram(appModel)
	if _, err := p.Run(); err != nil {
		logger.Log.Sugar().Fatalw("ошибка интерфейса", "error", err)
	}
}
