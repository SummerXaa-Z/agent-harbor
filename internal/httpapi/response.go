package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

type envelope struct {
	Code    int    `json:"code"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

const maxJSONBodyBytes int64 = 1 << 20

func writeJSON(w http.ResponseWriter, status int, data any) {
	setJSONResponseHeaders(w)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Code: 0, Data: data})
}

func writeError(w http.ResponseWriter, err error) {
	setJSONResponseHeaders(w)
	var appErr domain.AppError
	if errors.As(err, &appErr) {
		w.WriteHeader(appErr.Status)
		_ = json.NewEncoder(w).Encode(envelope{
			Code:    appErr.Status,
			Error:   appErr.Code,
			Message: appErr.Message,
		})
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(envelope{
		Code:    http.StatusInternalServerError,
		Error:   "INTERNAL_ERROR",
		Message: "internal server error",
	})
}

func setJSONResponseHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func decodeJSON(r *http.Request, out any) error {
	if err := requireJSONContentType(r); err != nil {
		return err
	}

	limitedBody := http.MaxBytesReader(nil, r.Body, maxJSONBodyBytes)
	defer limitedBody.Close()

	decoder := json.NewDecoder(limitedBody)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return domain.PayloadTooLarge("request body exceeds 1MiB")
		}
		return domain.BadRequest("INVALID_JSON", "request body must be valid JSON")
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return domain.PayloadTooLarge("request body exceeds 1MiB")
		}
		return domain.BadRequest("INVALID_JSON", "request body must contain a single JSON value")
	}
	return nil
}

func requireJSONContentType(r *http.Request) error {
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	if contentType == "" {
		return domain.UnsupportedMediaType("content type must be application/json")
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		return domain.UnsupportedMediaType("content type must be application/json")
	}
	return nil
}
