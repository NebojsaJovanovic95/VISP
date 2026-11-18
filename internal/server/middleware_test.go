package server

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddleware(t *testing.T) {
	var output bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(originalOutput)

	handler := loggingMiddleware(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		},
	))

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	logOutput := output.String()

	for _, expected := range []string{
		"POST",
		"/test",
		"201",
	} {
		if !strings.Contains(logOutput, expected) {
			t.Errorf("expected log output to contain %q, got %q", expected, logOutput)
		}
	}
}
