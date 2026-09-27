package service

import (
	"context"
	"strings"

	"github.com/samber/lo"

	"teriyaki-sauce-service/internal/model"
	"teriyaki-sauce-service/internal/repository"
)

const _maxPriority = 3

// TaskService — логика задач.
type TaskService struct {
	tasks *repository.TaskRepository
	tags  *repository.TagRepository
}

// NewTaskService создаёт TaskService.
func NewTaskService(tasks *repository.TaskRepository, tags *repository.TagRepository) *TaskService {
	return &TaskService{tasks: tasks, tags: tags}
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

// fill подгружает теги задач одним запросом. tags и notes всегда не nil.
func (s *TaskService) fill(ctx context.Context, tasks []model.Task) ([]model.Task, error) {
	ids := lo.Map(tasks, func(t model.Task, _ int) int64 { return t.ID })

	tags, err := s.tags.ListByTaskIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	for i := range tasks {
		tasks[i].Tags = lo.CoalesceSliceOrEmpty(tags[tasks[i].ID])
		tasks[i].Notes = []model.Note{} // ponytail: заметки подгружаются в 4.2
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
