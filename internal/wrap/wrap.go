// Package wrap is the stdio MCP wrapper (M2): a pass-through proxy the agent
// client never notices, with a single interception point at tools/call.
// Decisions come from the same internal/policy engine the gateway uses,
// loaded from harbor.yaml on disk - the call path never touches the network.
// Audit events go to an append-only local spool that harbor sync uploads.
package wrap

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/policy"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
)

// mcpDenyCode sits in the JSON-RPC server-defined range; the error data
// carries the structured denial (policy rule, reason, audit id) next to the
// LLM-first message.
const mcpDenyCode = -32063

const (
	toolCallMethod = "tools/call"
	maxLineBytes   = 32 << 20
)

type Options struct {
	// AgentID and TargetID key into harbor.yaml (the policy's workspace
	// boundary and rules apply exactly as on the gateway).
	AgentID  string
	TargetID string
	// PolicyPath is read fresh (mtime-checked) on every call: the operator
	// edits harbor.yaml and the next tool call already follows it.
	PolicyPath string
	// SpoolDir receives one JSONL line per audited event (append-only;
	// harbor sync checkpoints acknowledged bytes without truncating active files).
	SpoolDir string
	// ReportError receives local audit failures without interrupting tool calls.
	// The default reports to stderr; it never writes to the MCP stdout stream.
	ReportError func(error)
	// Clock is a seam for tests.
	Now func() time.Time
}

// Pump wires the client's stdio to the child MCP server. The zero
// interception surface: every request, response, and notification passes
// through byte-for-byte except tools/call requests.
type Pump struct {
	options Options

	mu        sync.Mutex
	policyMod time.Time
	loaded    *policy.Policy
	spoolFile *os.File
	pendingMu sync.Mutex
	pending   map[string]pendingCall
}

type pendingCall struct {
	tool       string
	argsDigest string
	startedAt  time.Time
}

func NewPump(options Options) *Pump {
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.ReportError == nil {
		options.ReportError = func(err error) { fmt.Fprintf(os.Stderr, "harbor-wrap: audit spool: %v\n", err) }
	}
	return &Pump{options: options, pending: map[string]pendingCall{}}
}

// lockedWriter serializes writes to the client stdout: child responses and
// wrap-generated denials share it.
type lockedWriter struct {
	mu     sync.Mutex
	target io.Writer
}

func (w *lockedWriter) writeLine(line []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := w.target.Write(line); err != nil {
		return err
	}
	_, err := w.target.Write([]byte("\n"))
	return err
}

// Run copies client->child (intercepting tools/call) and child->client
// (recording outcomes) until either side closes. It returns when the
// transport ends; child shutdown is the caller's responsibility.
func (p *Pump) Run(clientIn io.Reader, clientOut io.Writer, childIn io.Writer, childOut io.Reader) error {
	client := &lockedWriter{target: clientOut}
	done := make(chan error, 2)
	go func() { done <- p.pumpClientToChild(clientIn, childIn, client) }()
	go func() { done <- p.pumpChildToClient(childOut, client) }()
	first := <-done
	return first
}

func (p *Pump) pumpClientToChild(clientIn io.Reader, childIn io.Writer, client *lockedWriter) error {
	scanner := bufio.NewScanner(clientIn)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	writer := bufio.NewWriter(childIn)
	for scanner.Scan() {
		line := scanner.Bytes()
		if response, deny := p.interceptToolCall(line); deny {
			// The deny goes straight back to the client; the child
			// never sees the call.
			if err := client.writeLine([]byte(response)); err != nil {
				return err
			}
			continue
		}
		if _, err := writer.Write(line); err != nil {
			return err
		}
		if err := writer.WriteByte('\n'); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (p *Pump) pumpChildToClient(childOut io.Reader, client *lockedWriter) error {
	scanner := bufio.NewScanner(childOut)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	for scanner.Scan() {
		line := scanner.Bytes()
		p.recordCompletion(line)
		if err := client.writeLine(line); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// wireRequest is the shape the pump needs; unknown fields pass through
// untouched because the raw line is forwarded, never re-encoded.
type wireRequest struct {
	ID     *json.RawMessage `json:"id"`
	Method string           `json:"method"`
	Params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"params"`
}

// interceptToolCall inspects a client line; for tools/call requests it
// decides locally and, on denial, returns the JSON-RPC error line to emit
// instead. Allowed calls are returned as (nil, false) and forwarded as-is
// with the pending map remembering them for outcome recording.
func (p *Pump) interceptToolCall(line []byte) (string, bool) {
	var request wireRequest
	if err := json.Unmarshal(line, &request); err != nil {
		return "", false // not JSON (or a comment) - forward untouched
	}
	if request.Method != toolCallMethod || request.ID == nil {
		return "", false // notifications and other requests pass through
	}
	tool := request.Params.Name
	decision := p.decide(tool)
	if decision.Allowed {
		id := string(*request.ID)
		p.pendingMu.Lock()
		p.pending[id] = pendingCall{tool: tool, argsDigest: digestArguments(request.Params.Arguments), startedAt: p.options.Now()}
		p.pendingMu.Unlock()
		return "", false
	}
	auditID := security.NewID("trc")
	p.spool(Event{
		ID:         auditID,
		CallerID:   p.options.AgentID,
		TargetID:   p.options.TargetID,
		Capability: tool,
		Method:     toolCallMethod,
		Decision:   "denied",
		Reason:     decision.Reason,
		RuleRef:    decision.RuleRef,
		ArgsDigest: digestArguments(request.Params.Arguments),
		RiskFlags:  policy.NameRiskFlags(tool),
		CreatedAt:  p.options.Now().UTC(),
		Source:     "wrap",
	})
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      *request.ID,
		"error": map[string]any{
			"code": mcpDenyCode,
			"message": fmt.Sprintf(
				"%s. Do not retry; ask the operator if you need this capability. (audit %s)",
				decision.Reason, auditID),
			"data": map[string]any{
				"policy":  decision.RuleRef,
				"reason":  decision.Reason,
				"auditId": auditID,
			},
		},
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return "", false
	}
	return string(encoded), true
}

// recordCompletion correlates a child response with its pending call and
// writes the allowed-event audit record.
func (p *Pump) recordCompletion(line []byte) {
	var response struct {
		ID     *json.RawMessage `json:"id"`
		Result json.RawMessage  `json:"result"`
		Error  *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(line, &response); err != nil || response.ID == nil {
		return
	}
	id := string(*response.ID)
	p.pendingMu.Lock()
	call, ok := p.pending[id]
	delete(p.pending, id)
	p.pendingMu.Unlock()
	if !ok {
		return
	}
	outcome := "ok"
	if response.Error != nil {
		outcome = "error"
	} else {
		// MCP tool execution failures are successful JSON-RPC responses
		// with result.isError set, not top-level protocol errors.
		var result struct {
			IsError bool `json:"isError"`
		}
		if json.Unmarshal(response.Result, &result) == nil && result.IsError {
			outcome = "error"
		}
	}
	p.spool(Event{
		ID:         security.NewID("trc"),
		CallerID:   p.options.AgentID,
		TargetID:   p.options.TargetID,
		Capability: call.tool,
		Method:     toolCallMethod,
		Decision:   "allowed",
		Reason:     "wrap passthrough",
		ArgsDigest: call.argsDigest,
		RiskFlags:  policy.NameRiskFlags(call.tool),
		DurationMs: p.options.Now().Sub(call.startedAt).Milliseconds(),
		Outcome:    outcome,
		CreatedAt:  p.options.Now().UTC(),
		Source:     "wrap",
	})
}

// decide loads harbor.yaml when its mtime changed - the operator's edits
// apply on the very next call without restarting anything.
func (p *Pump) decide(tool string) policy.Decision {
	p.mu.Lock()
	defer p.mu.Unlock()
	info, err := os.Stat(p.options.PolicyPath)
	if err == nil && (p.loaded == nil || !info.ModTime().Equal(p.policyMod)) {
		if parsed, parseErr := policy.LoadFile(p.options.PolicyPath); parseErr == nil {
			p.loaded = parsed
			p.policyMod = info.ModTime()
		}
		// A bad edit never takes the tool down: the last good policy
		// keeps deciding (the UX design's never-crash-on-bad-file rule).
	}
	if p.loaded == nil {
		return policy.Decision{Allowed: false, Reason: "no harbor.yaml loaded; wrap refuses to fail open"}
	}
	return p.loaded.Decide(policy.Request{
		Agent:  p.options.AgentID,
		Target: p.options.TargetID,
		Tool:   tool,
		Action: policy.ActionInvoke,
	})
}

// Event is the spool record: field names match the server's TraceEvent JSON
// so harbor sync can upload without translation.
type Event struct {
	ID         string    `json:"id"`
	CallerID   string    `json:"callerAgentId"`
	TargetID   string    `json:"targetAgentId"`
	Capability string    `json:"capabilityId"`
	Method     string    `json:"routeKey"`
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason,omitempty"`
	RuleRef    string    `json:"ruleRef,omitempty"`
	ArgsDigest string    `json:"argsDigest,omitempty"`
	RiskFlags  []string  `json:"riskFlags,omitempty"`
	DurationMs int64     `json:"durationMs,omitempty"`
	Outcome    string    `json:"outcome,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	Source     string    `json:"source,omitempty"`
}

func (p *Pump) spool(event Event) {
	p.mu.Lock()
	err := p.writeSpool(event)
	p.mu.Unlock()
	if err != nil {
		p.options.ReportError(err)
	}
}

// writeSpool runs under mu. Each pump owns a unique append-only file so
// independent wrapper sessions cannot interleave or reuse checkpoint offsets.
func (p *Pump) writeSpool(event Event) error {
	if p.spoolFile == nil {
		if err := os.MkdirAll(p.options.SpoolDir, 0o700); err != nil {
			return err
		}
		file, err := os.CreateTemp(p.options.SpoolDir, "wrap-"+event.CreatedAt.Format("20060102-150405")+"-*.jsonl")
		if err != nil {
			return err
		}
		p.spoolFile = file
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}
	line := append(encoded, '\n')
	n, err := p.spoolFile.Write(line)
	if err != nil {
		return err
	}
	if n != len(line) {
		return io.ErrShortWrite
	}
	return nil
}

// digestArguments records the arguments as a digest only - the audit trail
// must not become a second copy of the payload data.
func digestArguments(arguments json.RawMessage) string {
	if len(arguments) == 0 {
		return ""
	}
	sum := sha256.Sum256(arguments)
	return "sha256:" + hex.EncodeToString(sum[:])
}
