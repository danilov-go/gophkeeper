package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/danilov-go/gophkeeper/internal/models"
)

// PullHandler принимает версии от клиента и возвращает секреты, которые необходимо обновить.
//
// @Summary      Получить секреты, требующие обновления
// @Description  Принимает версии от клиента и возвращает секреты, которые необходимо обновить
// @Tags         secrets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        device-id  header    string                 true  "Идентификатор устройства"
// @Param        body       body      models.SecretVersions  true  "Версии секретов на клиенте"
// @Success      200        {array}   models.CipherData      "Список зашифрованных секретов"
// @Success      204        "Нет секретов для обновления"
// @Failure      400        {string}  string                 "Невалидный JSON"
// @Failure      401        {string}  string                 "Неавторизованный доступ"
// @Failure      415        {string}  string                 "Неверный Content-Type"
// @Failure      500        {string}  string                 "Внутренняя ошибка сервиса"
// @Router       /api/pull [post]
func (h *Handler) PullHandler() LoginHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, user models.AuthUser) {
		ctx := r.Context()
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			h.logger.Errorw("не соответствие content-type", "content-type", r.Header.Get("Content-Type"))
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}
		version, err := decode[models.SecretVersions](w, r)
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
		if _, err = w.Write(res); err != nil {
			h.logger.Errorw("ошибка записи", "error", err)
		}
	}
}

// PushHandler принимает журнал изменений от клиента и применяет его на сервере.
//
// @Summary      Отправить журнал изменений
// @Description  Принимает журнал изменений от клиента и применяет его на сервере
// @Tags         secrets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        device-id  header    string             true  "Идентификатор устройства"
// @Param        body       body      models.Changelogs  true  "Журнал изменений секретов"
// @Success      200        "Изменения успешно применены"
// @Failure      400        {string}  string                 "Невалидный JSON или отсутствует заголовок device-id"
// @Failure      401        {string}  string                 "Неавторизованный доступ"
// @Failure      415        {string}  string                 "Неверный Content-Type"
// @Failure      500        {string}  string                 "Внутренняя ошибка сервиса"
// @Router       /api/push [post]
func (h *Handler) PushHandler() LoginHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, user models.AuthUser) {
		ctx := r.Context()
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			h.logger.Errorw("не соответствие content-type", "content-type", r.Header.Get("Content-Type"))
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}
		changelogs, err := decode[models.Changelogs](w, r)
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

// StreamHandler отправляет клиенту события синхронизации.
//
// @Summary      Отправить событие синхронизации
// @Description  Отправляет клиенту события синхронизации.
// @Tags         sync
// @Produce      text/event-stream
// @Security     BearerAuth
// @Param        device-id  header    string  true  "Идентификатор устройства"
// @Success      200        {string}  string        "Поток событий в формате SSE (data: message\n\n)"
// @Failure      400        {string}  string        "Отсутствует заголовок device-id"
// @Failure      401        {string}  string        "Неавторизованный доступ"
// @Failure      500        {string}  string        "Внутренняя ошибка"
// @Router       /api/event [get]
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
