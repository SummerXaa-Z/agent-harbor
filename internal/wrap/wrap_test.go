package wrap

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testPolicy: workspace circle with targets w1 + agent a1; deny anything
// with "delete" in the name; open mode for everything else.
const testPolicy = `version: 1
workspaces:
  - id: main
    strict: false
    targets:
      - id: w1
agents:
  - id: a1
    workspace: main
    deny:
      - capability: "w1.*delete*"
        reason: read-only playground
`

// TestHelperProcess is the fake stdio MCP echo server: re-exec of the test
// binary (the classic helper-process pattern - no go run, no PATH luck
// needed). It answers tools/call with the tool name so tests can assert the
// round trip.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_FAKE_MCP") != "1" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	writer := bufio.NewWriter(os.Stdout)
	for scanner.Scan() {
		var request map[string]any
		if json.Unmarshal(scanner.Bytes(), &request) != nil {
			continue
		}
		id, hasID := request["id"]
		if !hasID {
			continue
		}
		method, _ := request["method"].(string)
		if method != "tools/call" {
			continue
		}
		params, _ := request["params"].(map[string]any)
		name, _ := params["name"].(string)
		encodedID, _ := json.Marshal(id)
		fmt.Fprintf(writer, "{\"jsonrpc\":\"2.0\",\"id\":%s,\"result\":{\"echo\":\"%s\"}}\n", string(encodedID), name)
		writer.Flush()
	}
	os.Exit(0)
}

type wrapHarness struct {
	clientIn  io.WriteCloser
	clientOut *bufio.Reader
	pumpDone  chan error
	spoolDir  string
	policyAt  string
	pump      *Pump
}

func startWrap(t *testing.T, policyText string) *wrapHarness {
	t.Helper()
	dir := t.TempDir()
	policyPath := filepath.Join(dir, "harbor.yaml")
	if err := os.WriteFile(policyPath, []byte(policyText), 0o644); err != nil {
		t.Fatal(err)
	}
	child := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--")
	child.Env = append(os.Environ(), "GO_WANT_FAKE_MCP=1")
	childInPipe, err := child.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	childOutPipe, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatalf("start fake server: %v (go on PATH?)", err)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })

	clientReader, clientWriter := io.Pipe()
	clientOutReader, clientOutPipe := io.Pipe()

	spoolDir := filepath.Join(dir, "spool")
	pump := NewPump(Options{
		AgentID:    "a1",
		TargetID:   "w1",
		PolicyPath: policyPath,
		SpoolDir:   spoolDir,
		Now:        func() time.Time { return time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC) },
	})
	done := make(chan error, 1)
	go func() { done <- pump.Run(clientReader, clientOutPipe, childInPipe, childOutPipe) }()

	return &wrapHarness{
		clientIn:  clientWriter,
		clientOut: bufio.NewReader(clientOutReader),
		pumpDone:  done,
		spoolDir:  spoolDir,
		policyAt:  policyPath,
		pump:      pump,
	}
}

func (h *wrapHarness) send(t *testing.T, line string) {
	t.Helper()
	if _, err := h.clientIn.Write([]byte(line + "\n")); err != nil {
		t.Fatalf("send: %v", err)
	}
}

func (h *wrapHarness) readLine(t *testing.T) map[string]any {
	t.Helper()
	line, err := h.clientOut.ReadString('\n')
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &decoded); err != nil {
		t.Fatalf("decode response %q: %v", line, err)
	}
	return decoded
}

func (h *wrapHarness) spoolEvents(t *testing.T) []Event {
	t.Helper()
	events := []Event{}
	entries, err := os.ReadDir(h.spoolDir)
	if err != nil {
		return events
	}
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(h.spoolDir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
			if line == "" {
				continue
			}
			var event Event
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				t.Fatalf("decode spool %q: %v", line, err)
			}
			events = append(events, event)
		}
	}
	return events
}

func TestWrapPassesAllowedCallsThroughAndRecords(t *testing.T) {
	h := startWrap(t, testPolicy)
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_notes","arguments":{"q":"hello"}}}`)
	response := h.readLine(t)
	result, _ := resultContent(response)
	if result != "search_notes" {
		t.Fatalf("allowed call should round-trip through the child: %+v", response)
	}
	events := h.spoolEvents(t)
	if len(events) != 1 || events[0].Decision != "allowed" || events[0].Capability != "search_notes" {
		t.Fatalf("allowed event missing: %+v", events)
	}
	if events[0].Source != "wrap" || events[0].CallerID != "a1" || events[0].TargetID != "w1" {
		t.Fatalf("event identity fields wrong: %+v", events[0])
	}
}

func TestWrapDeniesWithoutReachingTheChild(t *testing.T) {
	h := startWrap(t, testPolicy)
	h.send(t, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"bulk_delete_notes","arguments":{"all":true}}}`)
	response := h.readLine(t)
	errObj, _ := response["error"].(map[string]any)
	if errObj == nil {
		t.Fatalf("expected a JSON-RPC error, got: %+v", response)
	}
	message, _ := errObj["message"].(string)
	if !strings.Contains(message, "Do not retry") || !strings.Contains(message, "read-only playground") {
		t.Fatalf("deny copy must be LLM-first with the reason: %+v", errObj)
	}
	data, _ := errObj["data"].(map[string]any)
	if data == nil || data["auditId"] == "" || data["policy"] == "" || data["reason"] != "read-only playground" {
		t.Fatalf("deny data must carry policy/reason/auditId: %+v", data)
	}
	events := h.spoolEvents(t)
	if len(events) != 1 || events[0].Decision != "denied" {
		t.Fatalf("deny event missing: %+v", events)
	}
	if !strings.HasPrefix(events[0].ArgsDigest, "sha256:") || len(events[0].RiskFlags) == 0 {
		t.Fatalf("denied event should carry the args digest and name risk flags: %+v", events[0])
	}
}

func TestWrapFailClosedWithoutPolicyAndNeverFailsOpenOnBadEdit(t *testing.T) {
	h := startWrap(t, testPolicy)
	// Corrupt the policy file: the last good snapshot keeps deciding.
	if err := os.WriteFile(h.policyAt, []byte("version: 9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_notes","arguments":{}}}`)
	if response := h.readLine(t); response["error"] == nil {
		t.Fatalf("allowed round-trip still expected with the cached policy: %+v", response)
	}
	// With no readable policy at all (fresh pump), every call is denied.
	dir := t.TempDir()
	pump := NewPump(Options{AgentID: "a1", TargetID: "w1", PolicyPath: filepath.Join(dir, "missing.yaml"), SpoolDir: filepath.Join(dir, "spool")})
	decision := pump.decide("anything")
	if decision.Allowed {
		t.Fatalf("wrap must never fail open: %+v", decision)
	}
}

func TestWrapEditsApplyOnTheNextCall(t *testing.T) {
	h := startWrap(t, testPolicy)
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"export_notes","arguments":{}}}`)
	if response := h.readLine(t); response["error"] != nil {
		t.Fatalf("export starts allowed in open mode: %+v", response)
	}
	// Operator tightens the file; mtime change is picked up without restart.
	tightened := strings.Replace(testPolicy, `      - capability: "w1.*delete*"`, `      - capability: "w1.*delete*"\n      - capability: "w1.export*"`, 1)
	tightened = strings.ReplaceAll(tightened, `\n`, "\n")
	if err := os.WriteFile(h.policyAt, []byte(tightened), 0o644); err != nil {
		t.Fatal(err)
	}
	// Different mtime: force by truncating the cached mod time.
	h.pump.forgetNextPolicyCacheForTest(t)
	h.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"export_notes","arguments":{}}}`)
	response := h.readLine(t)
	errObj, _ := response["error"].(map[string]any)
	if errObj == nil {
		t.Fatalf("the tightened deny must apply without restart: %+v", response)
	}
}

// forgetNextPolicyCacheForTest drops the cached policy stamp so the next
// decide() reloads even when the filesystem mtime granularity misses the
// edit interval.
func (p *Pump) forgetNextPolicyCacheForTest(t *testing.T) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.policyMod = time.Time{}
}

func resultContent(response map[string]any) (string, bool) {
	result, ok := response["result"].(map[string]any)
	if !ok {
		return "", false
	}
	echo, _ := result["echo"].(string)
	return echo, true
}

// Audit classification must not alter the response seen by the MCP client.
func TestWrapCompletionOutcomesPreserveResponses(t *testing.T) {
	cases := []struct {
		name    string
		result  string
		outcome string
	}{
		{"success without flag", `"result":{"content":[]}`, "ok"},
		{"explicit success", `"result":{"content":[],"isError":false}`, "ok"},
		{"tool error", `"result":{"content":[{"type":"text","text":"tool failed"}],"isError":true}`, "error"},
		{"protocol error", `"error":{"code":-32603,"message":"internal error"}`, "error"},
		{"nested flag is not tool error", `"result":{"structuredContent":{"isError":true}}`, "ok"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			policyPath := filepath.Join(dir, "harbor.yaml")
			if err := os.WriteFile(policyPath, []byte(testPolicy), 0o644); err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
			pump := NewPump(Options{
				AgentID: "a1", TargetID: "w1", PolicyPath: policyPath,
				SpoolDir: filepath.Join(dir, "spool"),
				Now:      func() time.Time { return now },
			})
			t.Cleanup(func() {
				if pump.spoolFile != nil {
					_ = pump.spoolFile.Close()
				}
			})
			request := `{"jsonrpc":"2.0","id":"call-1","method":"tools/call","params":{"name":"search_notes","arguments":{}}}`
			if _, deny := pump.interceptToolCall([]byte(request)); deny {
				t.Fatal("test call should be allowed")
			}
			now = now.Add(25 * time.Millisecond)
			response := `{ "jsonrpc": "2.0", "id": "call-1", ` + tc.result + " }\n"
			var output bytes.Buffer
			if err := pump.pumpChildToClient(strings.NewReader(response), &lockedWriter{target: &output}); err != nil {
				t.Fatal(err)
			}
			if output.String() != response {
				t.Fatalf("response changed: got %q, want %q", output.String(), response)
			}
			h := wrapHarness{spoolDir: pump.options.SpoolDir}
			events := h.spoolEvents(t)
			if len(events) != 1 {
				t.Fatalf("want one completion event, got %+v", events)
			}
			event := events[0]
			if event.Outcome != tc.outcome || event.Decision != "allowed" {
				t.Fatalf("want allowed/%s, got %+v", tc.outcome, event)
			}
			if event.Capability != "search_notes" || event.DurationMs != 25 {
				t.Fatalf("completion correlation changed: %+v", event)
			}
			pump.recordCompletion([]byte(response))
			if events := h.spoolEvents(t); len(events) != 1 {
				t.Fatalf("duplicate response created another event: %+v", events)
			}
		})
	}
}
