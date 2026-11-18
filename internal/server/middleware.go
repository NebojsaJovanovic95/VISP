package server

import (
	"log"
	"net/http"
	"time"
)

// loggingMiddleware logs details about each HTTP request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		log.Printf(
			"%s %s %d %s",
			r.Method,
			r.URL.Path,
			recorder.status,
			time.Since(start),
		)
	})
}

// statusRecorder captures the HTTP response status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status code before forwarding it.
func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
