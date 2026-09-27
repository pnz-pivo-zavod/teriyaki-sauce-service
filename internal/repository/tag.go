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
	//go:embed queries/tag/create.sql
	_tagCreateSQL string
	//go:embed queries/tag/list.sql
	_tagListSQL string
	//go:embed queries/tag/update.sql
	_tagUpdateSQL string
	//go:embed queries/tag/delete.sql
	_tagDeleteSQL string
)

// TagRepository — теги в Postgres.
type TagRepository struct {
	db *pgxpool.Pool
}

// NewTagRepository создаёт TagRepository.
func NewTagRepository(db *pgxpool.Pool) *TagRepository {
	return &TagRepository{db: db}
}

// Create создаёт тег пользователя. Имя занято → model.ErrConflict.
func (r *TagRepository) Create(ctx context.Context, userID int64, in model.TagInput) (model.Tag, error) {
	tag := model.Tag{Name: in.Name, Color: in.Color}

	err := r.db.QueryRow(ctx, _tagCreateSQL, userID, in.Name, in.Color).Scan(&tag.ID)
	if isUniqueViolation(err) {
		return model.Tag{}, fmt.Errorf("tag %q %w", in.Name, model.ErrConflict)
	}

	if err != nil {
		return model.Tag{}, fmt.Errorf("insert tag: %w", err)
	}

	return tag, nil
}

// List возвращает все теги пользователя, отсортированные по имени.
func (r *TagRepository) List(ctx context.Context, userID int64) ([]model.Tag, error) {
	rows, err := r.db.Query(ctx, _tagListSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("select tags: %w", err)
	}

	tags, err := pgx.CollectRows(rows, pgx.RowToStructByPos[model.Tag])
	if err != nil {
		return nil, fmt.Errorf("collect tags: %w", err)
	}

	return tags, nil
}

// Update перезаписывает имя и цвет тега пользователя.
// Нет тега → model.ErrNotFound, имя занято → model.ErrConflict.
func (r *TagRepository) Update(
	ctx context.Context, userID, id int64, in model.TagInput,
) (model.Tag, error) {
	tag := model.Tag{Name: in.Name, Color: in.Color}

	err := r.db.QueryRow(ctx, _tagUpdateSQL, id, userID, in.Name, in.Color).Scan(&tag.ID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return model.Tag{}, fmt.Errorf("tag %d %w", id, model.ErrNotFound)
	case isUniqueViolation(err):
		return model.Tag{}, fmt.Errorf("tag %q %w", in.Name, model.ErrConflict)
	case err != nil:
		return model.Tag{}, fmt.Errorf("update tag: %w", err)
	}

	return tag, nil
}

// Delete удаляет тег пользователя, он снимается со всех задач. Нет тега → model.ErrNotFound.
func (r *TagRepository) Delete(ctx context.Context, userID, id int64) error {
	res, err := r.db.Exec(ctx, _tagDeleteSQL, id, userID)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("tag %d %w", id, model.ErrNotFound)
	}

	return nil
}
