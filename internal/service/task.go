package service

import (
	"context"
	"strings"
	"time"

	"github.com/samber/lo"

	"teriyaki-sauce-service/internal/model"
	"teriyaki-sauce-service/internal/repository"
)

const (
	_maxPriority = 3
	_day         = 24 * time.Hour
)

// TaskService — логика задач.
type TaskService struct {
	tasks *repository.TaskRepository
	tags  *repository.TagRepository
	notes *repository.NoteRepository
}

// NewTaskService создаёт TaskService.
func NewTaskService(
	tasks *repository.TaskRepository, tags *repository.TagRepository, notes *repository.NoteRepository,
) *TaskService {
	return &TaskService{tasks: tasks, tags: tags, notes: notes}
}

// Create создаёт задачу и возвращает её целиком. isCompleted из запроса игнорируется.
func (s *TaskService) Create(ctx context.Context, userID int64, in model.TaskInput) (model.Task, error) {
	in, err := validateTask(in)
	if err != nil {
		return model.Task{}, err
	}

	id, err := s.tasks.Create(ctx, userID, in)
	if err != nil {
		return model.Task{}, err
	}

	return s.Get(ctx, userID, id)
}

// Get возвращает задачу пользователя с тегами и заметками.
func (s *TaskService) Get(ctx context.Context, userID, id int64) (model.Task, error) {
	task, err := s.tasks.Get(ctx, userID, id)
	if err != nil {
		return model.Task{}, err
	}

	tasks, err := s.fill(ctx, []model.Task{task})
	if err != nil {
		return model.Task{}, err
	}

	return tasks[0], nil
}

// List возвращает задачи пользователя по фильтру с тегами и заметками.
// Только StartDate → интервал в одни сутки от него.
func (s *TaskService) List(ctx context.Context, userID int64, f model.TaskFilter) ([]model.Task, error) {
	if f.StartDate != nil && f.EndDate == nil {
		end := f.StartDate.Add(_day)
		f.EndDate = &end
	}

	if f.StartDate != nil && !f.EndDate.After(*f.StartDate) {
		return nil, &model.ValidationError{Msg: "endDate must be after startDate"}
	}

	tasks, err := s.tasks.List(ctx, userID, f)
	if err != nil {
		return nil, err
	}

	return s.fill(ctx, tasks)
}

// Update полностью заменяет задачу (включая теги и isCompleted) и возвращает её целиком.
func (s *TaskService) Update(
	ctx context.Context, userID, id int64, in model.TaskInput,
) (model.Task, error) {
	in, err := validateTask(in)
	if err != nil {
		return model.Task{}, err
	}

	if err := s.tasks.Update(ctx, userID, id, in); err != nil {
		return model.Task{}, err
	}

	return s.Get(ctx, userID, id)
}

// Complete помечает задачу завершённой (идемпотентно) и возвращает её целиком.
func (s *TaskService) Complete(ctx context.Context, userID, id int64) (model.Task, error) {
	if err := s.tasks.Complete(ctx, userID, id); err != nil {
		return model.Task{}, err
	}

	return s.Get(ctx, userID, id)
}

// Delete удаляет задачу пользователя.
func (s *TaskService) Delete(ctx context.Context, userID, id int64) error {
	return s.tasks.Delete(ctx, userID, id)
}

// fill подгружает теги и заметки задач — по одному запросу на всё. tags и notes всегда не nil.
func (s *TaskService) fill(ctx context.Context, tasks []model.Task) ([]model.Task, error) {
	if len(tasks) == 0 {
		return tasks, nil
	}

	ids := lo.Map(tasks, func(t model.Task, _ int) int64 { return t.ID })

	tags, err := s.tags.ListByTaskIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	notes, err := s.notes.ListByTaskIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	for i := range tasks {
		tasks[i].Tags = lo.CoalesceSliceOrEmpty(tags[tasks[i].ID])
		tasks[i].Notes = lo.CoalesceSliceOrEmpty(notes[tasks[i].ID])
	}

	return tasks, nil
}

func validateTask(in model.TaskInput) (model.TaskInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return in, &model.ValidationError{Msg: "name is required"}
	}

	if in.Priority < 0 || in.Priority > _maxPriority {
		return in, &model.ValidationError{Msg: "priority must be 0..3"}
	}

	in.TagIDs = lo.Uniq(in.TagIDs)

	return in, nil
}
