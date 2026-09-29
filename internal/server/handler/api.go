package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
)

type secret struct {
	ID        int               `json:"id,omitempty"`
	SType     models.SecretType `json:"type"`
	SBody     []byte            `json:"body"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type secretID struct {
	ID int `json:"id"`
}

func (h *Handler) SaveSecret() LoginHandlerFunc {
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
		var secretBody secret
		err = json.Unmarshal(body, &secretBody)
		if err != nil {
			h.logger.Errorw("ошибка десилиризации", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		cipherData := models.CipherData{
			ID:        secretBody.ID,
			UserID:    user.ID,
			Type:      secretBody.SType,
			Cipher:    secretBody.SBody,
			UpdatedAt: secretBody.UpdatedAt,
		}
		id, err := h.storage.Save(ctx, user.Login, cipherData)
		if err != nil {
			h.logger.Errorw("ошибка сохранения/обновления", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		res, err := json.Marshal(secretID{ID: id})
		if err != nil {
			h.logger.Errorw("ошибка сериализации", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	}
}

func (h *Handler) GetSecret() LoginHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, user AuthUser) {
		ctx := r.Context()
		strID := r.URL.Query().Get("id")
		if strID == "" {
			h.logger.Errorw("в запросе отсутствует id")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(strID)
		if err != nil {
			h.logger.Errorw("ошибка парсинга id", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		secret, err := h.storage.Get(ctx, user.ID, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				h.logger.Errorw("секрет не найден или доступ запрещен", "id", id, "user_id", user.ID)
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			h.logger.Errorw("ошибка получения из хранилища", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		res, err := json.Marshal(secret)
		if err != nil {
			h.logger.Errorw("ошибка сериализации", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	}
}

func (h *Handler) GetAllSecret() LoginHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, user AuthUser) {
		ctx := r.Context()
		secrets, err := h.storage.GetAll(ctx, user.ID)
		if err != nil {
			h.logger.Errorw("ошибка получения из хранилища", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if len(secrets) == 0 {
			h.logger.Errorw("секреты не найдены", "user_id", user.ID)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		res, err := json.Marshal(secrets)
		if err != nil {
			h.logger.Errorw("ошибка сериализации", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	}
}

func (h *Handler) DeleteSecret() LoginHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, user AuthUser) {
		ctx := r.Context()
		strID := r.URL.Query().Get("id")
		if strID == "" {
			h.logger.Errorw("в запросе отсутствует id")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(strID)
		if err != nil {
			h.logger.Errorw("ошибка парсинга id", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		err = h.storage.Delete(ctx, user.ID, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				h.logger.Errorw("секрет не найден или доступ запрещен", "id", id, "user_id", user.ID)
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			h.logger.Errorw("ошибка удаления секрета", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
