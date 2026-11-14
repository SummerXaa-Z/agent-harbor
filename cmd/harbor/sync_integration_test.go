package main

import (
	"bufio"
	"bytes"
	"context"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"github.com/SummerXaa-Z/agent-harbor/internal/wrap"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The actual wrapper -> retained spool -> CLI -> authenticated API -> scoped
// store -> audit/why path, including an offline attempt and duplicate replay.
func TestWrapSyncAuditEndToEnd(t *testing.T) {
	dir := t.TempDir()
	spool := filepath.Join(dir, "spool")
	policy := filepath.Join(dir, "harbor.yaml")
	if err := os.WriteFile(policy, []byte("version: 1\nworkspaces:\n  - id: main\n    targets:\n      - id: target\nagents:\n  - id: caller\n    workspace: main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	clientIn, clientWrite := io.Pipe()
	clientRead, clientOut := io.Pipe()
	childRead, childIn := io.Pipe()
	childOut, childWrite := io.Pipe()
	for _, c := range []io.Closer{clientIn, clientWrite, clientRead, clientOut, childRead, childIn, childOut, childWrite} {
		defer c.Close()
	}
	pump := wrap.NewPump(wrap.Options{AgentID: "caller", TargetID: "target", PolicyPath: policy, SpoolDir: spool, ReportError: func(err error) { t.Errorf("spool: %v", err) }})
	done := make(chan error, 1)
	go func() { done <- pump.Run(clientIn, clientOut, childIn, childOut) }()
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_notes","arguments":{"q":"private"}}}` + "\n"
	response := `{"jsonrpc":"2.0","id":1,"result":{"isError":true,"content":[]}}` + "\n"
	childDone := make(chan error, 1)
	go func() {
		line, err := bufio.NewReader(childRead).ReadString('\n')
		if err == nil && line != request {
			err = io.ErrUnexpectedEOF
		}
		if err == nil {
			_, err = io.WriteString(childWrite, response)
		}
		childDone <- err
	}()
	if _, err := io.WriteString(clientWrite, request); err != nil {
		t.Fatal(err)
	}
	if line, err := bufio.NewReader(clientRead).ReadString('\n'); err != nil || line != response {
		t.Fatalf("response changed: %q %v", line, err)
	}
	if err := <-childDone; err != nil {
		t.Fatal(err)
	}
	clientWrite.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(spool, "wrap-*.jsonl"))
	if err != nil || len(files) != 1 {
		t.Fatalf("spool: %v %v", files, err)
	}
	before, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(before), "private") {
		t.Fatal("plaintext arguments entered audit spool")
	}
	offline := httptest.NewServer(nil)
	offlineURL := offline.URL
	offline.Close()
	var out, errs bytes.Buffer
	if code := run([]string{"sync", "--url", offlineURL, "--admin-key", "test-key", "--spool", spool}, &out, &errs); code != 4 {
		t.Fatalf("offline code %d: %s", code, &errs)
	}
	repo := store.NewMemory()
	for _, id := range []string{"caller", "target"} {
		if _, err := repo.CreateAgent(context.Background(), domain.Agent{ID: id, TenantID: "t1", WorkspaceID: "w1"}); err != nil {
			t.Fatal(err)
		}
	}
	server := httptest.NewServer(httpapi.New(repo, httpapi.WithAdminKey("test-key")).Router())
	defer server.Close()
	args := []string{"sync", "--url", server.URL, "--admin-key", "test-key", "--spool", spool}
	for _, a := range [][]string{args, args, append(append([]string{}, args...), "--replay")} {
		out.Reset()
		errs.Reset()
		if code := run(a, &out, &errs); code != 0 {
			t.Fatalf("sync %d: %s", code, &errs)
		}
	}
	events, err := repo.ListTraces(context.Background(), store.TraceFilter{ManagementScope: store.ManagementScope{TenantID: "t1", WorkspaceID: "w1"}})
	if err != nil || len(events) != 1 || events[0].Outcome != "error" || events[0].ToolName != "search_notes" || events[0].ArgsDigest == "" {
		t.Fatalf("events %+v %v", events, err)
	}
	for _, args := range [][]string{{"audit", "--url", server.URL, "--admin-key", "test-key", "--json"}, {"why", "--url", server.URL, "--admin-key", "test-key", events[0].ID}} {
		out.Reset()
		errs.Reset()
		if code := run(args, &out, &errs); code != 0 {
			t.Fatalf("read %d %s", code, &errs)
		}
		for _, want := range []string{"wrap", "error", "search_notes"} {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("missing %s in %s", want, &out)
			}
		}
	}
	after, err := os.ReadFile(files[0])
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("sync changed source spool")
	}
}
