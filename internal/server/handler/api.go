package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/danilov-go/gophkeeper/internal/models"
)

func (h *Handler) PullHandler() LoginHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, user models.AuthUser) {
		ctx := r.Context()
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			h.logger.Errorw("не соответствие content-type", "content-type", r.Header.Get("Content-Type"))
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			h.logger.Errorw("ошибка чтения body", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		var version models.SecretVersions
		err = json.Unmarshal(body, &version)
		if err != nil {
			h.logger.Errorw("ошибка десериализации", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		cipherData, err := h.service.Pull(ctx, user.ID, version)
		if err != nil {
			h.logger.Errorw("ошибка получения секретов из хранилища", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if len(cipherData) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		res, err := json.Marshal(cipherData)
		if err != nil {
			h.logger.Errorw("ошибка сериализации ответа", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	}
}

func (h *Handler) PushHandler() LoginHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, user models.AuthUser) {
		ctx := r.Context()
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			h.logger.Errorw("не соответствие content-type", "content-type", r.Header.Get("Content-Type"))
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			h.logger.Errorw("ошибка чтения body", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		var changelogs models.Changelogs
		err = json.Unmarshal(body, &changelogs)
		if err != nil {
			h.logger.Errorw("ошибка десериализации", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		deviceID := r.Header.Get("device-id")
		if deviceID == "" {
			h.logger.Errorw("отсутствует device-id")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		err = h.service.Push(ctx, user.ID, deviceID, changelogs)
		if err != nil {
			h.logger.Errorw("ошибка применения журнала изменений в сервисе", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) StreamHandler() LoginHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, user models.AuthUser) {
		ctx := r.Context()
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
			h.logger.Errorw("отсутствует device-id")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		stream, close := h.service.Event(ctx, user.ID, deviceID)
		defer close()
		for {
			select {
			case <-r.Context().Done():
				return
			case event, ok := <-stream:
				if !ok {
					return
				}
				_, err := fmt.Fprintf(w, "data: %s\n\n", event.Message)
				if err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}
