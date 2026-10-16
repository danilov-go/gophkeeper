package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/client/repository/memory"
	"github.com/danilov-go/gophkeeper/internal/client/sender"
	"github.com/danilov-go/gophkeeper/internal/client/service"
	"github.com/danilov-go/gophkeeper/internal/client/tui"
	"github.com/danilov-go/gophkeeper/internal/client/worker"
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
	var send service.Sender
	deviceID := fmt.Sprintf("%d", os.Getpid())
	if cfg.Protocol == "grpc" {
		var err error
		send, err = sender.NewGRPCSender(cfg.Net.String(), logger.Log.Sugar(), deviceID)
		if err != nil {
			logger.Log.Sugar().Fatalw("не удалось запустить gRPC клиент", "error", err)
		}
	} else {
		send = sender.NewHTTPSender(*cfg, logger.Log.Sugar(), deviceID)
	}
	syncChan := make(chan struct{}, 1)
	srvc := service.NewClientService(storage, send, syncChan)
	sendInterval := 5 * time.Minute
	stepSend := 5 * time.Second
	syncWorker := worker.NewSyncWorker(srvc, logger.Log.Sugar(), sendInterval, stepSend, syncChan)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()
	go syncWorker.Run(ctx)
	go func() {
		send.ServerEvent(ctx, func() {
			select {
			case syncChan <- struct{}{}:
			default:
			}
		})
	}()
	appModel := tui.NewAppModel(srvc, syncChan, syncWorker.Occurrence())
	p := tea.NewProgram(appModel)
	if _, err := p.Run(); err != nil {
		logger.Log.Sugar().Fatalw("ошибка интерфейса", "error", err)
	}
}
