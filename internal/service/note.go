package service

import (
	"context"
	"strings"

	"teriyaki-sauce-service/internal/model"
	"teriyaki-sauce-service/internal/repository"
)

// NoteService — логика заметок.
type NoteService struct {
	repo *repository.NoteRepository
}

// NewNoteService создаёт NoteService.
func NewNoteService(repo *repository.NoteRepository) *NoteService {
	return &NoteService{repo: repo}
}

// Create добавляет заметку к задаче пользователя.
func (s *NoteService) Create(ctx context.Context, userID int64, in model.NoteInput) (model.Note, error) {
	in, err := validateNote(in)
	if err != nil {
		return model.Note{}, err
	}

	return s.repo.Create(ctx, userID, in)
}

// Update меняет текст заметки; другой taskId переносит её в эту задачу.
func (s *NoteService) Update(
	ctx context.Context, userID, id int64, in model.NoteInput,
) (model.Note, error) {
	in, err := validateNote(in)
	if err != nil {
		return model.Note{}, err
	}

	return s.repo.Update(ctx, userID, id, in)
}

// Delete удаляет заметку пользователя.
func (s *NoteService) Delete(ctx context.Context, userID, id int64) error {
	return s.repo.Delete(ctx, userID, id)
}

func validateNote(in model.NoteInput) (model.NoteInput, error) {
	if in.TaskID <= 0 {
		return in, &model.ValidationError{Msg: "taskId is required"}
	}

	in.Text = strings.TrimSpace(in.Text)
	if in.Text == "" {
		return in, &model.ValidationError{Msg: "text is required"}
	}

	return in, nil
}
