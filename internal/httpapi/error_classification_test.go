package httpapi

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestClassifyUpstreamErrorDetectsDNSFailure(t *testing.T) {
	err := &net.DNSError{Err: "no such host", Name: "missing.invalid", IsNotFound: true}

	appErr := classifyUpstreamError(context.Background(), err)

	if appErr.Code != "UPSTREAM_DNS_ERROR" {
		t.Fatalf("expected DNS classification, got %#v", appErr)
	}
}

func TestClassifyUpstreamErrorDetectsTimeout(t *testing.T) {
	appErr := classifyUpstreamError(context.Background(), context.DeadlineExceeded)

	if appErr.Code != "UPSTREAM_TIMEOUT" {
		t.Fatalf("expected timeout classification, got %#v", appErr)
	}
}

func TestShouldRetryUpstreamErrorStopsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if shouldRetryUpstreamError(ctx, &net.DNSError{Err: "no such host"}) {
		t.Fatalf("canceled context should stop retry")
	}
}

func TestSleepBeforeRetryStopsOnCanceledContextWithZeroBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if sleepBeforeRetry(ctx, 0) {
		t.Fatalf("zero-backoff sleep should still honor canceled context")
	}
}

func TestSleepBeforeRetryAllowsOpenContextWithZeroBackoff(t *testing.T) {
	if !sleepBeforeRetry(context.Background(), 0*time.Millisecond) {
		t.Fatalf("zero-backoff sleep should allow open context")
	}
}
