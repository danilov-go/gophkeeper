// Package worker предоставляет фоновый воркер для периодической и принудительной синхронизации данных.
package worker

import (
	"context"
	"iter"
	"time"
)

// Syncer содержит метод для запуска процесса синхронизации данных.
type Syncer interface {
	SyncData(ctx context.Context) error
}

type log interface {
	Errorw(msg string, keysAndValues ...interface{})
}

// SyncWorker определяет воркер синхронизации.
type SyncWorker struct {
	syncer     Syncer
	logger     log
	interval   time.Duration
	cooldown   time.Duration
	syncCh     <-chan struct{}
	occurrence chan struct{}
}

// NewSyncWorker создает новый экземпляр SyncWorker.
func NewSyncWorker(syncer Syncer, l log, interval time.Duration, cooldown time.Duration, syncCh <-chan struct{}) *SyncWorker {
	return &SyncWorker{
		syncer:     syncer,
		logger:     l,
		interval:   interval,
		cooldown:   cooldown,
		syncCh:     syncCh,
		occurrence: make(chan struct{}, 1),
	}
}

// Run запускает воркер синхронизации.
func (w *SyncWorker) Run(ctx context.Context) {
	timer := time.NewTimer(w.interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			w.runSync(ctx)
			timer.Reset(w.interval)
		case <-w.syncCh:
			w.runSync(ctx)
			timer.Reset(w.cooldown)
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

// Occurrence возвращает канал для отслеживания фактов успешной синхронизации данных.
func (w *SyncWorker) Occurrence() iter.Seq[struct{}] {
	return func(yield func(struct{}) bool) {
		for range w.occurrence {
			if !yield(struct{}{}) {
				return
			}
		}
	}
}
