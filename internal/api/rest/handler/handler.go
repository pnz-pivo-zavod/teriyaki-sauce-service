package handler

import (
	"net/http"

	"teriyaki-sauce-service/internal/api/rest/response"
	"teriyaki-sauce-service/internal/auth"
)

// Handler — HTTP-хендлеры API.
type Handler struct{}

// New создаёт Handler.
func New() *Handler {
	return &Handler{}
}

// Health отвечает, что сервис жив.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, "ok")
}

// Me возвращает ID текущего пользователя — для проверки авторизации с фронта.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]int64{"userId": auth.UserID(r.Context())})
}
