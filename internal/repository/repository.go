package repository

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

//go:embed migrations/*.sql
var _migrations embed.FS

// Migrate накатывает на БД все непримененные миграции из migrations/.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	fsys, err := fs.Sub(_migrations, "migrations")
	if err != nil {
		return fmt.Errorf("migrations fs: %w", err)
	}

	// Закрытие db не закрывает pool.
	db := stdlib.OpenDBFromPool(pool)
	defer func() {
		if err := db.Close(); err != nil {
			log.Error().Err(err).Msg("close migrations db")
		}
	}()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, fsys)
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	for _, r := range results {
		log.Info().Str("migration", r.Source.Path).Dur("duration", r.Duration).Msg("migration applied")
	}

	return nil
}
