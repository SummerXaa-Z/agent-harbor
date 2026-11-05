package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// reviewAggregate summarizes a window of tool-call traces for the weekly
// ritual (UX design section 4): what ran, what was high-risk, and which
// deny rules the usage data suggests.
type reviewAggregate struct {
	TotalCalls    int
	DeniedCalls   int
	CapabilityUse map[string]*capabilityUse
}

type capabilityUse struct {
	Calls     int
	Denied    int
	RiskFlags map[string]bool
	Callers   map[string]bool
}

func aggregateReview(rows []traceRow, names map[string]string) reviewAggregate {
	aggregate := reviewAggregate{CapabilityUse: map[string]*capabilityUse{}}
	for _, row := range rows {
		capability := capabilityKey(row, names)
		use, ok := aggregate.CapabilityUse[capability]
		if !ok {
			use = &capabilityUse{RiskFlags: map[string]bool{}, Callers: map[string]bool{}}
			aggregate.CapabilityUse[capability] = use
		}
		aggregate.TotalCalls++
		use.Calls++
		if row.CallerID != "" {
			use.Callers[row.CallerID] = true
		}
		if row.Decision == "denied" {
			aggregate.DeniedCalls++
			use.Denied++
		}
		for _, flag := range row.RiskFlags {
			use.RiskFlags[flag] = true
		}
	}
	return aggregate
}

// suggestions are high-risk capabilities that still ran (allowed) during the
// window - the deny candidates progressive hardening starts from.
func (a reviewAggregate) suggestions() []string {
	out := []string{}
	for capability, use := range a.CapabilityUse {
		if len(use.RiskFlags) == 0 || use.Calls-use.Denied <= 0 {
			continue
		}
		flags := sortedFlagList(use.RiskFlags)
		out = append(out, fmt.Sprintf("      - capability: %q   # %d call(s), %d denied, flags: %s",
			capability, use.Calls, use.Denied, strings.Join(flags, ",")))
	}
	sort.Strings(out)
	return out
}

func sortedFlagList(flags map[string]bool) []string {
	out := make([]string, 0, len(flags))
	for flag := range flags {
		out = append(out, flag)
	}
	sort.Strings(out)
	return out
}

// capabilityNames maps capability ids to readable "<target>.<tool>" names
// via the admin capabilities list; unmapped ids fall back to the raw id.
func capabilityNames(client *cliClient) map[string]string {
	envelope, err := client.getJSON("/api/v1/capabilities")
	if err != nil {
		return map[string]string{}
	}
	var rows []struct {
		ID       string `json:"id"`
		Key      string `json:"key"`
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(envelope.Data, &rows); err != nil {
		return map[string]string{}
	}
	names := map[string]string{}
	for _, row := range rows {
		names[row.ID] = row.TargetID + "." + row.Key
	}
	return names
}

func capabilityKey(row traceRow, names map[string]string) string {
	if row.CapabilityID == "" {
		return "unknown"
	}
	if name, ok := names[row.CapabilityID]; ok {
		return name
	}
	// Unmapped (stale capability rows): keep the target-qualified shape so
	// aggregated keys stay consistent.
	return row.TargetID + "." + row.CapabilityID
}

func cmdReview(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("review", stderr)
	clientFlags := addClientFlags(flags)
	days := flags.Int("days", 7, "window length in days")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	client := clientFromFlags(clientFlags)
	query := url.Values{}
	query.Set("limit", "500")
	query.Set("since", time.Now().AddDate(0, 0, -*days).UTC().Format(time.RFC3339))
	envelope, err := client.getJSON("/api/v1/audit/traces?" + query.Encode())
	if err != nil {
		fmt.Fprintf(stderr, "harbor review: %v\n", err)
		return 4
	}
	if envelope.Error != "" {
		fmt.Fprintf(stderr, "harbor review: %s: %s\n", envelope.Error, envelope.Message)
		if envelope.Code == http.StatusForbidden {
			return 3
		}
		return 4
	}
	var rows []traceRow
	if err := json.Unmarshal(envelope.Data, &rows); err != nil {
		fmt.Fprintf(stderr, "harbor review: decode traces: %v\n", err)
		return 4
	}
	aggregate := aggregateReview(rows, capabilityNames(client))

	fmt.Fprintf(stdout, "last %d day(s): %d call(s), %d denied\n\n", *days, aggregate.TotalCalls, aggregate.DeniedCalls)
	names := make([]string, 0, len(aggregate.CapabilityUse))
	for name := range aggregate.CapabilityUse {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return aggregate.CapabilityUse[names[i]].Calls > aggregate.CapabilityUse[names[j]].Calls
	})
	for _, name := range names {
		use := aggregate.CapabilityUse[name]
		flag := ""
		if len(use.RiskFlags) > 0 {
			flag = "  [" + strings.Join(sortedFlagList(use.RiskFlags), ",") + "]"
		}
		fmt.Fprintf(stdout, "  %-40s %4d call(s), %d denied%s\n", name, use.Calls, use.Denied, flag)
	}
	if suggestions := aggregate.suggestions(); len(suggestions) > 0 {
		fmt.Fprintf(stdout, "\nsuggested deny candidates (append to harbor.yaml, then harbor apply):\n    deny:\n")
		for _, suggestion := range suggestions {
			fmt.Fprintln(stdout, suggestion)
		}
	} else {
		fmt.Fprintln(stdout, "\nno high-risk capabilities ran unblocked this window - nothing to tighten")
	}
	return 0
}
