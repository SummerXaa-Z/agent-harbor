package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// apiEnvelope mirrors the server response shape.
type apiEnvelope struct {
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
	Message string          `json:"message"`
}

type cliClient struct {
	baseURL    string
	adminKey   string
	httpClient *http.Client
}

func clientFromFlags(flags *userFlags) *cliClient {
	return &cliClient{
		baseURL:    strings.TrimRight(flags.url, "/"),
		adminKey:   flags.adminKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// userFlags carries the shared client flags; every API subcommand parses
// them so the surface stays uniform.
type userFlags struct {
	url      string
	adminKey string
}

func addClientFlags(flags *flag.FlagSet) *userFlags {
	user := &userFlags{}
	flags.StringVar(&user.url, "url", envOr("HARBOR_URL", "http://127.0.0.1:9090"), "server base URL")
	flags.StringVar(&user.adminKey, "admin-key", os.Getenv("HARBOR_ADMIN_KEY"), "admin key (sent as X-Admin-Key)")
	return user
}

// getJSON performs an authenticated GET and returns the decoded envelope.
func (c *cliClient) getJSON(path string) (apiEnvelope, error) {
	request, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return apiEnvelope{}, err
	}
	if c.adminKey != "" {
		request.Header.Set("X-Admin-Key", c.adminKey)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return apiEnvelope{}, err
	}
	defer response.Body.Close()
	var envelope apiEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return apiEnvelope{}, fmt.Errorf("decode response: %w", err)
	}
	return envelope, nil
}

// postJSON performs an authenticated JSON request and returns status plus envelope.
func (c *cliClient) doJSON(method string, path string, body any) (int, apiEnvelope, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, apiEnvelope{}, err
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return 0, apiEnvelope{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	if c.adminKey != "" {
		request.Header.Set("X-Admin-Key", c.adminKey)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return 0, apiEnvelope{}, err
	}
	defer response.Body.Close()
	var envelope apiEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return response.StatusCode, apiEnvelope{}, fmt.Errorf("decode response: %w", err)
	}
	return response.StatusCode, envelope, nil
}

// exitForError maps server responses onto the CLI exit codes: policy
// denials are a different exit than network trouble (UX design section 2).
func exitForError(status int, envelope apiEnvelope) int {
	if status == http.StatusForbidden || envelope.Code == http.StatusForbidden {
		return 3
	}
	return 4
}

type traceRow struct {
	Source       string    `json:"source"`
	Outcome      string    `json:"outcome,omitempty"`
	ToolName     string    `json:"toolName,omitempty"`
	ID           string    `json:"id"`
	RunID        string    `json:"runId,omitempty"`
	CallerID     string    `json:"callerAgentId,omitempty"`
	TargetID     string    `json:"targetAgentId"`
	CapabilityID string    `json:"capabilityId,omitempty"`
	RouteKey     string    `json:"routeKey,omitempty"`
	Decision     string    `json:"decision"`
	Reason       string    `json:"reason,omitempty"`
	ArgsDigest   string    `json:"argsDigest,omitempty"`
	RiskFlags    []string  `json:"riskFlags,omitempty"`
	RuleRef      string    `json:"ruleRef,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

func cmdAudit(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("audit", stderr)
	clientFlags := addClientFlags(flags)
	highRisk := flags.Bool("high-risk", false, "only high-risk (red-flagged) calls")
	follow := flags.Bool("follow", false, "stream new events as they arrive")
	jsonOut := flags.Bool("json", false, "raw JSON output")
	limit := flags.Int("limit", 50, "how many events to fetch")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	client := clientFromFlags(clientFlags)
	query := url.Values{}
	query.Set("limit", fmt.Sprintf("%d", *limit))
	if *highRisk {
		query.Set("highRisk", "true")
	}
	printed := map[string]bool{}
	for {
		envelope, err := client.getJSON("/api/v1/audit/traces?" + query.Encode())
		if err != nil {
			fmt.Fprintf(stderr, "harbor audit: %v\n", err)
			return 4
		}
		if envelope.Error != "" {
			fmt.Fprintf(stderr, "harbor audit: %s: %s\n", envelope.Error, envelope.Message)
			if envelope.Code == http.StatusForbidden {
				return 3
			}
			return 4
		}
		var rows []traceRow
		if err := json.Unmarshal(envelope.Data, &rows); err != nil {
			fmt.Fprintf(stderr, "harbor audit: decode traces: %v\n", err)
			return 4
		}
		newRows := 0
		for _, row := range rows {
			if printed[row.ID] {
				continue
			}
			printed[row.ID] = true
			newRows++
			if *jsonOut {
				fmt.Fprintln(stdout, string(mustMarshal(row)))
				continue
			}
			fmt.Fprintln(stdout, formatTraceRow(row))
		}
		if !*follow {
			return 0
		}
		if newRows == 0 {
			time.Sleep(2 * time.Second)
		}
	}
}

func formatTraceRow(row traceRow) string {
	decision := row.Decision
	if decision == "denied" {
		decision = "DENY "
	} else {
		decision = "allow"
	}
	line := fmt.Sprintf("%s  %s  %-18s -> %s  %s",
		row.CreatedAt.Local().Format("15:04:05"),
		decision,
		coalesce(row.CallerID, "-"),
		row.TargetID,
		coalesce(row.Reason, ""))
	if row.ToolName != "" {
		line += " " + row.ToolName
	}
	line += "  [" + coalesce(row.Source, "gateway") + "]"
	if row.Outcome != "" {
		line += " " + row.Outcome
	}
	if len(row.RiskFlags) > 0 {
		line += "  [" + strings.Join(row.RiskFlags, ",") + "]"
	}
	if row.RuleRef != "" {
		line += "  (" + row.RuleRef + ")"
	}
	return line
}

func cmdWhy(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("why", stderr)
	clientFlags := addClientFlags(flags)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	rest := flags.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "harbor why: exactly one trace id required")
		return 2
	}
	client := clientFromFlags(clientFlags)
	query := url.Values{}
	query.Set("traceId", rest[0])
	envelope, err := client.getJSON("/api/v1/audit/traces?" + query.Encode())
	if err != nil {
		fmt.Fprintf(stderr, "harbor why: %v\n", err)
		return 4
	}
	var rows []traceRow
	if err := json.Unmarshal(envelope.Data, &rows); err != nil {
		fmt.Fprintf(stderr, "harbor why: decode trace: %v\n", err)
		return 4
	}
	if len(rows) != 1 {
		fmt.Fprintf(stderr, "harbor why: trace %s not found\n", rest[0])
		return 2
	}
	row := rows[0]
	fmt.Fprintf(stdout, "trace    %s\n", row.ID)
	fmt.Fprintf(stdout, "when     %s\n", row.CreatedAt.Local().Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(stdout, "caller   %s -> %s (%s)\n", coalesce(row.CallerID, "-"), row.TargetID, coalesce(row.RouteKey, "call"))
	fmt.Fprintf(stdout, "decision %s\n", strings.ToUpper(row.Decision))
	fmt.Fprintf(stdout, "source   %s\n", coalesce(row.Source, "gateway"))
	if row.ToolName != "" {
		fmt.Fprintf(stdout, "tool     %s\n", row.ToolName)
	}
	if row.Outcome != "" {
		fmt.Fprintf(stdout, "outcome  %s\n", row.Outcome)
	}
	if row.Reason != "" {
		fmt.Fprintf(stdout, "reason   %s\n", row.Reason)
	}
	if row.RuleRef != "" {
		fmt.Fprintf(stdout, "rule     %s\n", row.RuleRef)
	}
	if len(row.RiskFlags) > 0 {
		fmt.Fprintf(stdout, "risk     %s\n", strings.Join(row.RiskFlags, ", "))
	}
	if row.ArgsDigest != "" {
		fmt.Fprintf(stdout, "args     %s (digest only; the payload is not stored)\n", row.ArgsDigest)
	}
	return 0
}

func cmdKey(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "harbor key: expected issue, revoke, list, or rotate")
		return 2
	}
	subcommand, rest := args[0], args[1:]
	switch subcommand {
	case "issue":
		return cmdKeyIssue(rest, stdout, stderr)
	case "revoke":
		return cmdKeyRevoke(rest, stdout, stderr)
	case "list":
		return cmdKeyList(rest, stdout, stderr)
	case "rotate":
		return cmdKeyRotate(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "harbor key: unknown subcommand %q\n", subcommand)
		return 2
	}
}

func cmdKeyIssue(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("key issue", stderr)
	clientFlags := addClientFlags(flags)
	name := flags.String("name", "", "key name (server defaults to dev-agent-key)")
	ttl := flags.Int64("ttl-seconds", 0, "key lifetime in seconds (0 = server default)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	rest := flags.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "harbor key issue: exactly one agent id required")
		return 2
	}
	client := clientFromFlags(clientFlags)
	body := map[string]any{"agentId": rest[0], "name": *name, "expiresInSeconds": *ttl}
	status, envelope, err := client.doJSON(http.MethodPost, "/api/v1/agent-keys", body)
	if err != nil {
		fmt.Fprintf(stderr, "harbor key issue: %v\n", err)
		return 4
	}
	if status != http.StatusCreated {
		fmt.Fprintf(stderr, "harbor key issue: %s: %s\n", envelope.Error, envelope.Message)
		return exitForError(status, envelope)
	}
	var created struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Prefix string `json:"prefix"`
	}
	if err := json.Unmarshal(envelope.Data, &created); err != nil {
		fmt.Fprintf(stderr, "harbor key issue: decode response: %v\n", err)
		return 4
	}
	fmt.Fprintln(stdout, "Agent key issued - shown once, store it now:")
	fmt.Fprintf(stdout, "  id     %s (prefix %s)\n", created.ID, created.Prefix)
	fmt.Fprintf(stdout, "  key    %s\n", created.Key)
	return 0
}

func cmdKeyRevoke(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("key revoke", stderr)
	clientFlags := addClientFlags(flags)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	rest := flags.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "harbor key revoke: exactly one key id required")
		return 2
	}
	client := clientFromFlags(clientFlags)
	status, envelope, err := client.doJSON(http.MethodDelete, "/api/v1/api-keys/"+url.PathEscape(rest[0]), nil)
	if err != nil {
		fmt.Fprintf(stderr, "harbor key revoke: %v\n", err)
		return 4
	}
	if status != http.StatusOK {
		fmt.Fprintf(stderr, "harbor key revoke: %s: %s\n", envelope.Error, envelope.Message)
		return exitForError(status, envelope)
	}
	fmt.Fprintf(stdout, "revoked %s\n", rest[0])
	return 0
}

func cmdKeyList(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("key list", stderr)
	clientFlags := addClientFlags(flags)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	rest := flags.Args()
	if len(rest) > 1 {
		fmt.Fprintln(stderr, "harbor key list: at most one agent id")
		return 2
	}
	client := clientFromFlags(clientFlags)
	envelope, err := client.getJSON("/api/v1/api-keys")
	if err != nil {
		fmt.Fprintf(stderr, "harbor key list: %v\n", err)
		return 4
	}
	var rows []map[string]any
	if err := json.Unmarshal(envelope.Data, &rows); err != nil {
		fmt.Fprintf(stderr, "harbor key list: decode response: %v\n", err)
		return 4
	}
	for _, row := range rows {
		fmt.Fprintln(stdout, string(mustMarshal(row)))
	}
	return 0
}

// cmdKeyRotate issues a replacement then revokes the old id: the two halves
// of rotation composed client-side because the server exposes both halves.
func cmdKeyRotate(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("key rotate", stderr)
	addClientFlags(flags) // registered for a uniform --help; the subcommands parse their own
	old := flags.String("old", "", "key id to revoke after issuing")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	rest := flags.Args()
	if len(rest) != 1 || *old == "" {
		fmt.Fprintln(stderr, "harbor key rotate: one agent id and --old <keyId> required")
		return 2
	}
	issue := []string{"issue", rest[0]}
	if code := cmdKeyIssue(issue, stdout, stderr); code != 0 {
		return code
	}
	revoke := []string{"revoke", *old}
	return cmdKeyRevoke(revoke, stdout, stderr)
}

func mustMarshal(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return encoded
}

func coalesce(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
