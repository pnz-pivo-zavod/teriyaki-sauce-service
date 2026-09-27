package handler

import (
	"net/http"
	"strconv"
	"time"

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

// List — GET /v1/tasks?startDate=&endDate=&isCompleted=. Даты в RFC3339.
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	f, err := parseTaskFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	tasks, err := h.svc.List(r.Context(), auth.UserID(r.Context()), f)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, tasks)
}

// Update — PUT /v1/task/{id}.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	var in model.TaskInput
	if err := decode(r, &in); err != nil {
		writeError(w, r, err)
		return
	}

	task, err := h.svc.Update(r.Context(), auth.UserID(r.Context()), id, in)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, task)
}

// Complete — PATCH /v1/task/{id}/complete.
func (h *TaskHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	task, err := h.svc.Complete(r.Context(), auth.UserID(r.Context()), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, task)
}

// Delete — DELETE /v1/task/{id}.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if err := h.svc.Delete(r.Context(), auth.UserID(r.Context()), id); err != nil {
		writeError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, nil)
}

func parseTaskFilter(r *http.Request) (model.TaskFilter, error) {
	var (
		f   model.TaskFilter
		q   = r.URL.Query()
		err error
	)

	if f.StartDate, err = parseTimeParam(q.Get("startDate"), "startDate"); err != nil {
		return f, err
	}

	if f.EndDate, err = parseTimeParam(q.Get("endDate"), "endDate"); err != nil {
		return f, err
	}

	if v := q.Get("isCompleted"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return f, &model.ValidationError{Msg: "isCompleted must be true or false"}
		}

		f.IsCompleted = &b
	}

	return f, nil
}

func parseTimeParam(v, name string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}

	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, &model.ValidationError{Msg: name + " must be RFC3339 (url-encode '+' as %2B)"}
	}

	return &t, nil
}
