package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/SummerXaa-Z/agent-harbor/internal/clients"
)

// cmdConnect wires a client to harbor: for writable registry entries it
// backs up the config and points each imported server name at the gateway
// (names stay stable, so tool ids do not change); print-only clients get a
// snippet to paste.
func cmdConnect(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("connect", stderr)
	clientFlags := addClientFlags(flags)
	agentKey := flags.String("agent-key", os.Getenv("HARBOR_AGENT_KEY"), "the caller agent key issued by harbor import")
	statePath := flags.String("state", defaultStateFile, "import state file")
	printOnly := flags.Bool("print", false, "print the snippet instead of editing the config")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	rest := flags.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "harbor connect: exactly one client key required (see harbor help)")
		return 2
	}
	spec, err := clients.Find(rest[0])
	if err != nil {
		fmt.Fprintf(stderr, "harbor connect: %v\n", err)
		return 2
	}
	state, err := loadState(*statePath)
	if err != nil {
		fmt.Fprintf(stderr, "harbor connect: %v\n", err)
		return 2
	}
	if *agentKey == "" && !*printOnly {
		fmt.Fprintln(stderr, "harbor connect: --agent-key (or HARBOR_AGENT_KEY) is required to wire a client")
		return 2
	}
	client := clientFromFlags(clientFlags)

	if *printOnly || !spec.Writable {
		if !*printOnly {
			fmt.Fprintf(stdout, "%s is print-only for now - paste this into %s:\n\n", spec.DisplayName, spec.ExistingConfigPath())
		}
		fmt.Fprint(stdout, snippetFor(spec, state, client, *agentKey))
		return 0
	}
	configPath := spec.ExistingConfigPath()
	if configPath == "" {
		fmt.Fprintf(stderr, "harbor connect: no config file found for %s (expected one of the registry paths)\n", spec.Key)
		return 2
	}
	if err := connectEditConfig(configPath, spec, state, client, *agentKey); err != nil {
		fmt.Fprintf(stderr, "harbor connect: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "wired %s (%s backed up as *.harbor.bak) - restart the client to take effect\n", spec.DisplayName, configPath)
	return 0
}

// connectEditConfig performs the minimal in-place edit on an mcp-json
// config: original entries are preserved under a backup, then each imported
// server name is replaced by its harbor endpoint with the agent key header.
func connectEditConfig(configPath string, spec clients.ClientSpec, state importState, client *cliClient, agentKey string) error {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return fmt.Errorf("parse %s: %w", configPath, err)
	}
	if document == nil {
		document = map[string]any{}
	}
	servers, ok := document["mcpServers"].(map[string]any)
	if !ok {
		servers = map[string]any{}
		document["mcpServers"] = servers
	}
	backup, err := os.Create(configPath + ".harbor.bak")
	if err != nil {
		return fmt.Errorf("backup: %w", err)
	}
	if _, err := backup.Write(raw); err != nil {
		backup.Close()
		return fmt.Errorf("backup: %w", err)
	}
	backup.Close()

	names := make([]string, 0, len(state.Targets))
	for name := range state.Targets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		servers[name] = map[string]any{
			"type": "http",
			"url":  client.baseURL + "/api/v1/mcp/agents/" + state.Targets[name] + "/rpc",
			"headers": map[string]any{
				"X-Agent-Key": agentKey,
			},
		}
	}
	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, encoded, 0o644)
}

// snippetFor renders a paste-ready snippet for print-only clients (the
// codex TOML shape in M1) or for --print on any client.
func snippetFor(spec clients.ClientSpec, state importState, client *cliClient, agentKey string) string {
	var builder strings.Builder
	if spec.Format == clients.FormatCodexTOML {
		builder.WriteString("# ~/.codex/config.toml\n")
		names := sortedKeys(state.Targets)
		for _, name := range names {
			fmt.Fprintf(&builder, "[mcp_servers.%s]\n", name)
			fmt.Fprintf(&builder, "url = %q\n", client.baseURL+"/api/v1/mcp/agents/"+state.Targets[name]+"/rpc")
			fmt.Fprintf(&builder, "http_headers = { \"X-Agent-Key\" = %q }\n\n", agentKey)
		}
		return builder.String()
	}
	entry := map[string]any{}
	names := sortedKeys(state.Targets)
	for _, name := range names {
		entry[name] = map[string]any{
			"type": "http",
			"url":  client.baseURL + "/api/v1/mcp/agents/" + state.Targets[name] + "/rpc",
			"headers": map[string]any{
				"X-Agent-Key": agentKey,
			},
		}
	}
	encoded, _ := json.MarshalIndent(entry, "", "  ")
	fmt.Fprintf(&builder, "// add to mcpServers:\n%s\n", encoded)
	return builder.String()
}

func sortedKeys(mapping map[string]string) []string {
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
