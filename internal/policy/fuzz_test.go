package policy

import (
	"bytes"
	"strings"
	"testing"
)

// FuzzParse guards the YAML walk: arbitrary input must error out cleanly,
// never panic, and anything that parses must be decidable.
func FuzzParse(f *testing.F) {
	f.Add([]byte(fixture))
	f.Add([]byte("version: 1\n"))
	f.Add([]byte("version: 1\nworkspaces:\n  - id: w\n    targets:\n      - id: t\nagents:\n  - id: a\n    workspace: w\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		policy, err := Parse(bytes.NewReader(data))
		if err != nil {
			return
		}
		// Whatever parses must answer requests without panicking.
		policy.Decide(Request{Agent: "a", Target: "t", Tool: "x", Action: ActionInvoke})
		policy.Decide(Request{Agent: "", Target: "", Tool: "", Action: ActionLoad, Subject: "*"})
	})
}

// FuzzDecide checks the semantic invariants over random structured policies:
// a matched deny always wins, targets outside the workspace are always
// denied, strict workspaces only allow on an explicit match, and load and
// invoke never drift.
func FuzzDecide(f *testing.F) {
	f.Add(uint8(0), uint8(0), uint8(0), uint8(0))
	f.Fuzz(func(t *testing.T, seed1, seed2, seed3, seed4 uint8) {
		strict := seed1%2 == 0
		agentID := pick([]string{"codex", "claude"}, seed2)
		target := pick([]string{"crm", "billing"}, seed3)
		tool := pick([]string{"lookup", "bulkDelete", "exportAll", "deep.nested"}, seed4)

		source := `version: 1
workspaces:
  - id: main
    strict: ` + boolString(strict) + `
    targets:
      - id: crm
agents:
  - id: codex
    workspace: main
    allow:
      - "crm.lookup*"
    deny:
      - capability: "crm.*Delete"
        reason: no bulk writes
  - id: claude
    workspace: main
    allow:
      - "crm.*"
`
		policy, err := Parse(strings.NewReader(source))
		if err != nil {
			t.Fatalf("generated policy must parse: %v", err)
		}
		capability := target + "." + tool
		request := Request{Agent: agentID, Target: target, Tool: tool, Action: ActionInvoke}
		decision := policy.Decide(request)

		agent := policy.agentByID[agentID]
		denyMatch := false
		for _, rule := range agent.Deny {
			if rule.glob.match(capability) {
				denyMatch = true
			}
		}
		allowMatch := false
		for _, rule := range agent.Allow {
			if rule.glob.match(capability) {
				allowMatch = true
			}
		}

		if denyMatch && decision.Allowed {
			t.Fatalf("deny always wins: %+v", decision)
		}
		if target != "crm" && decision.Allowed {
			t.Fatalf("cross-boundary target must be denied: %+v", decision)
		}
		if strict && decision.Allowed && !allowMatch {
			t.Fatalf("strict allow requires an explicit match: %+v", decision)
		}
		if strict && !decision.Allowed && allowMatch && !denyMatch {
			t.Fatalf("strict deny despite an allow match: %+v", decision)
		}
		if !strict && !decision.Allowed && !denyMatch && target == "crm" {
			t.Fatalf("open mode denied without a deny match: %+v", decision)
		}
		load := policy.Decide(Request{Agent: agentID, Target: target, Tool: tool, Action: ActionLoad})
		if load.Allowed != decision.Allowed {
			t.Fatalf("load/invoke drift: %+v vs %+v", load, decision)
		}
	})
}

func pick(values []string, seed uint8) string {
	return values[int(seed)%len(values)]
}
