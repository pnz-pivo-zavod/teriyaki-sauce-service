package rest

import (
	"net/http"
	"time"
)

const _readHeaderTimeout = 10 * time.Second

// Run запускает HTTP-сервер на addr. Блокируется до ошибки сервера.
func Run(addr string, h http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: _readHeaderTimeout,
	}

	return srv.ListenAndServe()
}
