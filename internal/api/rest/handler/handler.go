package handler

import (
	"net/http"

	"teriyaki-sauce-service/internal/api/rest/response"
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
