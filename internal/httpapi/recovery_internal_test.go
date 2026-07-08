package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONPanicRecoveryWritesUniformErrorEnvelope(t *testing.T) {
	previousLogger := recoveredPanicLogger
	recoveredPanicLogger = func(*http.Request, any, []byte) {}
	t.Cleanup(func() {
		recoveredPanicLogger = previousLogger
	})

	handler := jsonPanicRecovery(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("secret panic detail")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic recovery status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("panic recovery content type = %q, want application/json", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("panic recovery content type options = %q, want nosniff", got)
	}
	if strings.Contains(rec.Body.String(), "secret panic detail") {
		t.Fatalf("panic recovery response leaked panic detail: %s", rec.Body.String())
	}

	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode panic recovery envelope: %v body=%s", err, rec.Body.String())
	}
	if env.Code != http.StatusInternalServerError || env.Error != "INTERNAL_ERROR" || env.Message != "internal server error" {
		t.Fatalf("unexpected panic recovery envelope: %#v", env)
	}
}

func TestRecoveredPanicLogMessageRedactsPanicValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents?adminKey=secret", nil)

	message := recoveredPanicLogMessage(req, "secret panic detail")

	if strings.Contains(message, "secret panic detail") || strings.Contains(message, "adminKey=secret") {
		t.Fatalf("panic log message leaked sensitive value: %s", message)
	}
	if !strings.Contains(message, "panicType=string") || !strings.Contains(message, "method=POST") || !strings.Contains(message, "path=/api/v1/agents") {
		t.Fatalf("panic log message omitted safe context: %s", message)
	}
}

func TestJSONPanicRecoveryDoesNotRewriteCommittedResponse(t *testing.T) {
	previousLogger := recoveredPanicLogger
	recoveredPanicLogger = func(*http.Request, any, []byte) {}
	t.Cleanup(func() {
		recoveredPanicLogger = previousLogger
	})

	handler := jsonPanicRecovery(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("accepted before panic"))
		panic("late panic")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stream", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("committed response status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if got := rec.Body.String(); got != "accepted before panic" {
		t.Fatalf("committed response body = %q", got)
	}
}

func TestJSONPanicRecoveryPreservesAbortHandlerPanic(t *testing.T) {
	handler := jsonPanicRecovery(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	defer func() {
		if recovered := recover(); recovered != http.ErrAbortHandler {
			t.Fatalf("recovered panic = %#v, want http.ErrAbortHandler", recovered)
		}
	}()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/abort", nil))
}
