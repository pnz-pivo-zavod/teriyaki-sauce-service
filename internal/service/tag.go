package service

import (
	"context"
	"regexp"
	"strings"

	"teriyaki-sauce-service/internal/model"
	"teriyaki-sauce-service/internal/repository"
)

var _colorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// TagService — логика тегов.
type TagService struct {
	repo *repository.TagRepository
}

// NewTagService создаёт TagService.
func NewTagService(repo *repository.TagRepository) *TagService {
	return &TagService{repo: repo}
}

// Create создаёт тег пользователя.
func (s *TagService) Create(ctx context.Context, userID int64, in model.TagInput) (model.Tag, error) {
	in, err := validateTag(in)
	if err != nil {
		return model.Tag{}, err
	}

	return s.repo.Create(ctx, userID, in)
}

// List возвращает все теги пользователя.
func (s *TagService) List(ctx context.Context, userID int64) ([]model.Tag, error) {
	return s.repo.List(ctx, userID)
}

// Update полностью заменяет имя и цвет тега.
func (s *TagService) Update(
	ctx context.Context, userID, id int64, in model.TagInput,
) (model.Tag, error) {
	in, err := validateTag(in)
	if err != nil {
		return model.Tag{}, err
	}

	return s.repo.Update(ctx, userID, id, in)
}

// Delete удаляет тег пользователя.
func (s *TagService) Delete(ctx context.Context, userID, id int64) error {
	return s.repo.Delete(ctx, userID, id)
}

func validateTag(in model.TagInput) (model.TagInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return in, &model.ValidationError{Msg: "name is required"}
	}

	if in.Color != "" && !_colorRe.MatchString(in.Color) {
		return in, &model.ValidationError{Msg: "color must be #RRGGBB"}
	}

	return in, nil
}
