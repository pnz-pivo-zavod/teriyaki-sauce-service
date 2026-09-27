package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"teriyaki-sauce-service/internal/api/rest"
	"teriyaki-sauce-service/internal/api/rest/handler"
	"teriyaki-sauce-service/internal/api/rest/router"
	"teriyaki-sauce-service/internal/repository"
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

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("create db pool: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	if err := repository.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	h := handler.New()

	log.Info().Str("addr", addr).Msg("http server started")

	return rest.Run(addr, router.New(h))
}
