package model

import "errors"

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
)
