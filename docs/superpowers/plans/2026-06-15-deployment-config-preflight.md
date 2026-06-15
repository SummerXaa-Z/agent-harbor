# Deployment Config Preflight Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a production deployment configuration preflight that blocks unsafe production-mode startup while preserving local developer-preview behavior.

**Architecture:** Add pure app-level environment preflight helpers and run them from `app.New` before server construction. Extend the production hardening scenario to prove production mode rejects development-only flags and accepts a minimal safe configuration. Keep the slice backend/script/docs only, with no new dependencies and no permission semantics changes.

**Tech Stack:** Go app config, Bash scenario gate, Makefile release gates, Markdown docs.

---

## File Structure

- Modify `internal/app/app.go`
  - Add `deploymentModeFromEnv`, `deploymentConfigPreflightFromEnv`, and `validateDeploymentConfig`.
  - Run production preflight from `New`.
- Modify `internal/app/app_test.go`
  - Add tests for production blocking checks, production warnings, development mode compatibility, and invalid deployment mode.
- Modify `scripts/scenario-production-hardening.sh`
  - Add production-mode startup checks for unsafe flags and safe minimal config.
- Modify `README.md`, `docs/engineering/release-checklist.md`, and `CHANGELOG.md`
  - Document `AGENT_HARBOR_DEPLOYMENT_MODE=production` and the release-gate behavior.
- Modify this plan while executing
  - Check off each step after verification.

---

### Task 1: Add Production Config Preflight Tests

**Files:**
- Modify: `internal/app/app_test.go`

- [ ] **Step 1: Add failing tests**

Append these tests to `internal/app/app_test.go`:

```go
func TestDeploymentConfigPreflightBlocksUnsafeProductionFlags(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_ADMIN_KEY", "test-admin")
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", "stable-session-secret")
	t.Setenv("AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN", "true")
	t.Setenv("AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS", "true")

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected unsafe production config to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN") || !strings.Contains(got, "AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS") {
		t.Fatalf("expected production config error to name unsafe flags, got %q", got)
	}
}

func TestDeploymentConfigPreflightRequiresProductionAdminAuthentication(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", "stable-session-secret")

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected missing production admin authentication to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ADMIN_KEY") || !strings.Contains(got, "AGENT_HARBOR_ADMIN_IDENTITIES") {
		t.Fatalf("expected production config error to name admin authentication, got %q", got)
	}
}

func TestDeploymentConfigPreflightAllowsSafeProductionConfig(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_ADMIN_KEY", "test-admin")
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", "stable-session-secret")

	app, err := New(context.Background())
	if err != nil {
		t.Fatalf("safe production config should start: %v", err)
	}
	defer app.Close()
}

func TestDeploymentConfigPreflightWarnsForDerivedSessionSecret(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       "test-admin",
	})
	if err != nil {
		t.Fatalf("preflight should warn, not fail: %v", err)
	}
	if !hasDeploymentCheck(checks, "session_secret_explicit", "warning", "warning") {
		t.Fatalf("expected session secret warning, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightKeepsDevelopmentModeCompatible(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN", "true")

	app, err := New(context.Background())
	if err != nil {
		t.Fatalf("development mode should preserve explicit unauthenticated admin: %v", err)
	}
	defer app.Close()
}

func TestDeploymentModeRejectsInvalidValue(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "prod")

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected invalid deployment mode to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_DEPLOYMENT_MODE") {
		t.Fatalf("expected deployment mode error, got %q", got)
	}
}

func hasDeploymentCheck(checks []deploymentConfigCheck, code, severity, status string) bool {
	for _, check := range checks {
		if check.Code == code && check.Severity == severity && check.Status == status {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run app tests and confirm RED**

Run:

```bash
go test ./internal/app
```

Expected: FAIL because `deploymentConfigPreflightFromEnv`, `deploymentConfigCheck`, and deployment-mode validation do not exist yet.

---

### Task 2: Implement App-Level Deployment Preflight

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Add helper types and validation**

Add the following near the top-level app config helpers in `internal/app/app.go`:

```go
type deploymentConfigCheck struct {
	Code     string
	Severity string
	Status   string
	Message  string
}

func deploymentConfigPreflightFromEnv(env map[string]string) ([]deploymentConfigCheck, error) {
	mode, err := deploymentModeFromEnv(env)
	if err != nil {
		return nil, err
	}
	checks := validateDeploymentConfig(mode, env)
	var blockers []string
	for _, check := range checks {
		if check.Severity == "blocking" && check.Status == "failed" {
			blockers = append(blockers, check.Message)
		}
	}
	if len(blockers) > 0 {
		return checks, fmt.Errorf("production deployment config failed: %s", strings.Join(blockers, "; "))
	}
	return checks, nil
}

func deploymentModeFromEnv(env map[string]string) (string, error) {
	raw := strings.TrimSpace(env["AGENT_HARBOR_DEPLOYMENT_MODE"])
	if raw == "" {
		return "development", nil
	}
	switch raw {
	case "development", "production":
		return raw, nil
	default:
		return "", fmt.Errorf("AGENT_HARBOR_DEPLOYMENT_MODE must be development or production")
	}
}

func validateDeploymentConfig(mode string, env map[string]string) []deploymentConfigCheck {
	checks := []deploymentConfigCheck{
		{
			Code:     "deployment_mode",
			Severity: "info",
			Status:   "passed",
			Message:  "deployment mode is " + mode,
		},
	}
	if mode != "production" {
		return checks
	}

	checks = append(checks,
		productionBlockingCheck(
			"unauthenticated_admin_disabled",
			strings.EqualFold(strings.TrimSpace(env["AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN"]), "true"),
			"AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN must not be true in production",
			"unauthenticated admin access is disabled",
		),
		productionBlockingCheck(
			"private_upstreams_disabled",
			strings.EqualFold(strings.TrimSpace(env["AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS"]), "true"),
			"AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS must not be true in production",
			"private upstream access is disabled",
		),
		productionBlockingCheck(
			"admin_authentication_configured",
			strings.TrimSpace(env["AGENT_HARBOR_ADMIN_KEY"]) == "" && strings.TrimSpace(env["AGENT_HARBOR_ADMIN_IDENTITIES"]) == "",
			"AGENT_HARBOR_ADMIN_KEY or AGENT_HARBOR_ADMIN_IDENTITIES is required in production",
			"admin authentication is configured",
		),
	)

	if strings.TrimSpace(env["AGENT_HARBOR_SESSION_SECRET"]) == "" {
		checks = append(checks, deploymentConfigCheck{
			Code:     "session_secret_explicit",
			Severity: "warning",
			Status:   "warning",
			Message:  "AGENT_HARBOR_SESSION_SECRET should be set to a stable high-entropy value in production",
		})
	} else {
		checks = append(checks, deploymentConfigCheck{
			Code:     "session_secret_explicit",
			Severity: "warning",
			Status:   "passed",
			Message:  "AGENT_HARBOR_SESSION_SECRET is explicitly configured",
		})
	}
	return checks
}

func productionBlockingCheck(code string, failed bool, failedMessage string, passedMessage string) deploymentConfigCheck {
	check := deploymentConfigCheck{
		Code:     code,
		Severity: "blocking",
		Status:   "passed",
		Message:  passedMessage,
	}
	if failed {
		check.Status = "failed"
		check.Message = failedMessage
	}
	return check
}
```

- [ ] **Step 2: Build the environment map and call preflight from `New`**

Add:

```go
func deploymentEnvFromOS() map[string]string {
	keys := []string{
		"AGENT_HARBOR_DEPLOYMENT_MODE",
		"AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN",
		"AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS",
		"AGENT_HARBOR_ADMIN_KEY",
		"AGENT_HARBOR_ADMIN_IDENTITIES",
		"AGENT_HARBOR_SESSION_SECRET",
	}
	env := make(map[string]string, len(keys))
	for _, key := range keys {
		env[key] = os.Getenv(key)
	}
	return env
}
```

Call it in `New` after boolean/admin identity parsing and before `httpapi.New`:

```go
if _, err := deploymentConfigPreflightFromEnv(deploymentEnvFromOS()); err != nil {
	closeFn()
	return nil, err
}
```

- [ ] **Step 3: Run focused tests and fix compile issues**

Run:

```bash
go test ./internal/app
```

Expected: PASS.

---

### Task 3: Extend Production Hardening Scenario

**Files:**
- Modify: `scripts/scenario-production-hardening.sh`

- [ ] **Step 1: Add production-mode startup assertions**

After the unauthenticated admin fail-closed check and before starting the main API, add two short-lived startup checks:

```bash
if AGENT_HARBOR_ADDR="${API_HOST}:$((UNAUTH_API_PORT + 1))" \
  AGENT_HARBOR_DEPLOYMENT_MODE=production \
  AGENT_HARBOR_ADMIN_KEY="$ADMIN_KEY" \
  AGENT_HARBOR_SESSION_SECRET=production-hardening-session-secret \
  AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true \
  AGENT_HARBOR_DATABASE_URL= \
  AGENT_HARBOR_CREDENTIAL_KEY= \
  go run ./cmd/agent-harbor > "$LOG_DIR/api-prod-unsafe.log" 2>&1; then
	echo "expected production mode to reject AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true" >&2
	show_logs
	exit 1
fi
if ! grep -q "AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN" "$LOG_DIR/api-prod-unsafe.log"; then
	echo "production unsafe flag failure did not mention AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN" >&2
	show_logs
	exit 1
fi
echo "production deployment preflight rejects unauthenticated admin flag"

AGENT_HARBOR_ADDR="${API_HOST}:$((UNAUTH_API_PORT + 2))" \
AGENT_HARBOR_DEPLOYMENT_MODE=production \
AGENT_HARBOR_ADMIN_KEY="$ADMIN_KEY" \
AGENT_HARBOR_SESSION_SECRET=production-hardening-session-secret \
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=false \
AGENT_HARBOR_DATABASE_URL= \
AGENT_HARBOR_CREDENTIAL_KEY= \
  go run ./cmd/agent-harbor > "$LOG_DIR/api-prod-safe.log" 2>&1 &
PROD_SAFE_PID="$!"
PIDS+=("$PROD_SAFE_PID")
wait_http "Production config check API" "http://${API_HOST}:$((UNAUTH_API_PORT + 2))/healthz"
kill "$PROD_SAFE_PID" >/dev/null 2>&1 || true
wait "$PROD_SAFE_PID" >/dev/null 2>&1 || true
echo "production deployment preflight accepts safe minimal config"
```

- [ ] **Step 2: Run focused scenario verification**

Run:

```bash
bash -n scripts/scenario-production-hardening.sh
AGENT_HARBOR_PRODUCTION_GATE_API_PORT=9195 bash scripts/scenario-production-hardening.sh
```

Expected: PASS and the output includes both production deployment preflight messages.

---

### Task 4: Docs, Changelog, and Gates

**Files:**
- Modify: `README.md`
- Modify: `docs/engineering/release-checklist.md`
- Modify: `CHANGELOG.md`
- Modify: this plan

- [ ] **Step 1: Update README**

Document `AGENT_HARBOR_DEPLOYMENT_MODE` in the runtime configuration table and clarify that production mode blocks development-only flags.

- [ ] **Step 2: Update release checklist**

Add one sentence under the production safety baseline saying it also verifies production deployment preflight behavior.

- [ ] **Step 3: Update CHANGELOG**

Add under `## [Unreleased]`:

```md
- Production deployment mode now runs a configuration preflight that blocks development-only admin bypass and private-upstream flags before startup.
- 生产部署模式现在会执行配置预检，在启动前阻断开发专用的管理绕过和私有上游开关。
```

- [ ] **Step 4: Run full gates**

Run:

```bash
go test ./internal/app
make check
make release-check
git diff --check
```

Expected: all pass.

- [ ] **Step 5: Commit and create PR**

Run:

```bash
git add CHANGELOG.md README.md docs/engineering/release-checklist.md docs/superpowers/plans/2026-06-15-deployment-config-preflight.md internal/app/app.go internal/app/app_test.go scripts/scenario-production-hardening.sh
git commit -m "chore: add deployment config preflight"
git push -u origin codex/deployment-config-preflight
gh pr create --base main --head codex/deployment-config-preflight --title "Add deployment config preflight" --body "<summary and verification>"
```

Expected: PR opens against `main`.

---

## Self-Review

- Spec coverage: production-mode preflight, release scenario coverage, docs, tests, and PR flow are covered.
- Placeholder scan: no placeholder implementation steps remain.
- Scope check: no frontend UI, no new dependencies, and no permission semantics changes are included.
