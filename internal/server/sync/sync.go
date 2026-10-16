package sync

import (
	"sync"

	"github.com/danilov-go/gophkeeper/internal/models"
)

type Observer interface {
	GetUserID() int
	GetDeviceID() string
	Update(event models.SyncEvent)
	Stream() <-chan models.SyncEvent
}

type Publisher interface {
	Register(o Observer)
	Deregister(o Observer)
	Notify(event models.SyncEvent)
}

type SyncNotifier struct {
	observers sync.Map
}

func NewSyncNotifier() *SyncNotifier {
	return &SyncNotifier{}
}

func (n *SyncNotifier) Register(userID int, deviceID string) (<-chan models.SyncEvent, func()) {
	stream := NewHTTPStream(deviceID, userID)
	n.observers.Store(deviceID, stream)
	unsubscribe := func() {
		n.observers.Delete(deviceID)
		stream.Close()
	}
	return stream.Stream(), unsubscribe
}

func (n *SyncNotifier) Deregister(o Observer) {
	n.observers.Delete(o.GetDeviceID())
}

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
func (c *HTTPStream) Close() {
	close(c.ch)
}

type HTTPStream struct {
	deviceID string
	userID   int
	ch       chan models.SyncEvent
}

func NewHTTPStream(deviceID string, userID int) *HTTPStream {
	return &HTTPStream{
		deviceID: deviceID,
		userID:   userID,
		ch:       make(chan models.SyncEvent, 1),
	}
}

func (c *HTTPStream) GetDeviceID() string {
	return c.deviceID
}

func (c *HTTPStream) GetUserID() int {
	return c.userID
}

func (c *HTTPStream) Update(event models.SyncEvent) {
	select {
	case c.ch <- event:
	default:
	}
}

func (c *HTTPStream) Stream() <-chan models.SyncEvent {
	return c.ch
}
