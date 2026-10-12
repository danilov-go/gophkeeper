package handler

import (
	"fmt"
	"net/http"

	"github.com/danilov-go/gophkeeper/internal/server/sync"
)

func (h *Handler) EventHandler() LoginHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, user AuthUser) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		deviceID := r.Header.Get("device-id")
		if deviceID == "" {
			deviceID = "unknown"
		}
		stream := sync.NewClientStream(deviceID, user.ID)
		h.event.Register(stream)
		defer h.event.Deregister(stream)
		for {
			select {
			case <-r.Context().Done():
				return
			case event := <-stream.Stream():
				_, err := fmt.Fprintf(w, "data: %s\n\n", event.Message)
				if err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}
