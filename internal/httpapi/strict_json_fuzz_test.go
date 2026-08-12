package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type strictJSONFuzzPayload struct {
	Name  string            `json:"name"`
	Count int               `json:"count"`
	Meta  map[string]string `json:"meta,omitempty"`
}

func TestDecodeJSONRejectsAmbiguousOrInvalidPayloads(t *testing.T) {
	tests := map[string]string{
		"duplicate top-level field": `{"name":"first","name":"second","count":1}`,
		"duplicate nested field":    `{"name":"agent","count":1,"meta":{"scope":"first","scope":"second"}}`,
		"unknown field":             `{"name":"agent","count":1,"unexpected":true}`,
		"trailing value":            `{"name":"agent","count":1} {}`,
		"wrong type":                `{"name":"agent","count":"one"}`,
	}
	for name, payload := range tests {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(payload))
			request.Header.Set("Content-Type", "application/json")
			var decoded strictJSONFuzzPayload
			if err := decodeJSON(request, &decoded); err == nil {
				t.Fatalf("expected payload to be rejected: %s", payload)
			}
		})
	}
}

func TestDecodeJSONAcceptsSingleUnambiguousPayload(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name":"agent","count":1,"meta":{"scope":"tenant"}}`))
	request.Header.Set("Content-Type", "application/json")
	var decoded strictJSONFuzzPayload
	if err := decodeJSON(request, &decoded); err != nil {
		t.Fatalf("expected valid payload to be accepted: %v", err)
	}
	if decoded.Name != "agent" || decoded.Count != 1 || decoded.Meta["scope"] != "tenant" {
		t.Fatalf("unexpected decoded payload: %#v", decoded)
	}
}

func TestDecodeManagementMCPArgumentsRejectsAmbiguousOrInvalidPayloads(t *testing.T) {
	tests := map[string]string{
		"duplicate field": `{"tenantId":"first","tenantId":"second","workspaceId":"ws"}`,
		"unknown field":   `{"tenantId":"tenant","workspaceId":"ws","unexpected":true}`,
		"trailing value":  `{"tenantId":"tenant","workspaceId":"ws"} {}`,
		"wrong type":      `{"tenantId":1,"workspaceId":"ws"}`,
	}
	for name, payload := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeManagementMCPArguments[managementMCPScopeArgs](json.RawMessage(payload)); err == nil {
				t.Fatalf("expected MCP arguments to be rejected: %s", payload)
			}
		})
	}
}

func TestManagementMCPWriteConfirmationRejectsDuplicateFields(t *testing.T) {
	request := managementMCPRequest{}
	request.Params.Name = "create_admin_identity"
	request.Params.Arguments = json.RawMessage(`{
		"actor":"mcp-admin",
		"role":"tenant_admin",
		"tenantId":"tenant",
		"confirmation":{"confirmed":false,"confirmed":true,"reason":"ambiguous approval"}
	}`)
	if _, _, err := managementMCPWriteConfirmationForRequest(request); err == nil {
		t.Fatal("expected duplicate confirmation field to fail closed")
	}
}

func TestManagementMCPRequestRejectsDuplicateNestedFields(t *testing.T) {
	payload := `{
		"jsonrpc":"2.0",
		"id":"duplicate-arguments",
		"method":"tools/call",
		"params":{
			"name":"get_tenant_access_profile",
			"arguments":{"tenantId":"first","tenantId":"second","workspaceId":"ws"}
		}
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/management/mcp", bytes.NewBufferString(payload))
	request.Header.Set("Content-Type", "application/json")
	if _, err := managementMCPRequestFromHTTP(request); err == nil {
		t.Fatal("expected management MCP request with duplicate nested fields to fail closed")
	}
}

func FuzzDecodeJSON(f *testing.F) {
	for _, seed := range []string{
		`{"name":"agent","count":1}`,
		`{"name":"first","name":"second","count":1}`,
		`{"name":"agent","count":1,"meta":{"scope":"a","scope":"b"}}`,
		`{"name":"agent","count":1,"unexpected":true}`,
		`{"name":"agent","count":1} {}`,
		`{"name":"agent","count":"one"}`,
		``,
	} {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, payload []byte) {
		request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
		request.Header.Set("Content-Type", "application/json")
		var got strictJSONFuzzPayload
		if err := decodeJSON(request, &got); err != nil {
			return
		}

		var want strictJSONFuzzPayload
		if err := json.Unmarshal(payload, &want); err != nil {
			t.Fatalf("decodeJSON accepted payload rejected by encoding/json: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("accepted payload decoded inconsistently: got=%#v want=%#v", got, want)
		}
	})
}

func FuzzDecodeManagementMCPArguments(f *testing.F) {
	for _, seed := range []string{
		`{"tenantId":"tenant","workspaceId":"ws"}`,
		`{"tenantId":"first","tenantId":"second","workspaceId":"ws"}`,
		`{"tenantId":"tenant","workspaceId":"ws","unexpected":true}`,
		`{"tenantId":"tenant","workspaceId":"ws"} {}`,
		`{"tenantId":1,"workspaceId":"ws"}`,
		`null`,
		``,
	} {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, payload []byte) {
		got, err := decodeManagementMCPArguments[managementMCPScopeArgs](json.RawMessage(payload))
		if err != nil {
			return
		}

		normalized := payload
		if len(normalized) == 0 || bytes.Equal(normalized, []byte("null")) {
			normalized = []byte("{}")
		}
		var want managementMCPScopeArgs
		if err := json.Unmarshal(normalized, &want); err != nil {
			t.Fatalf("MCP decoder accepted payload rejected by encoding/json: %v", err)
		}
		if got != want {
			t.Fatalf("accepted MCP arguments decoded inconsistently: got=%#v want=%#v", got, want)
		}
	})
}
