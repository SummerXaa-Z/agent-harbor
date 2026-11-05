package policy

import (
	"fmt"
	"sort"
	"strings"
)

// Action distinguishes a visibility check from an enforcement check. Both
// run the same evaluation path - the single-filter constraint from the
// component design: tools/list filtering and tools/call decisions must never
// drift apart.
type Action string

const (
	ActionLoad   Action = "load"   // tools/list visibility
	ActionInvoke Action = "invoke" // tools/call enforcement
)

type Request struct {
	Agent   string
	Target  string
	Tool    string
	Action  Action
	Subject string
}

// Step is one link of the evidence chain; the chain explains a decision in
// the order the checks ran.
type Step struct {
	Check  string `json:"check"`
	Result string `json:"result"`
	Detail string `json:"detail,omitempty"`
}

type Decision struct {
	Allowed  bool
	Reason   string
	RuleRef  string
	Evidence []Step
}

// Decide implements the seven semantics rules of the component design:
// glob matching with the whole-server idiom, deny always winning, open vs
// strict defaults, cross-workspace always explicit, and visibility sharing
// this exact path. Unknown agents and unknown actions fail closed.
func (p *Policy) Decide(request Request) Decision {
	capability := request.Target + "." + request.Tool
	decision := Decision{Evidence: []Step{}}

	agent, known := p.agentByID[request.Agent]
	if !known {
		return denyf(decision, fmt.Sprintf("agent %q is not declared in harbor.yaml", request.Agent),
			Step{Check: "agent", Result: "unknown", Detail: request.Agent}, "")
	}
	workspace := p.workspaceByID[agent.Workspace]

	if request.Action != ActionLoad && request.Action != ActionInvoke {
		return denyf(decision, fmt.Sprintf("unknown action %q", request.Action),
			Step{Check: "action", Result: "unknown", Detail: string(request.Action)}, "")
	}

	if !workspaceHasTarget(workspace, request.Target) {
		return denyf(decision, fmt.Sprintf("target %q is outside workspace %q", request.Target, workspace.ID),
			Step{Check: "boundary", Result: "deny", Detail: fmt.Sprintf("target %s not in workspace %s", request.Target, workspace.ID)}, "")
	}
	decision.Evidence = append(decision.Evidence, Step{Check: "boundary", Result: "pass", Detail: fmt.Sprintf("target %s in workspace %s", request.Target, workspace.ID)})

	if match, ok := matchDeny(agent, capability); ok {
		return denyf(decision, match.Reason, Step{Check: "deny", Result: "deny", Detail: match.Capability}, ruleRef(match.Line))
	}
	decision.Evidence = append(decision.Evidence, Step{Check: "deny", Result: "pass", Detail: "no deny rule matched"})

	if request.Subject != "" && len(agent.Subjects) > 0 && !matchAny(agent.Subjects, request.Subject) {
		return denyf(decision, fmt.Sprintf("agent %q may not act for subject %q", agent.ID, request.Subject),
			Step{Check: "subject", Result: "deny", Detail: fmt.Sprintf("subject %s matches no selector", request.Subject)}, "")
	}

	if match, ok := matchAllow(agent, capability); ok {
		decision.Allowed = true
		decision.Reason = "allowed by " + match.Capability
		decision.RuleRef = ruleRef(match.Line)
		decision.Evidence = append(decision.Evidence, Step{Check: "allow", Result: "allow", Detail: match.Capability})
		return decision
	}

	if workspace.Strict {
		return denyf(decision, fmt.Sprintf("workspace %q is strict and no allow rule matched %q", workspace.ID, capability),
			Step{Check: "mode", Result: "deny", Detail: "strict default"}, ruleRef(workspace.Line))
	}
	decision.Allowed = true
	decision.Reason = "default-open"
	decision.Evidence = append(decision.Evidence, Step{Check: "mode", Result: "allow", Detail: "default-open"})
	return decision
}

// WorkspaceOf returns the workspace an agent belongs to (nil when undeclared).
func (p *Policy) WorkspaceOf(agentID string) *Workspace {
	agent, ok := p.agentByID[agentID]
	if !ok {
		return nil
	}
	return p.workspaceByID[agent.Workspace]
}

// AgentIDs lists declared agents sorted by id.
func (p *Policy) AgentIDs() []string {
	ids := make([]string, 0, len(p.Agents))
	for _, agent := range p.Agents {
		ids = append(ids, agent.ID)
	}
	sort.Strings(ids)
	return ids
}

// nameRiskPatterns drive the audit red flags for capability names; the
// action-based half of the heuristic lives server-side with the capability
// metadata (component design section 3).
var nameRiskPatterns = []string{
	"delete", "drop", "remove", "truncate", "purge",
	"export", "secret", "credential", "admin",
}

// NameRiskFlags returns the name-pattern risk flags for a capability name,
// sorted; empty means no name-based flag.
func NameRiskFlags(name string) []string {
	lower := strings.ToLower(name)
	flags := []string{}
	for _, pattern := range nameRiskPatterns {
		if strings.Contains(lower, pattern) {
			flags = append(flags, pattern)
		}
	}
	sort.Strings(flags)
	return flags
}

func denyf(decision Decision, reason string, step Step, ref string) Decision {
	decision.Allowed = false
	decision.Reason = reason
	decision.RuleRef = ref
	decision.Evidence = append(decision.Evidence, step)
	return decision
}

func workspaceHasTarget(workspace *Workspace, targetID string) bool {
	for _, target := range workspace.Targets {
		if target.ID == targetID {
			return true
		}
	}
	return false
}

func matchDeny(agent *Agent, capability string) (DenyRule, bool) {
	for _, rule := range agent.Deny {
		if rule.glob.match(capability) {
			return rule, true
		}
	}
	return DenyRule{}, false
}

func matchAllow(agent *Agent, capability string) (Rule, bool) {
	for _, rule := range agent.Allow {
		if rule.glob.match(capability) {
			return rule, true
		}
	}
	return Rule{}, false
}

// matchAny compiles selectors on the fly; subject lists are tiny and this
// path runs once per call, not per pattern.
func matchAny(selectors []string, subject string) bool {
	for _, selector := range selectors {
		compiled, err := compileGlob(selector)
		if err != nil {
			continue
		}
		if compiled.match(subject) {
			return true
		}
	}
	return false
}

func ruleRef(line int) string {
	return fmt.Sprintf("harbor.yaml:%d", line)
}
