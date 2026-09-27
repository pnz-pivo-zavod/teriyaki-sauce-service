package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"

	"teriyaki-sauce-service/internal/api/rest/handler"
	"teriyaki-sauce-service/internal/api/rest/response"
)

// New собирает роутер со всеми middleware и маршрутами.
func New(auth func(http.Handler) http.Handler, tag *handler.TagHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer, logRequests, cors)
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusNotFound, "route not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	r.Get("/health", handler.Health)

	r.Route("/v1", func(r chi.Router) {
		r.Use(auth)

		r.Get("/me", handler.Me)

		r.Post("/tag", tag.Create)
		r.Get("/tags", tag.List)
		r.Put("/tag/{id}", tag.Update)
		r.Delete("/tag/{id}", tag.Delete)
	})

	return r
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", ww.Status()).
			Dur("duration", time.Since(start)).
			Msg("request")
	})
}

// ponytail: allow-all CORS, ограничить origin перед продом.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr := w.Header()
		hdr.Set("Access-Control-Allow-Origin", "*")
		hdr.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		hdr.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
