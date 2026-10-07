package sync

import (
	"context"
	"time"
)

type Syncer interface {
	Sync(ctx context.Context) error
}

type Logger interface {
	Errorw(msg string, keysAndValues ...interface{})
}

type SyncWorker struct {
	syncer   Syncer
	logger   Logger
	interval time.Duration
	step     time.Duration
	syncCh   chan struct{}
}

func NewSyncWorker(syncer Syncer, l Logger, interval time.Duration, step time.Duration) *SyncWorker {
	return &SyncWorker{
		syncer:   syncer,
		logger:   l,
		interval: interval,
		step:     step,
		syncCh:   make(chan struct{}, 1),
	}
}

func (w *SyncWorker) RunSync() {
	select {
	case w.syncCh <- struct{}{}:
	default:
	}
}

func (w *SyncWorker) Worker(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.send(ctx)
		case <-w.syncCh:
			w.send(ctx)
			ticker.Reset(w.interval)
		}
	}
}

func (w *SyncWorker) send(ctx context.Context) {
	count := 0
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		err := w.syncer.Sync(ctx)
		if err == nil {
			return
		}
		count++
		duration := w.step * (1 << uint(count-1))
		if duration >= w.interval {
			w.logger.Errorw("лимит попыток синхронизации исчерпан", "err", err)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(duration):
		}
	}
}
