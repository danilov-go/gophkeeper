package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/danilov-go/gophkeeper/internal/models"
)

func (h *Handler) PullHandler() LoginHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, user AuthUser) {
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
		cipherData, err := h.storage.GetSecrets(ctx, user.ID, version)
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
	return func(w http.ResponseWriter, r *http.Request, user AuthUser) {
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
		err = h.storage.UpdateSecrets(ctx, user.ID, changelogs)
		if err != nil {
			h.logger.Errorw("ошибка применения журнала изменений в хранилище", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
