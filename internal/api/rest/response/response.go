package response

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

type envelope struct {
	Data  any     `json:"data"`
	Error *string `json:"error"`
}

// JSON пишет успешный ответ {"data": data, "error": null}.
func JSON(w http.ResponseWriter, status int, data any) {
	write(w, status, envelope{Data: data})
}

// Error пишет ответ с ошибкой {"data": null, "error": msg}.
func Error(w http.ResponseWriter, status int, msg string) {
	write(w, status, envelope{Error: &msg})
}

func write(w http.ResponseWriter, status int, body envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error().Err(err).Msg("write response")
	}
}
