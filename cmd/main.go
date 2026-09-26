package main

import (
	"github.com/rs/zerolog/log"

	"teriyaki-sauce-service/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal().Err(err).Msg("app stopped")
	}
}
