package handler

import (
	"net/http"

	"teriyaki-sauce-service/internal/api/rest/response"
	"teriyaki-sauce-service/internal/auth"
	"teriyaki-sauce-service/internal/model"
	"teriyaki-sauce-service/internal/service"
)

// TaskHandler — HTTP-хендлеры задач.
type TaskHandler struct {
	svc *service.TaskService
}

// NewTaskHandler создаёт TaskHandler.
func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// Create — POST /v1/task.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in model.TaskInput
	if err := decode(r, &in); err != nil {
		writeError(w, r, err)
		return
	}

	task, err := h.svc.Create(r.Context(), auth.UserID(r.Context()), in)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, task)
}

// Get — GET /v1/task/{id}.
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	task, err := h.svc.Get(r.Context(), auth.UserID(r.Context()), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, task)
}
