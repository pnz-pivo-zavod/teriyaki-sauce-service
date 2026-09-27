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
	//go:embed queries/task/list.sql
	_taskListSQL string
	//go:embed queries/task/update.sql
	_taskUpdateSQL string
	//go:embed queries/task/complete.sql
	_taskCompleteSQL string
	//go:embed queries/task/delete.sql
	_taskDeleteSQL string
	//go:embed queries/task/add_tags.sql
	_taskAddTagsSQL string
	//go:embed queries/task/delete_tags.sql
	_taskDeleteTagsSQL string
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

// List возвращает задачи пользователя по фильтру без тегов и заметок,
// отсортированные по date (без даты — в конце), затем по id.
func (r *TaskRepository) List(ctx context.Context, userID int64, f model.TaskFilter) ([]model.Task, error) {
	rows, err := r.db.Query(ctx, _taskListSQL, userID, f.StartDate, f.EndDate, f.IsCompleted)
	if err != nil {
		return nil, fmt.Errorf("select tasks: %w", err)
	}

	tasks, err := pgx.CollectRows(rows, scanTask)
	if err != nil {
		return nil, fmt.Errorf("collect tasks: %w", err)
	}

	return tasks, nil
}

// Update полностью перезаписывает задачу и её теги в одной транзакции.
// Нет задачи → model.ErrNotFound; in.TagIDs — как в Create.
func (r *TaskRepository) Update(ctx context.Context, userID, id int64, in model.TaskInput) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		res, err := tx.Exec(ctx, _taskUpdateSQL,
			id, userID, in.Name, in.Description, in.Date, in.NotifyAt, in.Priority, in.IsCompleted,
		)
		if err != nil {
			return fmt.Errorf("update task: %w", err)
		}

		if res.RowsAffected() == 0 {
			return fmt.Errorf("task %d %w", id, model.ErrNotFound)
		}

		if _, err := tx.Exec(ctx, _taskDeleteTagsSQL, id); err != nil {
			return fmt.Errorf("delete task tags: %w", err)
		}

		return addTags(ctx, tx, userID, id, in.TagIDs)
	})
}

// Complete помечает задачу завершённой. Нет задачи → model.ErrNotFound.
func (r *TaskRepository) Complete(ctx context.Context, userID, id int64) error {
	return r.execOne(ctx, _taskCompleteSQL, "complete", userID, id)
}

// Delete удаляет задачу вместе с заметками и привязками тегов. Нет задачи → model.ErrNotFound.
func (r *TaskRepository) Delete(ctx context.Context, userID, id int64) error {
	return r.execOne(ctx, _taskDeleteSQL, "delete", userID, id)
}

// execOne выполняет запрос с параметрами (id, userID), 0 затронутых строк → model.ErrNotFound.
func (r *TaskRepository) execOne(ctx context.Context, sql, op string, userID, id int64) error {
	res, err := r.db.Exec(ctx, sql, id, userID)
	if err != nil {
		return fmt.Errorf("%s task: %w", op, err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("task %d %w", id, model.ErrNotFound)
	}

	return nil
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
