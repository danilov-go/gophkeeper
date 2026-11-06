package sync

import (
	"testing"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestSyncNotifier(t *testing.T) {
	notifier := NewSyncNotifier()
	userID := 15
	chan1, closeChan1 := notifier.Register(userID, "device1")
	chan2, closeChan2 := notifier.Register(userID, "device2")
	defer func() {
		defer func() { recover() }()
		closeChan2()
	}()
	event := models.SyncEvent{
		UserID:   userID,
		DeviceID: "device1",
		Message:  "sync",
	}
	notifier.Notify(event)
	select {
	case <-chan1:
		t.Fatal("device1 получил уведомление, которое сам отправил")
	case <-time.After(20 * time.Millisecond):
	}
	select {
	case receivedEvent := <-chan2:
		assert.Equal(t, userID, receivedEvent.UserID)
		assert.Equal(t, "device1", receivedEvent.DeviceID)
		assert.Equal(t, "sync", receivedEvent.Message)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("device1 не получил уведомление")
	}
	eventSync := models.SyncEvent{
		UserID:   userID,
		DeviceID: "device2",
		Message:  "sync",
	}
	done := make(chan bool)
	go func() {
		notifier.Notify(eventSync)
		notifier.Notify(eventSync)
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ошибка из-за переполнения канала")
	}
	for len(chan1) > 0 {
		<-chan1
	}
	closeChan1()
	time.Sleep(1 * time.Millisecond)
	_, ok := <-chan1
	assert.False(t, ok)
}
