package httpapi

import (
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
