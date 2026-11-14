package main

import (
	"encoding/json"
	"fmt"
	"github.com/SummerXaa-Z/agent-harbor/internal/wrap"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func syncTestLine(id string) string {
	e := wrap.Event{ID: id, CallerID: "caller", TargetID: "target", Capability: "read_notes", Method: "tools/call", Decision: "allowed", Outcome: "ok", Source: "wrap", CreatedAt: time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)}
	b, _ := json.Marshal(e)
	return string(b) + "\n"
}
func writeSyncSpool(t *testing.T, dir, data string) string {
	t.Helper()
	p := filepath.Join(dir, "wrap-test.jsonl")
	if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}
func syncTestServer(t *testing.T, receive func([]wrap.Event), status *int) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/audit/traces/ingest" || r.Method != "POST" || r.Header.Get("X-Admin-Key") != "test-key" {
			t.Errorf("wrong request: %s %s", r.Method, r.URL.Path)
		}
		var b struct {
			Events []wrap.Event `json:"events"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			t.Error(err)
		}
		if status != nil && *status != 200 {
			w.WriteHeader(*status)
			fmt.Fprintf(w, `{"code":%d,"error":"REJECTED"}`, *status)
			return
		}
		if receive != nil {
			receive(b.Events)
		}
		fmt.Fprintf(w, `{"code":0,"data":{"accepted":%d}}`, len(b.Events))
	}))
	t.Cleanup(s.Close)
	return s
}
func syncTestClient(s *httptest.Server) *cliClient {
	return &cliClient{baseURL: s.URL, adminKey: "test-key", httpClient: s.Client()}
}
func TestSyncResumeReplayAndPartialLine(t *testing.T) {
	d := t.TempDir()
	one, two := syncTestLine("one"), syncTestLine("two")
	p := writeSyncSpool(t, d, one+strings.TrimSuffix(two, "\n"))
	var ids []string
	c := syncTestClient(syncTestServer(t, func(es []wrap.Event) {
		for _, e := range es {
			ids = append(ids, e.ID)
		}
	}, nil))
	for _, want := range []int{1, 0} {
		if n, err := syncSpool(c, d, false); n != want || err != nil {
			t.Fatalf("%d %v", n, err)
		}
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("\n")
	_ = f.Close()
	if n, err := syncSpool(c, d, false); n != 1 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
	if n, err := syncSpool(c, d, true); n != 2 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
	if strings.Join(ids, ",") != "one,two,one,two" {
		t.Fatal(ids)
	}
	b, _ := os.ReadFile(p)
	if string(b) != one+two {
		t.Fatal("modified spool")
	}
}
func TestSyncFailurePreservesSpoolAndCursor(t *testing.T) {
	d := t.TempDir()
	line := syncTestLine("retry")
	p := writeSyncSpool(t, d, line)
	status := 503
	received := 0
	c := syncTestClient(syncTestServer(t, func(es []wrap.Event) { received += len(es) }, &status))
	if _, err := syncSpool(c, d, false); err == nil {
		t.Fatal("expected rejection")
	}
	entries, _ := os.ReadDir(syncCheckpointDir(d, c.baseURL, c.adminKey))
	if len(entries) != 0 {
		t.Fatal("advanced cursor")
	}
	status = 200
	if n, err := syncSpool(c, d, false); n != 1 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
	if received != 1 {
		t.Fatal(received)
	}
	b, _ := os.ReadFile(p)
	if string(b) != line {
		t.Fatal("altered spool")
	}
}
func TestSyncConcurrentAppendWaitsForNextInvocation(t *testing.T) {
	d := t.TempDir()
	p := writeSyncSpool(t, d, syncTestLine("first"))
	added := false
	c := syncTestClient(syncTestServer(t, func(es []wrap.Event) {
		if !added {
			f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0)
			if err != nil {
				t.Error(err)
				return
			}
			_, _ = f.WriteString(syncTestLine("second"))
			_ = f.Close()
			added = true
		}
	}, nil))
	for i := 0; i < 2; i++ {
		if n, err := syncSpool(c, d, false); n != 1 || err != nil {
			t.Fatalf("%d %v", n, err)
		}
	}
}
func TestSyncCorruptLineAndOversizedLineAreNotAcknowledged(t *testing.T) {
	for _, bad := range []string{"{broken}\n", strings.TrimSuffix(syncTestLine("unknown"), "}\n") + `,"futureField":true}` + "\n", strings.Repeat("x", syncMaxLine+1) + "\n"} {
		t.Run(fmt.Sprint(len(bad)), func(t *testing.T) {
			d := t.TempDir()
			writeSyncSpool(t, d, syncTestLine("valid")+bad)
			n := 0
			c := syncTestClient(syncTestServer(t, func(es []wrap.Event) { n += len(es) }, nil))
			if _, err := syncSpool(c, d, false); err == nil {
				t.Fatal("expected failure")
			}
			if n != 0 {
				t.Fatal("sent invalid batch")
			}
		})
	}
}
func TestSyncReplacedFileReplaysAndDestinationHasOwnCursor(t *testing.T) {
	d := t.TempDir()
	writeSyncSpool(t, d, syncTestLine("one"))
	received := 0
	receive := func(es []wrap.Event) { received += len(es) }
	c := syncTestClient(syncTestServer(t, receive, nil))
	if _, err := syncSpool(c, d, false); err != nil {
		t.Fatal(err)
	}
	writeSyncSpool(t, d, syncTestLine("two"))
	if n, err := syncSpool(c, d, false); n != 1 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
	if n, err := syncSpool(syncTestClient(syncTestServer(t, receive, nil)), d, false); n != 1 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
	if received != 3 {
		t.Fatal(received)
	}
	if syncCheckpointDir(d, c.baseURL, "rotated") == syncCheckpointDir(d, c.baseURL, c.adminKey) {
		t.Fatal("key namespace collision")
	}
}
func TestSyncRejectsIncompleteAcknowledgementAndRedirect(t *testing.T) {
	for _, response := range []string{`{"code":0,"data":{"accepted":0}}`, `{"code":0,"data":{}}`, strings.Repeat("x", syncMaxResponse+1)} {
		t.Run(fmt.Sprint(len(response)), func(t *testing.T) {
			d := t.TempDir()
			writeSyncSpool(t, d, syncTestLine("one"))
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, response) }))
			defer s.Close()
			if _, err := syncSpool(syncTestClient(s), d, false); err == nil {
				t.Fatal("invalid ack accepted")
			}
		})
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("followed redirect") }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	d := t.TempDir()
	writeSyncSpool(t, d, syncTestLine("one"))
	var out, errOut strings.Builder
	if code := cmdSync([]string{"--spool", d, "--url", redirect.URL, "--admin-key", "test-key"}, &out, &errOut); code != 4 {
		t.Fatal(code)
	}
}
func TestSyncBatchesAndCheckpointFailure(t *testing.T) {
	d := t.TempDir()
	var data strings.Builder
	for i := 0; i < syncBatchEvents+1; i++ {
		data.WriteString(syncTestLine(fmt.Sprint(i)))
	}
	writeSyncSpool(t, d, data.String())
	var batches []int
	c := syncTestClient(syncTestServer(t, func(es []wrap.Event) { batches = append(batches, len(es)) }, nil))
	if n, err := syncSpool(c, d, false); n != syncBatchEvents+1 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
	if len(batches) != 2 || batches[0] != syncBatchEvents || batches[1] != 1 {
		t.Fatal(batches)
	}
	broken := t.TempDir()
	writeSyncSpool(t, broken, syncTestLine("one"))
	dir := syncCheckpointDir(broken, c.baseURL, c.adminKey)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	cp := filepath.Join(dir, "wrap-test.jsonl.json")
	if err := os.Mkdir(cp, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := syncSpool(c, broken, true); err == nil {
		t.Fatal("checkpoint failure ignored")
	}
	if err := os.Remove(cp); err != nil {
		t.Fatal(err)
	}
	if n, err := syncSpool(c, broken, false); n != 1 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
}
func TestSyncCommandExitCodesAndReplay(t *testing.T) {
	d := t.TempDir()
	writeSyncSpool(t, d, syncTestLine("one"))
	status := 403
	s := syncTestServer(t, nil, &status)
	args := []string{"sync", "--spool", d, "--url", s.URL, "--admin-key", "test-key"}
	var out, errOut strings.Builder
	if code := run(args, &out, &errOut); code != 3 {
		t.Fatalf("%d %s", code, errOut.String())
	}
	status = 200
	for _, want := range []string{"synced 1 events", "synced 0 events"} {
		out.Reset()
		errOut.Reset()
		if code := run(args, &out, &errOut); code != 0 || !strings.Contains(out.String(), want) {
			t.Fatalf("%d %s %s", code, out.String(), errOut.String())
		}
	}
	out.Reset()
	if code := run(append(args, "--replay"), &out, &errOut); code != 0 || !strings.Contains(out.String(), "synced 1 events") {
		t.Fatalf("%d %s", code, errOut.String())
	}
	cp := filepath.Join(syncCheckpointDir(d, s.URL, "test-key"), "wrap-test.jsonl.json")
	if err := os.WriteFile(cp, []byte("bad checkpoint"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := run(args, &out, &errOut); code != 2 {
		t.Fatal(code)
	}
	if code := run(append(args, "--replay"), &out, &errOut); code != 0 {
		t.Fatalf("%d %s", code, errOut.String())
	}
}
func TestSyncNetworkFailureAndRetryAfterRestart(t *testing.T) {
	d := t.TempDir()
	writeSyncSpool(t, d, syncTestLine("one"))
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = c.Close()
	}))
	defer s.Close()
	var out, errOut strings.Builder
	if code := cmdSync([]string{"--spool", d, "--url", s.URL}, &out, &errOut); code != 4 {
		t.Fatalf("%d %s", code, errOut.String())
	}
	entries, _ := os.ReadDir(syncCheckpointDir(d, s.URL, ""))
	if len(entries) != 0 {
		t.Fatal("cursor advanced")
	}
	c := syncTestClient(syncTestServer(t, nil, nil))
	if n, err := syncSpool(c, d, false); n != 1 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
	restarted := &cliClient{baseURL: c.baseURL, adminKey: c.adminKey, httpClient: &http.Client{Timeout: time.Second}}
	if n, err := syncSpool(restarted, d, false); n != 0 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
}
func TestSyncCheckpointHashesAcknowledgedBytesNotLaterRewrite(t *testing.T) {
	d := t.TempDir()
	writeSyncSpool(t, d, syncTestLine("one"))
	rewritten := false
	var ids []string
	c := syncTestClient(syncTestServer(t, func(es []wrap.Event) {
		for _, e := range es {
			ids = append(ids, e.ID)
		}
		if !rewritten {
			writeSyncSpool(t, d, syncTestLine("two"))
			rewritten = true
		}
	}, nil))
	for i := 0; i < 2; i++ {
		if n, err := syncSpool(c, d, false); n != 1 || err != nil {
			t.Fatalf("%d %v", n, err)
		}
	}
	if strings.Join(ids, ",") != "one,two" {
		t.Fatal(ids)
	}
}
