package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDoUpstreamRequestDoesNotFollowRedirects(t *testing.T) {
	var redirected atomic.Bool
	var sourceReceivedCredential atomic.Bool
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected.Store(true)
		if secret := r.Header.Get("X-AgentHarbor-Test-Credential"); secret != "" {
			t.Errorf("redirect destination received credential header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer destination.Close()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-AgentHarbor-Test-Credential") == "must-not-cross-redirect" {
			sourceReceivedCredential.Store(true)
		}
		w.Header().Set("Location", destination.URL+"/private")
		w.WriteHeader(http.StatusFound)
	}))
	defer source.Close()

	request, err := http.NewRequest(http.MethodPost, source.URL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0"}`))
	if err != nil {
		t.Fatalf("create upstream request: %v", err)
	}
	request.Header.Set("X-AgentHarbor-Test-Credential", "must-not-cross-redirect")
	response, err := doUpstreamRequest(request)
	if err != nil {
		t.Fatalf("perform upstream request: %v", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)

	if response.StatusCode != http.StatusFound {
		t.Fatalf("expected original redirect response, got %d", response.StatusCode)
	}
	if !sourceReceivedCredential.Load() {
		t.Fatal("original upstream did not receive configured credential header")
	}
	if redirected.Load() {
		t.Fatal("upstream client followed redirect")
	}
}
