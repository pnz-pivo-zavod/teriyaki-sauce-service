package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	tu "github.com/mymmrac/telego/telegoutil"

	"teriyaki-sauce-service/internal/api/rest/response"
)

const (
	_scheme = "tma "
	_maxAge = 24 * time.Hour
)

var (
	ErrInvalidInitData = errors.New("invalid init data")
	ErrExpired         = errors.New("init data expired")
)

type ctxKey struct{}

// Validate проверяет подпись и свежесть initData Telegram Mini App
// и возвращает Telegram ID пользователя.
func Validate(botToken, initData string, now time.Time) (int64, error) {
	values, err := tu.ValidateWebAppData(botToken, initData)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrInvalidInitData, err)
	}

	authDate, err := strconv.ParseInt(values.Get(tu.WebAppAuthDate), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: auth_date: %w", ErrInvalidInitData, err)
	}

	if now.Sub(time.Unix(authDate, 0)) > _maxAge {
		return 0, ErrExpired
	}

	var user struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(values.Get(tu.WebAppUser)), &user); err != nil || user.ID == 0 {
		return 0, fmt.Errorf("%w: no user", ErrInvalidInitData)
	}

	return user.ID, nil
}

// Middleware кладёт в context ID пользователя из заголовка `Authorization: tma <initData>`.
// devUserID != 0 отключает проверку: все запросы идут от этого пользователя.
func Middleware(botToken string, devUserID int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := devUserID
			if userID == 0 {
				initData, ok := strings.CutPrefix(r.Header.Get("Authorization"), _scheme)
				if !ok {
					response.Error(w, http.StatusUnauthorized, "missing tma authorization")
					return
				}

				var err error
				if userID, err = Validate(botToken, initData, time.Now()); err != nil {
					response.Error(w, http.StatusUnauthorized, err.Error())
					return
				}
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, userID)))
		})
	}
}

// UserID возвращает ID пользователя, положенный Middleware.
func UserID(ctx context.Context) int64 {
	id, _ := ctx.Value(ctxKey{}).(int64)
	return id
}
