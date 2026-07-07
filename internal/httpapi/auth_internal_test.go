package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/store"
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

func TestRecordConsoleLoginFailurePrunesExpiredClients(t *testing.T) {
	now := time.Date(2026, 7, 8, 6, 20, 0, 0, time.UTC)
	server := New(store.NewMemory(), WithClock(func() time.Time { return now }))
	server.loginFailures = map[string]consoleLoginFailure{
		"expired-client": {Count: 5, WindowEnds: now.Add(-time.Second)},
		"active-client":  {Count: 3, WindowEnds: now.Add(time.Minute)},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "203.0.113.25:44124"
	server.recordConsoleLoginFailure(req)

	if _, exists := server.loginFailures["expired-client"]; exists {
		t.Fatalf("expired login failure record should be pruned: %#v", server.loginFailures)
	}
	if failure, exists := server.loginFailures["active-client"]; !exists || failure.Count != 3 {
		t.Fatalf("active login failure record should be preserved, got exists=%v failure=%#v", exists, failure)
	}
	if failure, exists := server.loginFailures["203.0.113.25"]; !exists || failure.Count != 1 || !failure.WindowEnds.Equal(now.Add(consoleLoginFailureWindow)) {
		t.Fatalf("new login failure record should be tracked with a fresh window, got exists=%v failure=%#v", exists, failure)
	}
}
