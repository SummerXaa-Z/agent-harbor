package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/SummerXaa-Z/agent-harbor/internal/policy"
)

// starterPolicy is what `harbor init` writes: an open workspace with an
// empty circle and no agents - import (PR-E) fills it; the comments carry
// the whole grammar.
const starterPolicy = `# harbor.yaml - AgentHarbor policy, the single source of truth.
# The server loads it when AGENT_HARBOR_POLICY_FILE points here.
version: 1

workspaces:
  - id: main
    strict: false          # false: inside the circle everything is allowed (high-risk calls are flagged red in audit); true: default deny, allow rules apply
    targets: []            # targets inside the circle, e.g.:
                           #   - id: crm
                           #     endpoint: https://crm.internal/mcp
    domains: []            # data domains declared for this circle

agents: []                 # callers and their rules, e.g.:
                           #   - id: codex
                           #     workspace: main
                           #     allow: ["crm.*"]            # whole-server grants are the default idiom
                           #     deny:
                           #       - capability: "crm.*delete*"
                           #         reason: prod database is read-only
                           #     subjects: ["user:support-*"] # optional: subjects it may act for
`

func cmdInit(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("init", stderr)
	file := flags.String("file", "harbor.yaml", "policy file to create")
	force := flags.Bool("force", false, "overwrite an existing file")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if _, err := os.Stat(*file); err == nil && !*force {
		fmt.Fprintf(stderr, "harbor init: %s already exists (use --force)\n", *file)
		return 2
	}
	if err := os.WriteFile(*file, []byte(starterPolicy), 0o644); err != nil {
		fmt.Fprintf(stderr, "harbor init: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "wrote %s - add targets and agents, then start the server with AGENT_HARBOR_POLICY_FILE=%s\n", *file, *file)
	return 0
}

func cmdApply(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("apply", stderr)
	file := flags.String("file", envOr("AGENT_HARBOR_POLICY_FILE", "harbor.yaml"), "policy file to validate")
	dryRun := flags.Bool("dry-run", false, "validate and summarize without side effects")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	parsed, err := policy.LoadFile(*file)
	if err != nil {
		fmt.Fprintf(stderr, "harbor apply: %v\n", err)
		return 2
	}
	// The file itself is the materialized state (the server loads it at
	// boot), so apply's M1 job is validation plus a summary; --dry-run is
	// accepted for script compatibility and behaves identically.
	if *dryRun {
		fmt.Fprintf(stdout, "validated %s (dry run)\n", *file)
	}
	strict := 0
	for _, workspace := range parsed.Workspaces {
		strictWork := ""
		if workspace.Strict {
			strictWork = " (strict)"
			strict++
		}
		fmt.Fprintf(stdout, "workspace %-20s %d target(s)%s\n", workspace.ID, len(workspace.Targets), strictWork)
	}
	for _, agent := range parsed.Agents {
		fmt.Fprintf(stdout, "agent     %-20s %d allow, %d deny, %d subject selector(s)\n",
			agent.ID, len(agent.Allow), len(agent.Deny), len(agent.Subjects))
	}
	if len(parsed.Workspaces) == 0 {
		fmt.Fprintln(stdout, "no workspaces declared - every call will be boundary-denied")
	}
	if strict == 0 {
		fmt.Fprintln(stdout, "policy valid; all workspaces are open mode (default allow, high-risk flagged)")
	} else {
		fmt.Fprintf(stdout, "policy valid; %d workspace(s) in strict mode\n", strict)
	}
	return 0
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	return flags
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
