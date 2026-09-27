package repository

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"

	"teriyaki-sauce-service/internal/model"
)

var (
	//go:embed queries/note/create.sql
	_noteCreateSQL string
	//go:embed queries/note/update.sql
	_noteUpdateSQL string
	//go:embed queries/note/delete.sql
	_noteDeleteSQL string
	//go:embed queries/note/list_by_task_ids.sql
	_noteListByTaskIDsSQL string
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

// ListByTaskIDs возвращает заметки задач (по возрастанию date), сгруппированные по ID задачи.
// Задачи без заметок в результате отсутствуют.
func (r *NoteRepository) ListByTaskIDs(ctx context.Context, taskIDs []int64) (map[int64][]model.Note, error) {
	rows, err := r.db.Query(ctx, _noteListByTaskIDsSQL, taskIDs)
	if err != nil {
		return nil, fmt.Errorf("select task notes: %w", err)
	}

	notes, err := pgx.CollectRows(rows, pgx.RowToStructByPos[model.Note])
	if err != nil {
		return nil, fmt.Errorf("collect task notes: %w", err)
	}

	return lo.GroupBy(notes, func(n model.Note) int64 { return n.TaskID }), nil
}
