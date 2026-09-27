package model

import (
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

// ValidationError — некорректные входные данные, текст отдаётся клиенту.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string {
	return e.Msg
}

type (
	Tag struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color,omitempty"`
	}

	TagInput struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}

	Task struct {
		ID          int64      `json:"id"`
		Name        string     `json:"name"`
		Description string     `json:"description,omitempty"`
		Date        *time.Time `json:"date,omitempty"`
		NotifyAt    *time.Time `json:"notifyAt,omitempty"`
		Priority    int        `json:"priority"`
		IsCompleted bool       `json:"isCompleted"`
		Tags        []Tag      `json:"tags"`
		Notes       []Note     `json:"notes"`
	}

	// TaskInput — тело POST/PUT задачи. IsCompleted учитывается только в PUT.
	TaskInput struct {
		Name        string     `json:"name"`
		Description string     `json:"description"`
		Date        *time.Time `json:"date"`
		NotifyAt    *time.Time `json:"notifyAt"`
		Priority    int        `json:"priority"`
		TagIDs      []int64    `json:"tagIds"`
		IsCompleted bool       `json:"isCompleted"`
	}

	// TaskFilter — фильтры GET /v1/tasks, nil — фильтр не задан.
	// Даты — полуинтервал [StartDate, EndDate).
	TaskFilter struct {
		StartDate   *time.Time
		EndDate     *time.Time
		IsCompleted *bool
	}

	Note struct {
		ID     int64     `json:"id"`
		TaskID int64     `json:"taskId"`
		Text   string    `json:"text"`
		Date   time.Time `json:"date"`
	}

	// NoteInput — тело POST/PUT заметки. В PUT другой TaskID переносит заметку.
	NoteInput struct {
		TaskID int64  `json:"taskId"`
		Text   string `json:"text"`
	}
)
