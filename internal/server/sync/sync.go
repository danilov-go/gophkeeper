// Package sync содержит механизмы рассылки уведомлений
// о событиях синхронизации.
package sync

import (
	"sync"

	"github.com/danilov-go/gophkeeper/internal/models"
)

type streamID struct {
	userID   int
	deviceID string
}

// Observer принимает уведомления о синхронизации для конкретного устройства.
type Observer interface {
	GetUserID() int
	GetDeviceID() string
	Update(event models.SyncEvent)
	Stream() <-chan models.SyncEvent
}

// Publisher управляет подписчиками и рассылкой событий.
type Publisher interface {
	Register(o Observer)
	Deregister(o Observer)
	Notify(event models.SyncEvent)
}

// SyncNotifier координирует рассылку уведомлений.
type SyncNotifier struct {
	observers sync.Map
}

// NewSyncNotifier создает новый экземпляр SyncNotifier.
func NewSyncNotifier() *SyncNotifier {
	return &SyncNotifier{}
}

// Register регистрирует устройство для получения уведомлений синхронизации.
func (n *SyncNotifier) Register(userID int, deviceID string) (<-chan models.SyncEvent, func()) {
	id := streamID{userID: userID, deviceID: deviceID}
	stream := NewStream(deviceID, userID)
	n.observers.Store(id, stream)
	unsubscribe := func() {
		n.observers.Delete(id)
		stream.Close()
	}
	return stream.Stream(), unsubscribe
}

// Deregister удаляет устройство.
func (n *SyncNotifier) Deregister(o Observer) {
	id := streamID{userID: o.GetUserID(), deviceID: o.GetDeviceID()}
	n.observers.Delete(id)
}

// Notify рассылает событие всем зарегистрированным устройствам.
func (n *SyncNotifier) Notify(event models.SyncEvent) {
	n.observers.Range(func(key, val any) bool {
		if obs, ok := val.(Observer); ok {
			if obs.GetUserID() == event.UserID && obs.GetDeviceID() != event.DeviceID {
				obs.Update(event)
			}
		}
		return true
	})
}

// Close закрывает канал событий синхронизации.
func (c *Stream) Close() {
	close(c.ch)
}

// Stream реализует Observer для конкретного устройства.
type Stream struct {
	deviceID string
	userID   int
	ch       chan models.SyncEvent
}

// NewStream создает новый экземпляр Stream.
func NewStream(deviceID string, userID int) *Stream {
	return &Stream{
		deviceID: deviceID,
		userID:   userID,
		ch:       make(chan models.SyncEvent, 1),
	}
}

// GetDeviceID возвращает ID устройства подписчика.
func (c *Stream) GetDeviceID() string {
	return c.deviceID
}

// GetUserID возвращает ID подписчика.
func (c *Stream) GetUserID() int {
	return c.userID
}

// Update отправляет событие о синхронизации в поток событий.
func (c *Stream) Update(event models.SyncEvent) {
	select {
	case c.ch <- event:
	default:
	}
}

// Stream возвращает канал для получения входящих событий синхронизации.
func (c *Stream) Stream() <-chan models.SyncEvent {
	return c.ch
}
