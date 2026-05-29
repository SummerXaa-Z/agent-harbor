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

func Unauthorized(message string) AppError {
	return AppError{Status: 401, Code: "UNAUTHORIZED", Message: message}
}

func PermissionDenied(message string) AppError {
	return AppError{Status: 403, Code: "PERMISSION_DENIED", Message: message}
}

func NotFound(message string) AppError {
	return AppError{Status: 404, Code: "NOT_FOUND", Message: message}
}
