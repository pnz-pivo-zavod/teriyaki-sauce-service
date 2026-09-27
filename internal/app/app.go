package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/rs/zerolog/log"

	"teriyaki-sauce-service/internal/api/rest"
	"teriyaki-sauce-service/internal/api/rest/handler"
	"teriyaki-sauce-service/internal/api/rest/router"
	"teriyaki-sauce-service/internal/auth"
	"teriyaki-sauce-service/internal/repository"
	"teriyaki-sauce-service/internal/service"
)

const _defaultAddr = ":8080"

// Run собирает зависимости и запускает HTTP-сервер. Блокируется до ошибки сервера.
func Run() error {
	ctx := context.Background()

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = _defaultAddr
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	pool, err := repository.NewPool(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := repository.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	authMW, err := newAuth()
	if err != nil {
		return err
	}

	var (
		tagRepo  = repository.NewTagRepository(pool)
		taskRepo = repository.NewTaskRepository(pool)
		noteRepo = repository.NewNoteRepository(pool)
	)

	var (
		tagHandler  = handler.NewTagHandler(service.NewTagService(tagRepo))
		taskHandler = handler.NewTaskHandler(service.NewTaskService(taskRepo, tagRepo))
		noteHandler = handler.NewNoteHandler(service.NewNoteService(noteRepo))
	)

	log.Info().Str("addr", addr).Msg("http server started")

	return rest.Run(addr, router.New(authMW, tagHandler, taskHandler, noteHandler))
}

func newAuth() (func(http.Handler) http.Handler, error) {
	botToken := os.Getenv("BOT_TOKEN")

	var devUserID int64
	if v := os.Getenv("DEV_USER_ID"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id == 0 {
			return nil, fmt.Errorf("DEV_USER_ID must be non-zero int64, got %q", v)
		}

		devUserID = id
		log.Warn().Int64("user_id", devUserID).Msg("DEV_USER_ID set: initData validation disabled")
	}

	if botToken == "" && devUserID == 0 {
		return nil, errors.New("BOT_TOKEN or DEV_USER_ID is required")
	}

	return auth.Middleware(botToken, devUserID), nil
}
