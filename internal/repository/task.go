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
	//go:embed queries/task/create.sql
	_taskCreateSQL string
	//go:embed queries/task/get.sql
	_taskGetSQL string
	//go:embed queries/task/add_tags.sql
	_taskAddTagsSQL string
)

// TaskRepository — задачи в Postgres.
type TaskRepository struct {
	db *pgxpool.Pool
}

// NewTaskRepository создаёт TaskRepository.
func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create создаёт задачу и привязывает теги в одной транзакции. Возвращает ID задачи.
// in.TagIDs должны быть без дублей; чужой/несуществующий тег → *model.ValidationError.
func (r *TaskRepository) Create(ctx context.Context, userID int64, in model.TaskInput) (int64, error) {
	var id int64

	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, _taskCreateSQL,
			userID, in.Name, in.Description, in.Date, in.NotifyAt, in.Priority,
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("insert task: %w", err)
		}

		return addTags(ctx, tx, userID, id, in.TagIDs)
	})
	if err != nil {
		return 0, err
	}

	return id, nil
}

// Get возвращает задачу пользователя без тегов и заметок. Нет задачи → model.ErrNotFound.
func (r *TaskRepository) Get(ctx context.Context, userID, id int64) (model.Task, error) {
	rows, err := r.db.Query(ctx, _taskGetSQL, id, userID)
	if err != nil {
		return model.Task{}, fmt.Errorf("select task: %w", err)
	}

	task, err := pgx.CollectOneRow(rows, scanTask)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Task{}, fmt.Errorf("task %d %w", id, model.ErrNotFound)
	}

	if err != nil {
		return model.Task{}, fmt.Errorf("collect task: %w", err)
	}

	return task, nil
}

func addTags(ctx context.Context, tx pgx.Tx, userID, taskID int64, tagIDs []int64) error {
	if len(tagIDs) == 0 {
		return nil
	}

	res, err := tx.Exec(ctx, _taskAddTagsSQL, taskID, userID, tagIDs)
	if err != nil {
		return fmt.Errorf("insert task tags: %w", err)
	}

	if res.RowsAffected() != int64(len(tagIDs)) {
		return &model.ValidationError{Msg: "tagIds contain unknown tags"}
	}

	return nil
}

func scanTask(row pgx.CollectableRow) (model.Task, error) {
	var t model.Task
	err := row.Scan(&t.ID, &t.Name, &t.Description, &t.Date, &t.NotifyAt, &t.Priority, &t.IsCompleted)

	return t, err
}
