package repository

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"teriyaki-sauce-service/internal/model"
)

var (
	//go:embed queries/note/create.sql
	_noteCreateSQL string
	//go:embed queries/note/update.sql
	_noteUpdateSQL string
	//go:embed queries/note/delete.sql
	_noteDeleteSQL string
)

// NoteRepository — заметки в Postgres. Владелец заметки — владелец её задачи.
type NoteRepository struct {
	db *pgxpool.Pool
}

// NewNoteRepository создаёт NoteRepository.
func NewNoteRepository(db *pgxpool.Pool) *NoteRepository {
	return &NoteRepository{db: db}
}

// Create добавляет заметку к задаче пользователя. Нет задачи → model.ErrNotFound.
func (r *NoteRepository) Create(ctx context.Context, userID int64, in model.NoteInput) (model.Note, error) {
	rows, err := r.db.Query(ctx, _noteCreateSQL, in.TaskID, userID, in.Text)
	if err != nil {
		return model.Note{}, fmt.Errorf("insert note: %w", err)
	}

	note, err := pgx.CollectOneRow(rows, pgx.RowToStructByPos[model.Note])
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Note{}, fmt.Errorf("task %d %w", in.TaskID, model.ErrNotFound)
	}

	if err != nil {
		return model.Note{}, fmt.Errorf("collect note: %w", err)
	}

	return note, nil
}

// Update меняет текст заметки и переносит её в задачу in.TaskID.
// Нет заметки или целевой задачи → model.ErrNotFound.
func (r *NoteRepository) Update(
	ctx context.Context, userID, id int64, in model.NoteInput,
) (model.Note, error) {
	rows, err := r.db.Query(ctx, _noteUpdateSQL, id, userID, in.TaskID, in.Text)
	if err != nil {
		return model.Note{}, fmt.Errorf("update note: %w", err)
	}

	note, err := pgx.CollectOneRow(rows, pgx.RowToStructByPos[model.Note])
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Note{}, fmt.Errorf("note %d or task %d %w", id, in.TaskID, model.ErrNotFound)
	}

	if err != nil {
		return model.Note{}, fmt.Errorf("collect note: %w", err)
	}

	return note, nil
}

// Delete удаляет заметку пользователя. Нет заметки → model.ErrNotFound.
func (r *NoteRepository) Delete(ctx context.Context, userID, id int64) error {
	res, err := r.db.Exec(ctx, _noteDeleteSQL, id, userID)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("note %d %w", id, model.ErrNotFound)
	}

	return nil
}
