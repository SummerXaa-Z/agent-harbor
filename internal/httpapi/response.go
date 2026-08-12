package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	payload, err := io.ReadAll(limitedBody)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return domain.PayloadTooLarge("request body exceeds 1MiB")
		}
		return domain.BadRequest("INVALID_JSON", "request body must be valid JSON")
	}
	if err := decodeStrictJSON(payload, out); err != nil {
		return domain.BadRequest("INVALID_JSON", "request body must be valid, unambiguous JSON")
	}
	return nil
}

func decodeStrictJSON(payload []byte, out any) error {
	if err := validateUnambiguousJSON(payload); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	return decoder.Decode(out)
}

type jsonContainer struct {
	delim     json.Delim
	keys      map[string]struct{}
	expectKey bool
}

// validateUnambiguousJSON rejects multiple top-level values and duplicate
// object fields. encoding/json otherwise accepts duplicate fields using the
// last value, which is unsafe at authorization and management boundaries.
func validateUnambiguousJSON(payload []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	stack := []jsonContainer{}
	rootSeen := false

	consumeValue := func() error {
		if len(stack) == 0 {
			if rootSeen {
				return errors.New("JSON must contain a single value")
			}
			rootSeen = true
			return nil
		}
		container := &stack[len(stack)-1]
		if container.delim == '{' {
			if container.expectKey {
				return errors.New("JSON object field name is required")
			}
			container.expectKey = true
		}
		return nil
	}

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			if !rootSeen {
				return errors.New("JSON value is required")
			}
			if len(stack) != 0 {
				return io.ErrUnexpectedEOF
			}
			return nil
		}
		if err != nil {
			return err
		}

		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{', '[':
				if err := consumeValue(); err != nil {
					return err
				}
				container := jsonContainer{delim: delim}
				if delim == '{' {
					container.keys = map[string]struct{}{}
					container.expectKey = true
				}
				stack = append(stack, container)
			case '}', ']':
				if len(stack) == 0 ||
					(stack[len(stack)-1].delim == '{' && delim != '}') ||
					(stack[len(stack)-1].delim == '[' && delim != ']') {
					return fmt.Errorf("unexpected JSON delimiter %q", delim)
				}
				container := stack[len(stack)-1]
				if container.delim == '{' && !container.expectKey {
					return errors.New("JSON object field value is required")
				}
				stack = stack[:len(stack)-1]
			default:
				return fmt.Errorf("unexpected JSON delimiter %q", delim)
			}
			continue
		}

		if len(stack) > 0 {
			container := &stack[len(stack)-1]
			if container.delim == '{' && container.expectKey {
				key, ok := token.(string)
				if !ok {
					return errors.New("JSON object field name must be a string")
				}
				if _, exists := container.keys[key]; exists {
					return errors.New("JSON object contains a duplicate field")
				}
				container.keys[key] = struct{}{}
				container.expectKey = false
				continue
			}
		}
		if err := consumeValue(); err != nil {
			return err
		}
	}
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
