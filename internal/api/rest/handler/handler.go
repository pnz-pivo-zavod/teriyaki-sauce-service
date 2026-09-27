package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"teriyaki-sauce-service/internal/api/rest/response"
	"teriyaki-sauce-service/internal/auth"
	"teriyaki-sauce-service/internal/model"
)

// Health отвечает, что сервис жив.
func Health(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, "ok")
}

// Me возвращает ID текущего пользователя — для проверки авторизации с фронта.
func Me(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]int64{"userId": auth.UserID(r.Context())})
}

func decode(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return &model.ValidationError{Msg: "invalid json: " + err.Error()}
	}

	return nil
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, &model.ValidationError{Msg: "invalid id"}
	}

	return id, nil
}

// writeError переводит доменную ошибку в HTTP-статус. Неизвестные ошибки → 500 без деталей.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *model.ValidationError

	switch {
	case errors.As(err, &validationErr):
		response.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, model.ErrNotFound):
		response.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, model.ErrConflict):
		response.Error(w, http.StatusConflict, err.Error())
	default:
		log.Error().Err(err).Str("method", r.Method).Str("path", r.URL.Path).Msg("internal error")
		response.Error(w, http.StatusInternalServerError, "internal error")
	}
}
