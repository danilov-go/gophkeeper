package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

type mockSyncer struct {
	count int32
	err   error
}

func (s *mockSyncer) SyncData(ctx context.Context) error {
	atomic.AddInt32(&s.count, 1)
	return s.err
}

func (s *mockSyncer) GetCount() int {
	return int(atomic.LoadInt32(&s.count))
}

var errTest = errors.New("error test")

func TestSyncWorker(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		cooldown time.Duration
		trigger  bool
		expErr   error
		sleep    time.Duration
		count    int
	}{
		{
			name:     "запуск по интервалу",
			interval: 10 * time.Millisecond,
			cooldown: 1 * time.Hour,
			trigger:  false,
			expErr:   nil,
			sleep:    25 * time.Millisecond,
			count:    2,
		},
		{
			name:     "запуск через syncCh",
			interval: 1 * time.Hour,
			cooldown: 5 * time.Millisecond,
			trigger:  true,
			expErr:   nil,
			sleep:    10 * time.Millisecond,
			count:    1,
		},
		{
			name:     "синхронизация с ошибкой",
			interval: 1 * time.Hour,
			cooldown: 5 * time.Millisecond,
			trigger:  true,
			expErr:   errTest,
			sleep:    10 * time.Millisecond,
			count:    1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			syncer := &mockSyncer{err: tt.expErr}
			syncCh := make(chan struct{}, 1)
			logger := zaptest.NewLogger(t)
			worker := NewSyncWorker(syncer, logger.Sugar(), tt.interval, tt.cooldown, syncCh)
			go worker.Run(ctx)
			if tt.trigger {
				syncCh <- struct{}{}
				if tt.expErr == nil {
					select {
					case <-worker.Occurrence():
					case <-time.After(15 * time.Millisecond):
						assert.Fail(t, "канал Occurrence пуст")
					}
				}
			}
			time.Sleep(tt.sleep)
			assert.GreaterOrEqual(t, syncer.GetCount(), tt.count)
		})
	}
}
