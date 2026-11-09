package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SummerXaa-Z/agent-harbor/internal/clients"
	"github.com/SummerXaa-Z/agent-harbor/internal/policy"
)

// mcpServerEntry is one server from a client config or portable catalog,
// kept raw so credential references survive untouched.
type mcpServerEntry struct {
	Name string
	Raw  map[string]any
}

// scanSources reads MCP server configs in priority order (component design
// section 2): an explicit --from file, the portable catalog, Claude Code's
// user config, then a project .mcp.json. First sighting of a name wins.
func scanSources(from string) ([]mcpServerEntry, []string, error) {
	var sources []string
	if from != "" {
		sources = []string{from}
	} else {
		sources = defaultSourcePaths()
	}
	seen := map[string]bool{}
	entries := []mcpServerEntry{}
	used := []string{}
	for _, source := range sources {
		expanded := expandTilde(source)
		found, err := readMCPServersFile(expanded)
		if err != nil {
			continue // a missing or unreadable source is not an error
		}
		used = append(used, expanded)
		for _, entry := range found {
			if !seen[entry.Name] {
				seen[entry.Name] = true
				entries = append(entries, entry)
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, used, nil
}

func defaultSourcePaths() []string {
	return []string{
		"~/.agent-shared/mcp/servers.json",
		"~/.claude.json",
		"./.mcp.json",
	}
}

func readMCPServersFile(path string) ([]mcpServerEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	servers, _ := document["mcpServers"].(map[string]any)
	entries := []mcpServerEntry{}
	for name, value := range servers {
		if raw, ok := value.(map[string]any); ok {
			entries = append(entries, mcpServerEntry{Name: name, Raw: raw})
		}
	}
	return entries, nil
}

// entryEndpoint classifies an entry: HTTP targets can route through the
// harbor gateway today; stdio servers stay direct until the M2 wrapper.
func entryEndpoint(entry mcpServerEntry) (string, bool) {
	if url, ok := entry.Raw["url"].(string); ok && strings.TrimSpace(url) != "" {
		return strings.TrimSpace(url), true
	}
	return "", false
}

// entryEnv returns the entry's env block (credential values may hide here;
// they only migrate with --confirm).
func entryEnv(entry mcpServerEntry) map[string]string {
	env := map[string]string{}
	if raw, ok := entry.Raw["env"].(map[string]any); ok {
		for key, value := range raw {
			if text, ok := value.(string); ok {
				env[key] = text
			}
		}
	}
	return env
}

type importTarget struct {
	Name string
	ID   string
}

type importState struct {
	Targets map[string]string `json:"targets"` // server name -> harbor target agent id
	Clients map[string]string `json:"clients"` // client key  -> harbor caller agent id
}

const defaultStateFile = ".harbor-state.json"

func cmdImport(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("import", stderr)
	clientFlags := addClientFlags(flags)
	from := flags.String("from", "", "explicit source config (default: auto-discover)")
	confirm := flags.Bool("confirm", false, "migrate upstream credentials into harbor's encrypted storage")
	output := flags.String("output", "harbor.yaml", "generated policy file")
	statePath := flags.String("state", defaultStateFile, "import state file (name->id mappings)")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	entries, used, err := scanSources(*from)
	if err != nil {
		fmt.Fprintf(stderr, "harbor import: %v\n", err)
		return 2
	}
	if len(entries) == 0 {
		fmt.Fprintln(stdout, "no MCP servers found in any known source.")
		fmt.Fprintln(stdout, "start from scratch instead: harbor init")
		return 0
	}
	fmt.Fprintf(stdout, "scanned %d source(s): %s\n", len(used), strings.Join(used, ", "))

	client := clientFromFlags(clientFlags)
	httpTargets := []mcpServerEntry{}
	stdioTargets := []mcpServerEntry{}
	for _, entry := range entries {
		if _, ok := entryEndpoint(entry); ok {
			httpTargets = append(httpTargets, entry)
		} else {
			stdioTargets = append(stdioTargets, entry)
		}
	}
	fmt.Fprintf(stdout, "found %d HTTP target(s), %d stdio (stdio stays direct until the M2 wrapper)\n",
		len(httpTargets), len(stdioTargets))
	for _, entry := range stdioTargets {
		fmt.Fprintf(stdout, "  stdio: %s\n", entry.Name)
	}
	if len(httpTargets) == 0 {
		fmt.Fprintln(stdout, "nothing to import for the gateway; no policy written")
		return 0
	}

	state := importState{Targets: map[string]string{}, Clients: map[string]string{}}
	for _, entry := range httpTargets {
		endpoint, _ := entryEndpoint(entry)
		credentials := map[string]string{}
		if *confirm {
			for key, value := range entryEnv(entry) {
				credentials[key] = value
			}
		} else if len(entryEnv(entry)) > 0 {
			fmt.Fprintf(stdout, "  note: %s carries %d env value(s); rerun with --confirm to migrate them\n",
				entry.Name, len(entryEnv(entry)))
		}
		body := map[string]any{
			"tenantId":      "default",
			"workspaceId":   "default",
			"name":          entry.Name,
			"channelType":   "mcp",
			"channelConfig": map[string]any{"endpoint": endpoint},
			"credentials":   credentials,
			"status":        "active",
		}
		status, envelope, err := client.doJSON(http.MethodPost, "/api/v1/agents", body)
		if err != nil {
			fmt.Fprintf(stderr, "harbor import: create target %s: %v\n", entry.Name, err)
			return 4
		}
		if status != http.StatusCreated {
			fmt.Fprintf(stderr, "harbor import: create target %s: %s: %s\n", entry.Name, envelope.Error, envelope.Message)
			return exitForError(status, envelope)
		}
		var created struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(envelope.Data, &created); err != nil || created.ID == "" {
			fmt.Fprintf(stderr, "harbor import: create target %s: unexpected response\n", entry.Name)
			return 4
		}
		state.Targets[entry.Name] = created.ID
		fmt.Fprintf(stdout, "  target %-20s %s -> %s\n", entry.Name, endpoint, created.ID)
	}

	// One caller agent per installed client, with a one-time key.
	installed, err := clients.DetectInstalled()
	if err != nil {
		fmt.Fprintf(stderr, "harbor import: %v\n", err)
		return 2
	}
	for _, spec := range installed {
		body := map[string]any{
			"tenantId":    "default",
			"workspaceId": "default",
			"name":        "caller-" + spec.Key,
			"channelType": "local",
			"status":      "active",
		}
		status, envelope, err := client.doJSON(http.MethodPost, "/api/v1/agents", body)
		if err != nil || status != http.StatusCreated {
			fmt.Fprintf(stderr, "harbor import: create caller for %s: %v\n", spec.Key, err)
			return exitForError(status, envelope)
		}
		var created struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(envelope.Data, &created)
		state.Clients[spec.Key] = created.ID
	}

	if err := writeGeneratedPolicy(*output, state); err != nil {
		fmt.Fprintf(stderr, "harbor import: %v\n", err)
		return 2
	}
	if err := writeState(*statePath, state); err != nil {
		fmt.Fprintf(stderr, "harbor import: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "wrote %s (open mode, %d target(s) in circle) and %s\n", *output, len(state.Targets), *statePath)
	issueKeysForClients(client, state, installed, stdout, stderr)
	return 0
}

// writeGeneratedPolicy emits the import-as-baseline harbor.yaml: one open
// workspace containing every imported target - zero behavior change.
func writeGeneratedPolicy(path string, state importState) error {
	var builder strings.Builder
	builder.WriteString("# generated by harbor import - import as baseline: everything you already use\n")
	builder.WriteString("# stays allowed inside workspace main; tighten with deny rules or flip strict.\n")
	builder.WriteString("version: 1\n\nworkspaces:\n  - id: main\n    strict: false\n    targets:\n")
	names := make([]string, 0, len(state.Targets))
	for name := range state.Targets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(&builder, "      - id: %s\n", state.Targets[name])
	}
	builder.WriteString("\nagents:\n")
	clientKeys := make([]string, 0, len(state.Clients))
	for key := range state.Clients {
		clientKeys = append(clientKeys, key)
	}
	sort.Strings(clientKeys)
	for _, key := range clientKeys {
		fmt.Fprintf(&builder, "  - id: %s\n    workspace: main\n", state.Clients[key])
	}
	if _, err := policy.Parse(strings.NewReader(builder.String())); err != nil {
		// The generated file must always validate; this is a programming error.
		return fmt.Errorf("generated policy failed validation: %w", err)
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func writeState(path string, state importState) error {
	encoded, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0o644)
}

func loadState(path string) (importState, error) {
	state := importState{Targets: map[string]string{}, Clients: map[string]string{}}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, fmt.Errorf("no import state at %s - run harbor import first", path)
		}
		return state, err
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return state, fmt.Errorf("parse %s: %w", path, err)
	}
	return state, nil
}

func issueKeysForClients(client *cliClient, state importState, installed []clients.ClientSpec, stdout, stderr io.Writer) {
	for _, spec := range installed {
		agentID := state.Clients[spec.Key]
		if agentID == "" {
			continue
		}
		fmt.Fprintf(stdout, "\nconnect %s:\n", spec.Key)
		code := cmdKeyIssue([]string{agentID}, stdout, stderr)
		if code != 0 {
			return
		}
		fmt.Fprintf(stdout, "  harbor connect %s --agent-key <key-above>\n", spec.Key)
	}
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
