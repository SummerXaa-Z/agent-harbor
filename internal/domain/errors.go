package domain

import "fmt"

type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func BadRequest(code, message string) AppError {
	return AppError{Status: 400, Code: code, Message: message}
}

func Conflict(code, message string) AppError {
	return AppError{Status: 409, Code: code, Message: message}
}

func TooManyRequests(code, message string) AppError {
	return AppError{Status: 429, Code: code, Message: message}
}

func PayloadTooLarge(message string) AppError {
	return AppError{Status: 413, Code: "PAYLOAD_TOO_LARGE", Message: message}
}

func Unauthorized(message string) AppError {
	return AppError{Status: 401, Code: "UNAUTHORIZED", Message: message}
}

func PermissionDenied(message string) AppError {
	return AppError{Status: 403, Code: "PERMISSION_DENIED", Message: message}
}

func UpstreamError(message string) AppError {
	return AppError{Status: 502, Code: "UPSTREAM_ERROR", Message: message}
}

func UpstreamConnectError(message string) AppError {
	return AppError{Status: 502, Code: "UPSTREAM_CONNECT_ERROR", Message: message}
}

func UpstreamDNSError(message string) AppError {
	return AppError{Status: 502, Code: "UPSTREAM_DNS_ERROR", Message: message}
}

func UpstreamTLSError(message string) AppError {
	return AppError{Status: 502, Code: "UPSTREAM_TLS_ERROR", Message: message}
}

func UpstreamTimeout(message string) AppError {
	return AppError{Status: 504, Code: "UPSTREAM_TIMEOUT", Message: message}
}

func NotFound(message string) AppError {
	return AppError{Status: 404, Code: "NOT_FOUND", Message: message}
}
