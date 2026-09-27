package handler

import (
	"net/http"

	"teriyaki-sauce-service/internal/api/rest/response"
	"teriyaki-sauce-service/internal/auth"
	"teriyaki-sauce-service/internal/model"
	"teriyaki-sauce-service/internal/service"
)

// TagHandler — HTTP-хендлеры тегов.
type TagHandler struct {
	svc *service.TagService
}

// NewTagHandler создаёт TagHandler.
func NewTagHandler(svc *service.TagService) *TagHandler {
	return &TagHandler{svc: svc}
}

// Create — POST /v1/tag.
func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in model.TagInput
	if err := decode(r, &in); err != nil {
		writeError(w, r, err)
		return
	}

	tag, err := h.svc.Create(r.Context(), auth.UserID(r.Context()), in)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, tag)
}

// List — GET /v1/tags.
func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	tags, err := h.svc.List(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		writeError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, tags)
}

// Update — PUT /v1/tag/{id}.
func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	var in model.TagInput
	if err := decode(r, &in); err != nil {
		writeError(w, r, err)
		return
	}

	tag, err := h.svc.Update(r.Context(), auth.UserID(r.Context()), id, in)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, tag)
}

// Delete — DELETE /v1/tag/{id}.
func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
