package sync

import (
	"sync"
)

type SyncEvent struct {
	UserID  int
	Message string
}

type Publisher interface {
	Register(o Observer)
	Deregister(o Observer)
	Notify(event SyncEvent)
}

type Observer interface {
	Update(event SyncEvent)
	GetUserID() int
	GetDeviceID() string
}

type SyncNotifier struct {
	observers sync.Map
}

func NewSyncNotifier() *SyncNotifier {
	return &SyncNotifier{}
}

func (n *SyncNotifier) Register(o Observer) {
	n.observers.Store(o.GetDeviceID(), o)
}

func (n *SyncNotifier) Deregister(o Observer) {
	n.observers.Delete(o.GetDeviceID())
}

func (n *SyncNotifier) Notify(event SyncEvent) {
	n.observers.Range(func(key, val any) bool {
		if obs, ok := val.(Observer); ok {
			if obs.GetUserID() == event.UserID {
				obs.Update(event)
			}
		}
		return true
	})
}

type ClientStream struct {
	deviceID string
	userID   int
	ch       chan SyncEvent
}

func NewClientStream(deviceID string, userID int) *ClientStream {
	return &ClientStream{
		deviceID: deviceID,
		userID:   userID,
		ch:       make(chan SyncEvent, 10),
	}
}

func (c *ClientStream) GetDeviceID() string {
	return c.deviceID
}

func (c *ClientStream) GetUserID() int {
	return c.userID
}

func (c *ClientStream) Update(event SyncEvent) {
	select {
	case c.ch <- event:
	default:
	}
}

func (c *ClientStream) Stream() <-chan SyncEvent {
	return c.ch
}
