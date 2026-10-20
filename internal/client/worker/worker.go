package worker

import (
	"context"
	"time"
)

type Syncer interface {
	SyncData(ctx context.Context) error
}

type Logger interface {
	Errorw(msg string, keysAndValues ...interface{})
}

type SyncWorker struct {
	syncer     Syncer
	logger     Logger
	interval   time.Duration
	cooldown   time.Duration
	syncCh     <-chan struct{}
	occurrence chan struct{}
}

func NewSyncWorker(syncer Syncer, l Logger, interval time.Duration, cooldown time.Duration, syncCh <-chan struct{}) *SyncWorker {
	return &SyncWorker{
		syncer:     syncer,
		logger:     l,
		interval:   interval,
		cooldown:   cooldown,
		syncCh:     syncCh,
		occurrence: make(chan struct{}, 1),
	}
}

func (w *SyncWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runSync(ctx)
		case <-w.syncCh:
			w.runSync(ctx)
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.cooldown):
			}
		}
	}
}

func (w *SyncWorker) runSync(ctx context.Context) {
	netCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := w.syncer.SyncData(netCtx); err != nil {
		w.logger.Errorw("ошибка синхронизации данных", "error", err)
	} else {
		select {
		case w.occurrence <- struct{}{}:
		default:
		}
	}
}

func (w *SyncWorker) Occurrence() <-chan struct{} {
	return w.occurrence
}
