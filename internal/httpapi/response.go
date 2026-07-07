package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Code: 0, Data: data})
}

func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
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

func decodeJSON(r *http.Request, out any) error {
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
	return nil
}
