package httpapi

import (
	"net/http"
	"testing"
)

func TestConsoleSessionCSRFProtectionCoversUnsafeMethods(t *testing.T) {
	tests := []struct {
		method string
		want   bool
	}{
		{method: http.MethodGet, want: false},
		{method: http.MethodHead, want: false},
		{method: http.MethodOptions, want: false},
		{method: http.MethodPost, want: true},
		{method: http.MethodPut, want: true},
		{method: http.MethodPatch, want: true},
		{method: http.MethodDelete, want: true},
	}

	for _, tt := range tests {
		if got := requiresCSRFProtection(tt.method); got != tt.want {
			t.Fatalf("requiresCSRFProtection(%s) = %v, want %v", tt.method, got, tt.want)
		}
	}
}
