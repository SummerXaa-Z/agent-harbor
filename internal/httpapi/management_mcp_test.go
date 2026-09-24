package httpapi

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestManagementMCPTextTruncatesAtUTF8Boundary(t *testing.T) {
	text := managementMCPText(map[string]string{
		"message": strings.Repeat("权限", 4000),
	})
	if !strings.HasSuffix(text, "... truncated") {
		t.Fatalf("expected truncated MCP text content")
	}
	if !utf8.ValidString(text) {
		t.Fatalf("expected valid UTF-8 MCP text content")
	}
}

func TestManagementMCPWriteConfirmationFallsBackToSafetyMetadata(t *testing.T) {
	if !managementMCPToolRequiresConfirmation("create_admin_identity") {
		t.Fatalf("known management MCP write tool should require confirmation")
	}
	if managementMCPToolRequiresConfirmation("list_admin_identities") {
		t.Fatalf("known management MCP read tool should not require confirmation")
	}

	if !managementMCPToolRequiresConfirmationForMetadata(
		writeManagementMCPToolSafety("conditional"),
		managementMCPToolExecution{Idempotency: "safe_repeat"},
	) {
		t.Fatalf("mutating safety metadata should require confirmation even when execution metadata is incomplete")
	}
	if !managementMCPToolRequiresConfirmationForMetadata(
		managementMCPToolSafety{
			OperationType: "write",
			ReadOnly:      false,
			MutatesState:  false,
			ApprovalMode:  "conditional",
		},
		managementMCPToolExecution{Idempotency: "safe_repeat"},
	) {
		t.Fatalf("write operation type should require confirmation even when mutatesState is incomplete")
	}
	if managementMCPToolRequiresConfirmationForMetadata(
		readManagementMCPToolSafety(),
		managementMCPToolExecution{Idempotency: "safe_repeat"},
	) {
		t.Fatalf("read-only metadata should not require confirmation")
	}
}

func TestExplainAccessDecisionToolSchemaRequiresSubject(t *testing.T) {
	required, ok := explainAccessDecisionSchema()["required"].([]string)
	if !ok {
		t.Fatalf("explain tool schema has no required list: %#v", explainAccessDecisionSchema()["required"])
	}
	if !slices.Contains(required, "subjectId") {
		t.Fatalf("explain tool schema should require subjectId: %#v", required)
	}
}
