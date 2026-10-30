package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/client/repository/db"
	"github.com/danilov-go/gophkeeper/internal/client/sender"
	"github.com/danilov-go/gophkeeper/internal/client/service"
	"github.com/danilov-go/gophkeeper/internal/client/tui"
	"github.com/danilov-go/gophkeeper/internal/client/worker"
	"github.com/danilov-go/gophkeeper/internal/config"
	"github.com/danilov-go/gophkeeper/internal/logger"
	"github.com/danilov-go/gophkeeper/internal/strategy"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	if err := logger.Initialize("info"); err != nil {
		panic(err)
	}
	cfg := &config.ConfigClient{}
	if err := cfg.Get(); err != nil {
		logger.Log.Sugar().Fatal("ошибка парсинга конфигурации клиента")
	}
	if err := cfg.Validate(); err != nil {
		logger.Log.Sugar().Fatal(err)
	}
	db, err := db.InitDB(cfg.PathDB)
	if err != nil {
		logger.Log.Sugar().Fatal("ошибка инициализации базы данных", "error", err)
	}
	var send service.Sender
	deviceID := fmt.Sprintf("%d", os.Getpid())
	if cfg.Protocol == "grpc" {
		var err error
		send, err = sender.NewGRPCSender(cfg.Net.String(), logger.Log.Sugar(), deviceID, cfg.PathTLS)
		if err != nil {
			logger.Log.Sugar().Fatalw("не удалось запустить gRPC клиент", "error", err)
		}
	} else {
		send = sender.NewHTTPSender(*cfg, logger.Log.Sugar(), deviceID, cfg.PathTLS)
	}
	syncChan := make(chan struct{}, 1)
	conflict := strategy.NewClientStrategy()
	srvc := service.NewClientService(db, send, conflict, syncChan)
	syncWorker := worker.NewSyncWorker(srvc, logger.Log.Sugar(), cfg.SendInterval, cfg.Cooldown, syncChan)
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
	bulder := tui.Bulder{
		Version: buildVersion,
		Date:    buildDate,
		Commit:  buildCommit,
	}
	appModel := tui.NewAppModel(srvc, syncChan, syncWorker.Occurrence(), bulder)
	p := tea.NewProgram(appModel)
	if _, err := p.Run(); err != nil {
		logger.Log.Sugar().Fatalw("ошибка интерфейса", "error", err)
	}
}
