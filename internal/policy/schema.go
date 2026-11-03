package policy

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Policy is the compiled form of harbor.yaml. It is pure data with compiled
// globs; this package performs no IO and must stay independent of store and
// httpapi (it is the seed of the M3 harbor-policy module).
type Policy struct {
	Version    int
	Workspaces []Workspace
	Agents     []Agent

	workspaceByID map[string]*Workspace
	agentByID     map[string]*Agent
}

type Workspace struct {
	ID      string
	Strict  bool
	Targets []Target
	Domains []string
	Line    int
}

type Target struct {
	ID       string
	Endpoint string
	Line     int
}

type Agent struct {
	ID        string
	Workspace string
	Allow     []Rule
	Deny      []DenyRule
	Subjects  []string
	Line      int
}

// Rule is one allow entry; Capability is a glob over "<target>.<tool>".
type Rule struct {
	Capability string
	glob       glob
	Line       int
}

// DenyRule adds an operator-facing reason to a deny entry.
type DenyRule struct {
	Rule
	Reason string
}

// Parse reads harbor.yaml strictly: unknown fields are rejected with their
// line number (semantics rule 6), so future syntax can never fail silently.
func Parse(reader io.Reader) (*Policy, error) {
	var document yaml.Node
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("harbor.yaml: %w", err)
	}
	root := unwrapDocument(&document)
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("harbor.yaml:%d: expected a mapping at the top level", root.Line)
	}
	policy := &Policy{workspaceByID: map[string]*Workspace{}, agentByID: map[string]*Agent{}}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, value := root.Content[i], root.Content[i+1]
		switch key.Value {
		case "version":
			version, err := decodeInt(key, value)
			if err != nil {
				return nil, err
			}
			if version != 1 {
				return nil, fmt.Errorf("harbor.yaml:%d: unsupported version %d (this build understands 1)", value.Line, version)
			}
			policy.Version = version
		case "workspaces":
			if err := policy.decodeWorkspaces(key, value); err != nil {
				return nil, err
			}
		case "agents":
			if err := policy.decodeAgents(key, value); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("harbor.yaml:%d: unknown field %q", key.Line, key.Value)
		}
	}
	if policy.Version == 0 {
		return nil, fmt.Errorf("harbor.yaml: missing required field \"version\"")
	}
	return policy, nil
}

func (p *Policy) decodeWorkspaces(key, value *yaml.Node) error {
	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("harbor.yaml:%d: %s must be a list", value.Line, key.Value)
	}
	seen := map[string]bool{}
	for _, entry := range value.Content {
		workspace := Workspace{Line: entry.Line}
		if err := strictFields(entry, func(field string, node *yaml.Node) error {
			switch field {
			case "id":
				return decodeString(&workspace.ID, node)
			case "strict":
				return decodeBool(&workspace.Strict, node)
			case "domains":
				return decodeStringList(&workspace.Domains, node)
			case "targets":
				return p.decodeTargets(&workspace, node)
			}
			return nil
		}, "id", "strict", "targets", "domains"); err != nil {
			return err
		}
		if workspace.ID == "" {
			return fmt.Errorf("harbor.yaml:%d: workspace entry is missing \"id\"", entry.Line)
		}
		if seen[workspace.ID] {
			return fmt.Errorf("harbor.yaml:%d: duplicate workspace %q", entry.Line, workspace.ID)
		}
		seen[workspace.ID] = true
		p.Workspaces = append(p.Workspaces, workspace)
	}
	// Pointers must target the slice elements, so index after the append loop.
	for i := range p.Workspaces {
		p.workspaceByID[p.Workspaces[i].ID] = &p.Workspaces[i]
	}
	return nil
}

func (p *Policy) decodeTargets(workspace *Workspace, value *yaml.Node) error {
	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("harbor.yaml:%d: targets must be a list", value.Line)
	}
	for _, entry := range value.Content {
		target := Target{Line: entry.Line}
		if err := strictFields(entry, func(field string, node *yaml.Node) error {
			switch field {
			case "id":
				return decodeString(&target.ID, node)
			case "endpoint":
				return decodeString(&target.Endpoint, node)
			}
			return nil
		}, "id", "endpoint"); err != nil {
			return err
		}
		if target.ID == "" {
			return fmt.Errorf("harbor.yaml:%d: target entry is missing \"id\"", entry.Line)
		}
		workspace.Targets = append(workspace.Targets, target)
	}
	return nil
}

func (p *Policy) decodeAgents(key, value *yaml.Node) error {
	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("harbor.yaml:%d: %s must be a list", value.Line, key.Value)
	}
	seen := map[string]bool{}
	for _, entry := range value.Content {
		agent := Agent{Line: entry.Line}
		if err := strictFields(entry, func(field string, node *yaml.Node) error {
			switch field {
			case "id":
				return decodeString(&agent.ID, node)
			case "workspace":
				return decodeString(&agent.Workspace, node)
			case "subjects":
				return decodeStringList(&agent.Subjects, node)
			case "allow":
				return decodeAllow(&agent, node)
			case "deny":
				return decodeDeny(&agent, node)
			}
			return nil
		}, "id", "workspace", "allow", "deny", "subjects"); err != nil {
			return err
		}
		if agent.ID == "" {
			return fmt.Errorf("harbor.yaml:%d: agent entry is missing \"id\"", entry.Line)
		}
		if agent.Workspace == "" {
			return fmt.Errorf("harbor.yaml:%d: agent %q is missing \"workspace\"", entry.Line, agent.ID)
		}
		if seen[agent.ID] {
			return fmt.Errorf("harbor.yaml:%d: duplicate agent %q", entry.Line, agent.ID)
		}
		seen[agent.ID] = true
		p.Agents = append(p.Agents, agent)
	}
	for i := range p.Agents {
		p.agentByID[p.Agents[i].ID] = &p.Agents[i]
	}
	for _, agent := range p.Agents {
		if _, ok := p.workspaceByID[agent.Workspace]; !ok {
			return fmt.Errorf("harbor.yaml:%d: agent %q references unknown workspace %q", agent.Line, agent.ID, agent.Workspace)
		}
	}
	return nil
}

func decodeAllow(agent *Agent, value *yaml.Node) error {
	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("harbor.yaml:%d: allow must be a list", value.Line)
	}
	for _, entry := range value.Content {
		compiled, err := compileGlob(entry.Value)
		if err != nil {
			return fmt.Errorf("harbor.yaml:%d: %w", entry.Line, err)
		}
		agent.Allow = append(agent.Allow, Rule{Capability: entry.Value, glob: compiled, Line: entry.Line})
	}
	return nil
}

func decodeDeny(agent *Agent, value *yaml.Node) error {
	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("harbor.yaml:%d: deny must be a list", value.Line)
	}
	for _, entry := range value.Content {
		deny := DenyRule{Reason: "denied by policy"}
		if err := strictFields(entry, func(field string, node *yaml.Node) error {
			switch field {
			case "capability":
				compiled, err := compileGlob(node.Value)
				if err != nil {
					return fmt.Errorf("harbor.yaml:%d: %w", node.Line, err)
				}
				deny.Capability = node.Value
				deny.glob = compiled
				deny.Line = node.Line
				return nil
			case "reason":
				return decodeString(&deny.Reason, node)
			}
			return nil
		}, "capability", "reason"); err != nil {
			return err
		}
		if deny.Capability == "" {
			return fmt.Errorf("harbor.yaml:%d: deny entry is missing \"capability\"", entry.Line)
		}
		agent.Deny = append(agent.Deny, deny)
	}
	return nil
}

// strictFields walks a mapping node, rejects unknown keys with line numbers,
// and hands known keys to the callback. Null values pass through untouched.
func strictFields(node *yaml.Node, visit func(field string, value *yaml.Node) error, allowed ...string) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("harbor.yaml:%d: expected a mapping", node.Line)
	}
	known := map[string]bool{}
	for _, name := range allowed {
		known[name] = true
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if !known[key.Value] {
			return fmt.Errorf("harbor.yaml:%d: unknown field %q (known: %s)", key.Line, key.Value, joinQuoted(allowed))
		}
		if value.Tag == "!!null" {
			continue
		}
		if err := visit(key.Value, value); err != nil {
			return err
		}
	}
	return nil
}

func unwrapDocument(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func decodeString(target *string, node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("harbor.yaml:%d: expected a string", node.Line)
	}
	*target = node.Value
	return nil
}

func decodeBool(target *bool, node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode || (node.Value != "true" && node.Value != "false") {
		return fmt.Errorf("harbor.yaml:%d: expected true or false", node.Line)
	}
	*target = node.Value == "true"
	return nil
}

func decodeInt(key *yaml.Node, node *yaml.Node) (int, error) {
	if node.Kind != yaml.ScalarNode {
		return 0, fmt.Errorf("harbor.yaml:%d: expected an integer", node.Line)
	}
	value, err := strconv.Atoi(node.Value)
	if err != nil {
		return 0, fmt.Errorf("harbor.yaml:%d: %q is not an integer", node.Line, node.Value)
	}
	_ = key
	return value, nil
}

func decodeStringList(target *[]string, node *yaml.Node) error {
	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("harbor.yaml:%d: expected a list", node.Line)
	}
	for _, entry := range node.Content {
		*target = append(*target, entry.Value)
	}
	return nil
}

func joinQuoted(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = strconv.Quote(value)
	}
	return strings.Join(quoted, ", ")
}

// LoadFile reads and parses a harbor.yaml from disk; boot fails loudly on a
// bad policy file rather than running on assumptions.
func LoadFile(path string) (*Policy, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open policy file: %w", err)
	}
	defer file.Close()
	return Parse(file)
}
