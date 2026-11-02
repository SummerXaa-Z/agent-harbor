package httpapi_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestAgentRegistryValidation(t *testing.T) {
	router := newRouter()

	created := createAgent(t, router, map[string]any{
		"name":        "Local Test Agent",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	})
	if created.ID == "" || created.WorkspaceID != "ws-1" {
		t.Fatalf("unexpected created agent: %#v", created)
	}

	list := decodeData[[]agentResponse](t, request(t, router, http.MethodGet, "/api/v1/agents?workspaceId=ws-1", nil, ""))
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("unexpected list response: %#v", list)
	}

	read := decodeData[agentResponse](t, request(t, router, http.MethodGet, "/api/v1/agents/"+created.ID, nil, ""))
	if read.Name != "Local Test Agent" {
		t.Fatalf("unexpected get response: %#v", read)
	}

	secretResp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Agent",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"apiKey":   "do-not-store-here",
		},
	}, "")
	if secretResp.Code != http.StatusBadRequest {
		t.Fatalf("secret-like channelConfig should fail, got %d", secretResp.Code)
	}

	nestedSecretResp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Nested Bad Agent",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"metadata": map[string]any{
				"credentialHeaders": map[string]any{
					"Authorization": "apiToken",
				},
			},
		},
	}, "")
	if nestedSecretResp.Code != http.StatusBadRequest {
		t.Fatalf("nested credentialHeaders should not bypass secret-like channelConfig validation, got %d", nestedSecretResp.Code)
	}

	unsafeResp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Unsafe Agent",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "http://127.0.0.1:8080/mcp",
		},
	}, "")
	if unsafeResp.Code != http.StatusBadRequest {
		t.Fatalf("unsafe endpoint should fail, got %d", unsafeResp.Code)
	}

	missingEndpoint := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Missing Endpoint MCP",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
	}, "")
	if missingEndpoint.Code != http.StatusBadRequest {
		t.Fatalf("active MCP without endpoint should fail, got %d", missingEndpoint.Code)
	}

	badEndpointType := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Endpoint Type",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": 123,
		},
	}, "")
	if badEndpointType.Code != http.StatusBadRequest {
		t.Fatalf("non-string endpoint should fail, got %d", badEndpointType.Code)
	}
}

func TestCreateAgentAllowsPrivateEndpointWhenExplicit(t *testing.T) {
	router := newRouterWithPrivateUpstreams()

	resp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Local MCP Target",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "http://127.0.0.1:8080/mcp",
		},
	}, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("private endpoint should be allowed when explicit option is enabled, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestManagementMutationsRejectDuplicateAgentAndKey(t *testing.T) {
	router := newRouter()
	first := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"tenantId":    "tenant-dup",
		"name":        "Support Agent",
		"workspaceId": "ws-dup",
		"channelType": "local",
		"status":      "active",
	}, "")
	if first.Code != http.StatusCreated {
		t.Fatalf("first agent create failed: %d body=%s", first.Code, first.Body.String())
	}
	created := decodeData[agentResponse](t, first)

	duplicate := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"tenantId":    "tenant-dup",
		"name":        " support agent ",
		"workspaceId": "ws-dup",
		"channelType": "local",
		"status":      "active",
	}, "")
	if duplicate.Code != http.StatusConflict || !strings.Contains(duplicate.Body.String(), "DUPLICATE_RESOURCE_MUTATION") {
		t.Fatalf("duplicate agent should be a conflict, got %d body=%s", duplicate.Code, duplicate.Body.String())
	}

	firstKey := request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId":          created.ID,
		"name":             "console key",
		"expiresInSeconds": 900,
	}, "")
	if firstKey.Code != http.StatusCreated {
		t.Fatalf("first key create failed: %d body=%s", firstKey.Code, firstKey.Body.String())
	}
	duplicateKey := request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId":          created.ID,
		"name":             "console key",
		"expiresInSeconds": 900,
	}, "")
	if duplicateKey.Code != http.StatusConflict || !strings.Contains(duplicateKey.Body.String(), "DUPLICATE_RESOURCE_MUTATION") {
		t.Fatalf("duplicate key should be a conflict, got %d body=%s", duplicateKey.Code, duplicateKey.Body.String())
	}
}

func TestManagementCredentialRotationDuplicateIsNoop(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	agent := createAgent(t, router, map[string]any{
		"tenantId":    "tenant-rotate",
		"name":        "Credential Target",
		"workspaceId": "ws-rotate",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
		"credentials": map[string]any{
			"apiToken": "Bearer one",
		},
	})
	if agent.CredentialVersion != 1 {
		t.Fatalf("expected initial credential version 1, got %#v", agent)
	}

	duplicate := decodeData[agentResponse](t, request(t, router, http.MethodPost, "/api/v1/agents/"+agent.ID+"/credentials:rotate", map[string]any{
		"credentials": map[string]any{
			"apiToken": "Bearer one",
		},
	}, ""))
	if duplicate.CredentialVersion != agent.CredentialVersion {
		t.Fatalf("duplicate credential rotation should be a no-op, got %#v", duplicate)
	}
	events, err := repo.ListAuditEvents(context.Background(), store.AuditEventFilter{Action: "agent.credentials_rotated"})
	if err != nil {
		t.Fatalf("list rotation audit events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("duplicate credential rotation should not append audit events: %#v", events)
	}

	rotated := decodeData[agentResponse](t, request(t, router, http.MethodPost, "/api/v1/agents/"+agent.ID+"/credentials:rotate", map[string]any{
		"credentials": map[string]any{
			"apiToken": "Bearer two",
		},
	}, ""))
	if rotated.CredentialVersion != agent.CredentialVersion+1 {
		t.Fatalf("changed credential rotation should increment version, got %#v", rotated)
	}
}

func TestTenantHierarchyAPIsAndScopedManagementLists(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)

	root := decodeData[tenantResponse](t, request(t, router, http.MethodPost, "/api/v1/tenants", map[string]any{
		"id":     "tenant-root",
		"name":   "Root Tenant",
		"status": "active",
	}, ""))
	child := decodeData[tenantResponse](t, request(t, router, http.MethodPost, "/api/v1/tenants", map[string]any{
		"id":             "tenant-child",
		"parentTenantId": root.ID,
		"name":           "Child Tenant",
		"status":         "active",
	}, ""))
	grandchild := decodeData[tenantResponse](t, request(t, router, http.MethodPost, "/api/v1/tenants", map[string]any{
		"id":             "tenant-grandchild",
		"parentTenantId": child.ID,
		"name":           "Grandchild Tenant",
		"status":         "active",
	}, ""))
	if root.Level != 1 || child.Level != 2 || grandchild.Level != 3 {
		t.Fatalf("unexpected tenant levels: root=%#v child=%#v grandchild=%#v", root, child, grandchild)
	}
	level4 := request(t, router, http.MethodPost, "/api/v1/tenants", map[string]any{
		"id":             "tenant-level-4",
		"parentTenantId": grandchild.ID,
		"name":           "Too Deep",
		"status":         "active",
	}, "")
	if level4.Code != http.StatusBadRequest {
		t.Fatalf("fourth-level tenant should fail, got %d body=%s", level4.Code, level4.Body.String())
	}

	createAgent(t, router, map[string]any{"tenantId": root.ID, "workspaceId": "ws-a", "name": "Root Agent", "channelType": "local", "status": "active"})
	createAgent(t, router, map[string]any{"tenantId": child.ID, "workspaceId": "ws-a", "name": "Child Agent", "channelType": "local", "status": "active"})
	createAgent(t, router, map[string]any{"tenantId": grandchild.ID, "workspaceId": "ws-a", "name": "Grandchild Agent", "channelType": "local", "status": "active"})
	createAgent(t, router, map[string]any{"tenantId": "tenant-unrelated", "workspaceId": "ws-a", "name": "Unrelated Agent", "channelType": "local", "status": "active"})

	tenants := decodeData[[]tenantResponse](t, request(t, router, http.MethodGet, "/api/v1/tenants?tenantId="+root.ID, nil, ""))
	if got := tenantResponseIDs(tenants); !reflect.DeepEqual(got, []string{root.ID, child.ID, grandchild.ID}) {
		t.Fatalf("tenant subtree = %#v", got)
	}
	children := decodeData[[]tenantResponse](t, request(t, router, http.MethodGet, "/api/v1/tenants?parentTenantId="+root.ID, nil, ""))
	if got := tenantResponseIDs(children); !reflect.DeepEqual(got, []string{child.ID}) {
		t.Fatalf("direct children = %#v", got)
	}
	agents := decodeData[[]agentResponse](t, request(t, router, http.MethodGet, "/api/v1/agents?tenantId="+root.ID+"&workspaceId=ws-a", nil, ""))
	if got := agentResponseTenantIDs(agents); !reflect.DeepEqual(got, []string{root.ID, child.ID, grandchild.ID}) {
		t.Fatalf("scoped agent tenants = %#v", got)
	}
}

func TestDisableAgentBlocksExistingKey(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller := createAgent(t, router, map[string]any{
		"name":        "Disposable Caller",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	})
	target := createDirectAgent(t, repo, "Stub MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)
	key := decodeData[keyResponse](t, request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId": caller.ID,
	}, ""))
	request(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
	}, "")

	beforeDisable := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if beforeDisable.Code != http.StatusOK {
		t.Fatalf("expected existing key to work before disable, got %d", beforeDisable.Code)
	}

	disabled := decodeData[agentResponse](t, request(t, router, http.MethodDelete, "/api/v1/agents/"+caller.ID, nil, ""))
	if disabled.Status != "disabled" {
		t.Fatalf("expected disabled agent response, got %#v", disabled)
	}
	afterDisable := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if afterDisable.Code != http.StatusUnauthorized {
		t.Fatalf("disabled caller key should be unauthorized, got %d", afterDisable.Code)
	}
}

func TestAgentRejectsSecretLikeHeaders(t *testing.T) {
	router := newRouter()
	for _, headerName := range []string{"Authorization", "Cookie", "X-Api-Key"} {
		resp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
			"name":        "Secret Header MCP",
			"workspaceId": "ws-1",
			"channelType": "mcp",
			"status":      "active",
			"channelConfig": map[string]any{
				"endpoint": "https://api.example.com/mcp",
				"headers": map[string]any{
					headerName: "should-not-live-here",
				},
			},
		}, "")
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("secret-like configured header %s should fail, got %d body=%s", headerName, resp.Code, resp.Body.String())
		}
	}
}

func TestAgentCredentialsAreAcceptedAndRedacted(t *testing.T) {
	router := newRouter()
	secret := "Bearer credential-redaction-secret"

	resp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Credentialed MCP",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
		"credentials": map[string]any{
			"apiToken": secret,
		},
	}, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("credentialed agent create failed: status=%d body=%s", resp.Code, resp.Body.String())
	}
	if strings.Contains(resp.Body.String(), secret) || strings.Contains(resp.Body.String(), "credentials") {
		t.Fatalf("create response leaked credentials: %s", resp.Body.String())
	}
	created := decodeData[agentResponse](t, resp)

	read := request(t, router, http.MethodGet, "/api/v1/agents/"+created.ID, nil, "")
	if read.Code != http.StatusOK {
		t.Fatalf("get credentialed agent failed: status=%d body=%s", read.Code, read.Body.String())
	}
	if strings.Contains(read.Body.String(), secret) || strings.Contains(read.Body.String(), "credentials") {
		t.Fatalf("get response leaked credentials: %s", read.Body.String())
	}

	list := request(t, router, http.MethodGet, "/api/v1/agents?workspaceId=ws-1", nil, "")
	if list.Code != http.StatusOK {
		t.Fatalf("list credentialed agent failed: status=%d body=%s", list.Code, list.Body.String())
	}
	if strings.Contains(list.Body.String(), secret) || strings.Contains(list.Body.String(), "credentials") {
		t.Fatalf("list response leaked credentials: %s", list.Body.String())
	}
}

func TestPatchAgentUpdatesMutableFieldsAndValidatesConfig(t *testing.T) {
	router := newRouter()
	secret := "Bearer patch-secret"
	createdResp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Patchable MCP",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "draft",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
		"credentials": map[string]any{
			"apiToken": secret,
		},
	}, "")
	if createdResp.Code != http.StatusCreated {
		t.Fatalf("create patchable agent failed: status=%d body=%s", createdResp.Code, createdResp.Body.String())
	}
	created := decodeData[agentResponse](t, createdResp)

	updatedResp := request(t, router, http.MethodPatch, "/api/v1/agents/"+created.ID, map[string]any{
		"name":        "Patched MCP",
		"description": "updated through partial patch",
		"ownerId":     "platform-team",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api-updated.example.com/mcp",
			"headers": map[string]any{
				"X-AgentHarbor-Tenant": "default",
			},
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
	}, "")
	if updatedResp.Code != http.StatusOK {
		t.Fatalf("patch agent should succeed, got %d body=%s", updatedResp.Code, updatedResp.Body.String())
	}
	if strings.Contains(updatedResp.Body.String(), secret) || strings.Contains(updatedResp.Body.String(), "credentials") {
		t.Fatalf("patch response leaked credentials: %s", updatedResp.Body.String())
	}
	updated := decodeData[agentResponse](t, updatedResp)
	if updated.Name != "Patched MCP" || updated.Description != "updated through partial patch" || updated.OwnerID != "platform-team" || updated.Status != "active" {
		t.Fatalf("unexpected patched agent metadata: %#v", updated)
	}
	if updated.ChannelConfig["endpoint"] != "https://api-updated.example.com/mcp" {
		t.Fatalf("unexpected patched channel config: %#v", updated.ChannelConfig)
	}

	secretConfig := request(t, router, http.MethodPatch, "/api/v1/agents/"+created.ID, map[string]any{
		"channelConfig": map[string]any{
			"endpoint":      "https://api.example.com/mcp",
			"Authorization": "do-not-store-here",
		},
	}, "")
	if secretConfig.Code != http.StatusBadRequest {
		t.Fatalf("secret-like patch channelConfig should fail, got %d body=%s", secretConfig.Code, secretConfig.Body.String())
	}

	arraySecretConfig := request(t, router, http.MethodPatch, "/api/v1/agents/"+created.ID, map[string]any{
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"metadata": []any{
				map[string]any{"authorization": "Bearer should-not-echo"},
			},
		},
	}, "")
	if arraySecretConfig.Code != http.StatusBadRequest {
		t.Fatalf("secret-like patch channelConfig inside arrays should fail, got %d body=%s", arraySecretConfig.Code, arraySecretConfig.Body.String())
	}

	unsafeEndpoint := request(t, router, http.MethodPatch, "/api/v1/agents/"+created.ID, map[string]any{
		"channelConfig": map[string]any{
			"endpoint": "http://127.0.0.1:8080/mcp",
		},
	}, "")
	if unsafeEndpoint.Code != http.StatusBadRequest {
		t.Fatalf("unsafe patch endpoint should fail, got %d body=%s", unsafeEndpoint.Code, unsafeEndpoint.Body.String())
	}
}

func TestRotateAgentCredentialsTakesEffectOnNextProxyCall(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	seenAuthorizations := []string{}
	originalTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		seenAuthorizations = append(seenAuthorizations, r.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, nil
	})
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	caller, key := createLocalCallerWithKey(t, router, "Rotate Caller")
	target := createDirectAgentWithCredentials(t, repo, "Rotate MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": "https://api.example.com/mcp",
		"credentialHeaders": map[string]any{
			"Authorization": "apiToken",
		},
	}, map[string]string{
		"apiToken": "Bearer old-token",
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	beforeRotate := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if beforeRotate.Code != http.StatusAccepted {
		t.Fatalf("expected proxy call before rotate, got %d body=%s", beforeRotate.Code, beforeRotate.Body.String())
	}

	rotateResp := request(t, router, http.MethodPost, "/api/v1/agents/"+target.ID+"/credentials:rotate", map[string]any{
		"credentials": map[string]any{
			"apiToken": "Bearer new-token",
		},
	}, "")
	if rotateResp.Code != http.StatusOK {
		t.Fatalf("rotate credentials should succeed, got %d body=%s", rotateResp.Code, rotateResp.Body.String())
	}
	if strings.Contains(rotateResp.Body.String(), "Bearer new-token") || strings.Contains(rotateResp.Body.String(), "credentials") {
		t.Fatalf("rotate response leaked credentials: %s", rotateResp.Body.String())
	}

	afterRotate := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if afterRotate.Code != http.StatusAccepted {
		t.Fatalf("expected proxy call after rotate, got %d body=%s", afterRotate.Code, afterRotate.Body.String())
	}
	if len(seenAuthorizations) != 2 || seenAuthorizations[0] != "Bearer old-token" || seenAuthorizations[1] != "Bearer new-token" {
		t.Fatalf("unexpected Authorization headers after rotation: %#v", seenAuthorizations)
	}
}

func TestManagementAuditEventsRecordAgentLifecycleWithoutSecrets(t *testing.T) {
	router := newRouter()
	oldSecret := "Bearer audit-old-secret"
	newSecret := "Bearer audit-new-secret"

	createResp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Audited MCP",
		"tenantId":    "tenant-audit",
		"workspaceId": "ws-audit",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
		"credentials": map[string]any{
			"apiToken": oldSecret,
		},
	}, "")
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create audited agent failed: status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	if strings.Contains(createResp.Body.String(), oldSecret) || strings.Contains(createResp.Body.String(), "credentials") {
		t.Fatalf("create response leaked credentials: %s", createResp.Body.String())
	}
	created := decodeData[agentResponse](t, createResp)
	if created.CredentialVersion != 1 {
		t.Fatalf("credentialed agent should start at version 1, got %#v", created)
	}

	updateResp := request(t, router, http.MethodPatch, "/api/v1/agents/"+created.ID, map[string]any{
		"name":   "Audited MCP Updated",
		"status": "draft",
	}, "")
	if updateResp.Code != http.StatusOK {
		t.Fatalf("patch audited agent failed: status=%d body=%s", updateResp.Code, updateResp.Body.String())
	}
	updated := decodeData[agentResponse](t, updateResp)
	if updated.CredentialVersion != 1 {
		t.Fatalf("metadata update should not change credential version: %#v", updated)
	}

	emptyRotateResp := request(t, router, http.MethodPost, "/api/v1/agents/"+created.ID+"/credentials:rotate", map[string]any{
		"credentials": map[string]any{},
	}, "")
	if emptyRotateResp.Code != http.StatusBadRequest {
		t.Fatalf("empty credential rotation should fail, got %d body=%s", emptyRotateResp.Code, emptyRotateResp.Body.String())
	}

	rotateResp := request(t, router, http.MethodPost, "/api/v1/agents/"+created.ID+"/credentials:rotate", map[string]any{
		"credentials": map[string]any{
			"apiToken": newSecret,
		},
	}, "")
	if rotateResp.Code != http.StatusOK {
		t.Fatalf("rotate audited credentials failed: status=%d body=%s", rotateResp.Code, rotateResp.Body.String())
	}
	if strings.Contains(rotateResp.Body.String(), oldSecret) || strings.Contains(rotateResp.Body.String(), newSecret) || strings.Contains(rotateResp.Body.String(), "credentials") {
		t.Fatalf("rotate response leaked credentials: %s", rotateResp.Body.String())
	}
	rotated := decodeData[agentResponse](t, rotateResp)
	if rotated.CredentialVersion != 2 {
		t.Fatalf("credential rotation should increment version to 2, got %#v", rotated)
	}

	eventsResp := request(t, router, http.MethodGet, "/api/v1/audit/events?resourceId="+created.ID, nil, "")
	if eventsResp.Code != http.StatusOK {
		t.Fatalf("list audit events failed: status=%d body=%s", eventsResp.Code, eventsResp.Body.String())
	}
	if strings.Contains(eventsResp.Body.String(), oldSecret) || strings.Contains(eventsResp.Body.String(), newSecret) {
		t.Fatalf("audit events leaked credential values: %s", eventsResp.Body.String())
	}
	events := decodeData[[]auditEventResponse](t, eventsResp)
	if got := auditActions(events); strings.Join(got, ",") != "agent.created,agent.updated,agent.credentials_rotated" {
		t.Fatalf("unexpected audit actions: %#v events=%#v", got, events)
	}
	rotation := events[2]
	if rotation.Metadata["credentialVersion"] != float64(2) {
		t.Fatalf("rotation audit event should include new credential version, got %#v", rotation.Metadata)
	}
	keys, ok := rotation.Metadata["credentialKeys"].([]any)
	if !ok || len(keys) != 1 || keys[0] != "apiToken" {
		t.Fatalf("rotation audit event should include credential key names only, got %#v", rotation.Metadata)
	}
}

func TestManagementAuditFailureBlocksAgentCreateAndUpdate(t *testing.T) {
	base := store.NewMemory()
	router := newRouterWithRepo(&failingAuditedAgentRepository{Repository: base})

	createResp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Audit Required Agent",
		"tenantId":    "tenant-audit-failure",
		"workspaceId": "ws-audit-failure",
		"channelType": "local",
		"status":      "active",
	}, "")
	if createResp.Code != http.StatusInternalServerError {
		t.Fatalf("create should fail when audit persistence fails: status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	agents, err := base.ListAgents(t.Context(), store.AgentFilter{ManagementScope: store.ManagementScope{
		TenantID:    "tenant-audit-failure",
		WorkspaceID: "ws-audit-failure",
	}})
	if err != nil {
		t.Fatalf("list agents after failed create: %v", err)
	}
	if len(agents) != 0 {
		t.Fatalf("failed audited create should not persist agent: %#v", agents)
	}

	existing := domain.Agent{
		ID:            security.NewID("agt"),
		TenantID:      "tenant-audit-failure",
		WorkspaceID:   "ws-audit-failure",
		Name:          "Original Agent",
		ChannelType:   "local",
		ChannelConfig: map[string]any{},
		Status:        domain.AgentStatusActive,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if _, err := base.CreateAgent(t.Context(), existing); err != nil {
		t.Fatalf("seed existing agent: %v", err)
	}

	updateResp := request(t, router, http.MethodPatch, "/api/v1/agents/"+existing.ID, map[string]any{
		"name": "Updated Agent",
	}, "")
	if updateResp.Code != http.StatusInternalServerError {
		t.Fatalf("update should fail when audit persistence fails: status=%d body=%s", updateResp.Code, updateResp.Body.String())
	}
	after, ok, err := base.GetAgent(t.Context(), existing.ID)
	if err != nil {
		t.Fatalf("get agent after failed update: %v", err)
	}
	if !ok {
		t.Fatalf("seeded agent disappeared after failed update")
	}
	if after.Name != "Original Agent" {
		t.Fatalf("failed audited update should keep previous name, got %q", after.Name)
	}
}

func TestRotateAgentCredentialsRejectsEmptyCredentialBag(t *testing.T) {
	router := newRouter()
	agent := createAgent(t, router, map[string]any{
		"name":        "No Credential Local",
		"workspaceId": "ws-audit",
		"channelType": "local",
		"status":      "active",
	})

	resp := request(t, router, http.MethodPost, "/api/v1/agents/"+agent.ID+"/credentials:rotate", map[string]any{
		"credentials": map[string]any{},
	}, "")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("empty credential rotation should fail, got %d body=%s", resp.Code, resp.Body.String())
	}
	read := decodeData[agentResponse](t, request(t, router, http.MethodGet, "/api/v1/agents/"+agent.ID, nil, ""))
	if read.CredentialVersion != 0 {
		t.Fatalf("rejected empty rotation should not change credential version, got %#v", read)
	}
}

func TestAgentRejectsInvalidProxyConfig(t *testing.T) {
	router := newRouter()
	badHeaderValue := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Header Value",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
		"channelConfig": map[string]any{
			"headers": map[string]any{
				"X-AgentHarbor-Tenant": 123,
			},
		},
	}, "")
	if badHeaderValue.Code != http.StatusBadRequest {
		t.Fatalf("non-string configured header should fail, got %d body=%s", badHeaderValue.Code, badHeaderValue.Body.String())
	}

	badHeaderName := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Header Name",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
		"channelConfig": map[string]any{
			"headers": map[string]any{
				"X-Bad\nHeader": "value",
			},
		},
	}, "")
	if badHeaderName.Code != http.StatusBadRequest {
		t.Fatalf("invalid configured header name should fail, got %d body=%s", badHeaderName.Code, badHeaderName.Body.String())
	}

	badCredentialValue := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Credential Value",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
		"credentials": map[string]any{
			"apiToken": "Bearer good\nX-Bad: injected",
		},
	}, "")
	if badCredentialValue.Code != http.StatusBadRequest {
		t.Fatalf("credential header value with newline should fail, got %d body=%s", badCredentialValue.Code, badCredentialValue.Body.String())
	}

	badCredentialKey := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Credential Key",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "Bearer copied-secret",
			},
		},
		"credentials": map[string]any{
			"Bearer copied-secret": "Bearer credential-redaction-secret",
		},
	}, "")
	if badCredentialKey.Code != http.StatusBadRequest {
		t.Fatalf("credential key that looks like secret material should fail, got %d body=%s", badCredentialKey.Code, badCredentialKey.Body.String())
	}

	badTimeout := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Timeout",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
		"channelConfig": map[string]any{
			"timeoutMs": 30001,
		},
	}, "")
	if badTimeout.Code != http.StatusBadRequest {
		t.Fatalf("out-of-range timeout should fail, got %d body=%s", badTimeout.Code, badTimeout.Body.String())
	}

	missingCredential := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Missing Credential",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
	}, "")
	if missingCredential.Code != http.StatusBadRequest {
		t.Fatalf("missing credential header reference should fail, got %d body=%s", missingCredential.Code, missingCredential.Body.String())
	}

	badRetryAttempts := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Retry Attempts",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
		"channelConfig": map[string]any{
			"retry": map[string]any{
				"maxAttempts": 5,
			},
		},
	}, "")
	if badRetryAttempts.Code != http.StatusBadRequest {
		t.Fatalf("out-of-range retry maxAttempts should fail, got %d body=%s", badRetryAttempts.Code, badRetryAttempts.Body.String())
	}

	badRetryStatus := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Retry Status",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
		"channelConfig": map[string]any{
			"retry": map[string]any{
				"statusCodes": []any{429},
			},
		},
	}, "")
	if badRetryStatus.Code != http.StatusBadRequest {
		t.Fatalf("non-5xx retry status should fail, got %d body=%s", badRetryStatus.Code, badRetryStatus.Body.String())
	}
}
