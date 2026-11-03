package policy

import (
	"fmt"
	"regexp"
	"strings"
)

// A capability key is "<target>.<tool>". Globs use a single metacharacter:
// `*` matches any run of characters including dots, so "crm.*" covers every
// tool (including nested names) of target crm - the documented whole-server
// idiom.
type glob struct {
	pattern *regexp.Regexp
	source  string
}

// compileGlob builds the matcher for a capability pattern. Matching is
// case-insensitive: patterns are written by humans ("*delete*"), while
// capability casing belongs to machines ("bulkDelete") - the trap of a deny
// rule silently missing "Delete" is worse than a rare false positive.
func compileGlob(pattern string) (glob, error) {
	if strings.TrimSpace(pattern) == "" {
		return glob{}, fmt.Errorf("empty capability pattern")
	}
	chunks := strings.Split(pattern, "*")
	var builder strings.Builder
	builder.WriteString("^(?i)")
	builder.WriteString(regexp.QuoteMeta(chunks[0]))
	for _, chunk := range chunks[1:] {
		builder.WriteString(".*")
		builder.WriteString(regexp.QuoteMeta(chunk))
	}
	builder.WriteString("$")
	compiled, err := regexp.Compile(builder.String())
	if err != nil {
		return glob{}, fmt.Errorf("capability pattern %q: %w", pattern, err)
	}
	return glob{pattern: compiled, source: pattern}, nil
}

func (g glob) match(capability string) bool {
	return g.pattern.MatchString(capability)
}
