package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SummerXaa-Z/agent-harbor/internal/wrap"
)

func parseFlags(args []string, stderr io.Writer) (*wrap.Options, []string) {
	flags := flag.NewFlagSet("harbor-wrap", flag.ContinueOnError)
	flags.SetOutput(stderr)
	home, _ := os.UserHomeDir()
	agent := flags.String("agent", "", "agent id in harbor.yaml (required)")
	target := flags.String("target", "", "target id in harbor.yaml (required)")
	policyPath := flags.String("policy", filepath.Join(home, ".harbor", "harbor.yaml"), "harbor.yaml path (mtime-watched)")
	spoolDir := flags.String("spool", filepath.Join(home, ".harbor", "spool"), "audit spool directory")
	if err := flags.Parse(args); err != nil {
		return nil, nil
	}
	command := flags.Args()
	if *agent == "" || *target == "" || len(command) == 0 {
		fmt.Fprintln(stderr, "harbor-wrap: --agent and --target and the server command are required")
		fmt.Fprintln(stderr, "usage: harbor-wrap --agent <id> --target <id> [--policy path] [--spool dir] -- <command> [args...]")
		return nil, nil
	}
	return &wrap.Options{
		AgentID:    *agent,
		TargetID:   *target,
		PolicyPath: *policyPath,
		SpoolDir:   *spoolDir,
	}, command
}
