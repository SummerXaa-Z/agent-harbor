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
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return domain.BadRequest("INVALID_JSON", "request body must be valid JSON")
	}
	return nil
}
