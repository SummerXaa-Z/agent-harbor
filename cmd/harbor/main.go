// Command harbor is the v2 operator CLI (UX design section 2). Exit codes:
// 0 success, 2 usage/config error, 3 policy denial, 4 network failure.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/SummerXaa-Z/agent-harbor/internal/app"
)

const usageText = `harbor - AgentHarbor operator CLI

Usage:
  harbor serve [--addr :9090]                       run the gateway (env as for agent-harbor)
  harbor init [--file harbor.yaml] [--force]        write a starter policy file
  harbor apply [--file harbor.yaml] [--dry-run]     validate the policy file and summarize it
  harbor audit [--high-risk] [--follow] [--json] [--limit N]
  harbor why <traceId>
  harbor key issue <agentId> [--name N] [--ttl-seconds S]
  harbor key revoke <keyId>
  harbor key list [<agentId>]
  harbor key rotate <agentId> --old <keyId>

Client flags (audit/why/key): --url (HARBOR_URL, default http://127.0.0.1:9090),
--admin-key (HARBOR_ADMIN_KEY) sent as X-Admin-Key.

Exit codes: 0 success, 2 usage/config error, 3 policy denial, 4 network failure.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return 2
	}
	command, rest := args[0], args[1:]
	var code int
	switch command {
	case "serve":
		code = cmdServe(rest, stdout, stderr)
	case "init":
		code = cmdInit(rest, stdout, stderr)
	case "apply":
		code = cmdApply(rest, stdout, stderr)
	case "audit":
		code = cmdAudit(rest, stdout, stderr)
	case "why":
		code = cmdWhy(rest, stdout, stderr)
	case "key":
		code = cmdKey(rest, stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usageText)
		return 0
	default:
		fmt.Fprintf(stderr, "harbor: unknown command %q\n\n%s", command, usageText)
		return 2
	}
	return code
}

func cmdServe(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("serve", stderr)
	addr := flags.String("addr", envOr("AGENT_HARBOR_ADDR", ":9090"), "listen address")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if err := app.RunServer(context.Background(), *addr); err != nil {
		fmt.Fprintf(stderr, "harbor serve: %v\n", err)
		return 2
	}
	return 0
}
