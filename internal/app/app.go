package app

import (
	"os"

	"github.com/rs/zerolog/log"

	"teriyaki-sauce-service/internal/api/rest"
	"teriyaki-sauce-service/internal/api/rest/handler"
	"teriyaki-sauce-service/internal/api/rest/router"
)

const _defaultAddr = ":8080"

// Run собирает зависимости и запускает HTTP-сервер. Блокируется до ошибки сервера.
func Run() error {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = _defaultAddr
	}

	h := handler.New()

	log.Info().Str("addr", addr).Msg("http server started")

	return rest.Run(addr, router.New(h))
}
