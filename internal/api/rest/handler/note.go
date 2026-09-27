package handler

import (
	"net/http"

	"teriyaki-sauce-service/internal/api/rest/response"
	"teriyaki-sauce-service/internal/auth"
	"teriyaki-sauce-service/internal/model"
	"teriyaki-sauce-service/internal/service"
)

// NoteHandler — HTTP-хендлеры заметок.
type NoteHandler struct {
	svc *service.NoteService
}

// NewNoteHandler создаёт NoteHandler.
func NewNoteHandler(svc *service.NoteService) *NoteHandler {
	return &NoteHandler{svc: svc}
}

// Create — POST /v1/note.
func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in model.NoteInput
	if err := decode(r, &in); err != nil {
		writeError(w, r, err)
		return
	}

	note, err := h.svc.Create(r.Context(), auth.UserID(r.Context()), in)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, note)
}

// Update — PUT /v1/note/{id}.
func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	var in model.NoteInput
	if err := decode(r, &in); err != nil {
		writeError(w, r, err)
		return
	}

	note, err := h.svc.Update(r.Context(), auth.UserID(r.Context()), id, in)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, note)
}

// Delete — DELETE /v1/note/{id}.
func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
