# Production Journey Acceptance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the v0.2.0 console read as one production-ready operator journey from setup to access query, permission change, status check, and handoff.

**Architecture:** Add one pure frontend journey model, render it through a compact checkpoint component, and wire it into existing workspaces without adding backend APIs or new dependencies. Keep permission decisions in existing backend/readiness/presenter logic; the new model only derives operator stage, next action, and navigation context.

**Tech Stack:** React 19, TypeScript 6, Vite 8, Node `node:test`, existing shell scenario gates.

---

## File Structure

- Create `frontend/src/productionJourney.ts`
  - Pure model for current journey stage, completed stages, next action, and target hash.
- Create `frontend/tests/productionJourney.test.mjs`
  - Unit tests for empty, configured, denied, in-change, ready, and blocked journey states.
- Create `frontend/src/components/ProductionJourneyCheckpoint.tsx`
  - Compact visual treatment for "where am I / what is next"; no API calls and no local state.
- Modify `frontend/src/ConsoleController.tsx`
  - Derive the production journey from existing data and pass a checkpoint into the primary workspaces.
- Modify `frontend/src/components/ConsoleViews.tsx`
  - Accept optional checkpoint slots in Getting Started, Ask, Registry, AiAdmin, Access, and Evidence views.
- Modify `frontend/src/i18n.ts`
  - Add checkpoint copy and replace visible "evidence" / "证据" labels with acceptance/records/status wording.
- Modify `frontend/src/styles.css`
  - Add thin checkpoint styles using existing tokens.
- Modify `frontend/tests/consoleNavigation.test.mjs`, `frontend/tests/styleTheme.test.mjs`, and a new or existing i18n guard test
  - Lock routing, component structure, wording, and no-new-dependency boundary.
- Modify `CHANGELOG.md`
  - Add EN + zh-CN user-facing entry.
- Modify this plan while executing
  - Check off each completed step.

---

### Task 1: Pure Production Journey Model

**Files:**
- Create: `frontend/src/productionJourney.ts`
- Create: `frontend/tests/productionJourney.test.mjs`

- [x] **Step 1: Write the failing model tests**

Create `frontend/tests/productionJourney.test.mjs`:

```js
import assert from "node:assert/strict";
import test from "node:test";

import {
  deriveProductionJourney,
  productionJourneyStages
} from "../src/productionJourney.ts";

const now = "2026-06-15T10:00:00Z";

function consoleData(overrides = {}) {
  return {
    accessGrants: [],
    agents: [],
    apiBase: "http://127.0.0.1:9090",
    auditEvents: [],
    capabilities: [],
    capabilitiesLoadedFromApi: true,
    capabilityAssignmentsLoadedFromApi: true,
    channels: [],
    evidenceRuns: [],
    grantsLoadedFromApi: true,
    instanceAssignments: [],
    loadedFromApi: true,
    providers: [],
    routePolicies: [],
    routePoliciesLoadedFromApi: true,
    setupLoadedFromApi: true,
    systemMetrics: [],
    tenantEntitlements: [],
    tenants: [],
    traces: [],
    workspaceAssignments: [],
    ...overrides
  };
}

const tenant = { createdAt: now, id: "tenant-root", level: 0, name: "Customer Service", status: "active", updatedAt: now };
const caller = {
  channelType: "local",
  createdAt: now,
  credentialVersion: 1,
  id: "agent-caller",
  name: "Support Assistant",
  status: "active",
  tenantId: tenant.id,
  updatedAt: now,
  workspaceId: "ws-support"
};
const target = { ...caller, channelType: "mcp", id: "agent-mcp", name: "Ticket Tool Service" };
const capability = {
  action: "read",
  dataDomains: ["support"],
  description: "Search tickets.",
  discoveredAt: now,
  discoveryStatus: "approved",
  displayName: "Search tickets",
  enforcementMode: "gateway",
  id: "cap-search",
  key: "search_ticket",
  riskLevel: "low",
  sensitivity: "internal",
  targetId: target.id,
  type: "mcp_tool",
  updatedAt: now,
  version: 1
};
const entitlement = {
  capabilityId: capability.id,
  createdAt: now,
  effect: "allow",
  id: "ent-support",
  priority: 50,
  status: "enabled",
  targetId: target.id,
  tenantId: tenant.id,
  updatedAt: now
};

function configuredData(overrides = {}) {
  return consoleData({
    agents: [caller, target],
    capabilities: [capability],
    tenantEntitlements: [entitlement],
    tenants: [tenant],
    ...overrides
  });
}

test("production journey exposes a stable stage order", () => {
  assert.deepEqual(productionJourneyStages.map((stage) => stage.key), [
    "setup",
    "resources",
    "access_query",
    "permission_change",
    "go_live_status",
    "handoff"
  ]);
});

test("empty live systems start at setup and point to getting started", () => {
  const journey = deriveProductionJourney({ data: consoleData(), activeNav: "getting-started" });

  assert.equal(journey.state, "empty");
  assert.equal(journey.currentStageKey, "setup");
  assert.equal(journey.nextActionHash, "#getting-started");
  assert.equal(journey.nextActionKey, "productionJourney.next.setup");
  assert.deepEqual(journey.completedStageKeys, []);
});

test("configured systems default to access query before a decision exists", () => {
  const journey = deriveProductionJourney({ data: configuredData(), activeNav: "ask" });

  assert.equal(journey.state, "configured");
  assert.equal(journey.currentStageKey, "access_query");
  assert.equal(journey.nextActionHash, "#ask");
  assert.equal(journey.nextActionKey, "productionJourney.next.ask");
  assert.deepEqual(journey.completedStageKeys, ["setup", "resources"]);
});

test("denied access query points to permission change without marking it complete", () => {
  const journey = deriveProductionJourney({
    accessOutcome: "denied",
    data: configuredData(),
    activeNav: "ask"
  });

  assert.equal(journey.state, "denied");
  assert.equal(journey.currentStageKey, "access_query");
  assert.equal(journey.nextActionHash, "#ai-admin");
  assert.equal(journey.nextActionKey, "productionJourney.next.fixDenied");
  assert.deepEqual(journey.completedStageKeys, ["setup", "resources"]);
});

test("permission change handoff marks the journey as in change", () => {
  const journey = deriveProductionJourney({
    data: configuredData(),
    hasPermissionChangeContext: true,
    activeNav: "ai-admin"
  });

  assert.equal(journey.state, "in_change");
  assert.equal(journey.currentStageKey, "permission_change");
  assert.equal(journey.nextActionHash, "#ai-admin");
  assert.equal(journey.nextActionKey, "productionJourney.next.completePermissionChange");
});

test("ready permission changes point to go-live status and handoff", () => {
  const journey = deriveProductionJourney({
    data: configuredData({ traces: [{ createdAt: now, decision: "allowed", id: "trace-1", routeType: "mcp", targetAgentId: target.id }] }),
    permissionReady: true,
    activeNav: "ai-admin"
  });

  assert.equal(journey.state, "ready");
  assert.equal(journey.currentStageKey, "go_live_status");
  assert.equal(journey.nextActionHash, "#evidence");
  assert.equal(journey.nextActionKey, "productionJourney.next.confirmGoLive");
  assert.deepEqual(journey.completedStageKeys, ["setup", "resources", "access_query", "permission_change"]);
});

test("blocked permission changes stay on the permission change workspace", () => {
  const journey = deriveProductionJourney({
    data: configuredData(),
    permissionBlocked: true,
    activeNav: "evidence"
  });

  assert.equal(journey.state, "blocked");
  assert.equal(journey.currentStageKey, "permission_change");
  assert.equal(journey.nextActionHash, "#ai-admin");
  assert.equal(journey.nextActionKey, "productionJourney.next.resolveBlocker");
});
```

- [x] **Step 2: Run the model tests and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/productionJourney.test.mjs
```

Expected: FAIL because `frontend/src/productionJourney.ts` does not exist.

- [x] **Step 3: Implement the pure model**

Create `frontend/src/productionJourney.ts`:

```ts
import type { NavKey } from "./consoleNavigation";
import type { ConsoleData } from "./types";

export type ProductionJourneyStageKey =
  | "setup"
  | "resources"
  | "access_query"
  | "permission_change"
  | "go_live_status"
  | "handoff";

export type ProductionJourneyState =
  | "empty"
  | "partial"
  | "configured"
  | "denied"
  | "in_change"
  | "ready"
  | "blocked";

export interface ProductionJourneyStage {
  key: ProductionJourneyStageKey;
  labelKey: string;
}

export interface ProductionJourneyInput {
  accessOutcome?: "allowed" | "denied" | null;
  activeNav?: NavKey;
  data: ConsoleData;
  hasPermissionChangeContext?: boolean;
  permissionBlocked?: boolean;
  permissionReady?: boolean;
}

export interface ProductionJourney {
  completedStageKeys: ProductionJourneyStageKey[];
  currentStageKey: ProductionJourneyStageKey;
  nextActionHash: string;
  nextActionKey: string;
  state: ProductionJourneyState;
}

export const productionJourneyStages: ProductionJourneyStage[] = [
  { key: "setup", labelKey: "productionJourney.stage.setup" },
  { key: "resources", labelKey: "productionJourney.stage.resources" },
  { key: "access_query", labelKey: "productionJourney.stage.accessQuery" },
  { key: "permission_change", labelKey: "productionJourney.stage.permissionChange" },
  { key: "go_live_status", labelKey: "productionJourney.stage.goLiveStatus" },
  { key: "handoff", labelKey: "productionJourney.stage.handoff" }
];

export function deriveProductionJourney(input: ProductionJourneyInput): ProductionJourney {
  const setupComplete = isProductionSetupComplete(input.data);
  const hasLiveSetupData = input.data.setupLoadedFromApi;
  const hasAnyConfiguredResource =
    input.data.tenants.length > 0 ||
    input.data.agents.length > 0 ||
    input.data.capabilities.length > 0 ||
    input.data.tenantEntitlements.length > 0;

  if (!setupComplete) {
    return {
      completedStageKeys: [],
      currentStageKey: "setup",
      nextActionHash: hasAnyConfiguredResource ? "#registry" : "#getting-started",
      nextActionKey: hasAnyConfiguredResource
        ? "productionJourney.next.continueSetup"
        : "productionJourney.next.setup",
      state: hasLiveSetupData && hasAnyConfiguredResource ? "partial" : "empty"
    };
  }

  if (input.permissionBlocked) {
    return {
      completedStageKeys: ["setup", "resources", "access_query"],
      currentStageKey: "permission_change",
      nextActionHash: "#ai-admin",
      nextActionKey: "productionJourney.next.resolveBlocker",
      state: "blocked"
    };
  }

  if (input.permissionReady) {
    return {
      completedStageKeys: ["setup", "resources", "access_query", "permission_change"],
      currentStageKey: "go_live_status",
      nextActionHash: "#evidence",
      nextActionKey: "productionJourney.next.confirmGoLive",
      state: "ready"
    };
  }

  if (input.hasPermissionChangeContext || input.activeNav === "ai-admin") {
    return {
      completedStageKeys: ["setup", "resources", "access_query"],
      currentStageKey: "permission_change",
      nextActionHash: "#ai-admin",
      nextActionKey: "productionJourney.next.completePermissionChange",
      state: "in_change"
    };
  }

  if (input.accessOutcome === "denied") {
    return {
      completedStageKeys: ["setup", "resources"],
      currentStageKey: "access_query",
      nextActionHash: "#ai-admin",
      nextActionKey: "productionJourney.next.fixDenied",
      state: "denied"
    };
  }

  return {
    completedStageKeys: ["setup", "resources"],
    currentStageKey: "access_query",
    nextActionHash: "#ask",
    nextActionKey: "productionJourney.next.ask",
    state: "configured"
  };
}

function isProductionSetupComplete(data: ConsoleData) {
  return (
    data.setupLoadedFromApi &&
    data.tenants.length > 0 &&
    data.agents.some((agent) => agent.status === "active") &&
    data.capabilities.length > 0 &&
    data.tenantEntitlements.length > 0
  );
}
```

- [x] **Step 4: Run the model tests and confirm GREEN**

Run:

```bash
pnpm --dir frontend exec node --test tests/productionJourney.test.mjs
```

Expected: PASS with 7 tests.

---

### Task 2: Compact Production Journey Checkpoint

**Files:**
- Create: `frontend/src/components/ProductionJourneyCheckpoint.tsx`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/src/styles.css`
- Modify: `frontend/tests/styleTheme.test.mjs`

- [x] **Step 1: Add failing structure tests for the checkpoint**

Extend `frontend/tests/styleTheme.test.mjs` with:

```js
const productionJourney = readExistingFile(new URL("../src/productionJourney.ts", import.meta.url));
const productionJourneyCheckpoint = readExistingFile(new URL("../src/components/ProductionJourneyCheckpoint.tsx", import.meta.url));
```

Add this test near the other component-structure tests:

```js
test("production journey checkpoint stays compact and model-driven", () => {
  assert.match(productionJourney, /export function deriveProductionJourney/);
  assert.match(productionJourney, /export const productionJourneyStages/);
  assert.match(productionJourneyCheckpoint, /export function ProductionJourneyCheckpoint/);
  assert.match(productionJourneyCheckpoint, /productionJourneyStages\.map/);
  assert.match(productionJourneyCheckpoint, /className="production-journey-checkpoint"/);
  assert.match(productionJourneyCheckpoint, /href=\{journey\.nextActionHash\}/);
  assert.doesNotMatch(productionJourneyCheckpoint, /useState|useEffect|fetch\(/);
  assert.match(styles, /\.production-journey-checkpoint\s*\{/);
  assert.match(styles, /\.production-journey-checkpoint\s*\{[^}]*display:\s*grid;/s);
  assert.doesNotMatch(styles, /\.production-journey-checkpoint\s*\{[^}]*box-shadow:/s);
});
```

- [x] **Step 2: Run structure tests and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs
```

Expected: FAIL because the checkpoint component and styles do not exist.

- [x] **Step 3: Add checkpoint i18n copy**

Add EN keys in `frontend/src/i18n.ts`:

```ts
"productionJourney.aria": "Production journey",
"productionJourney.kicker": "Production path",
"productionJourney.nextLabel": "Next",
"productionJourney.stage.setup": "Setup",
"productionJourney.stage.resources": "Resources",
"productionJourney.stage.accessQuery": "Access query",
"productionJourney.stage.permissionChange": "Permission change",
"productionJourney.stage.goLiveStatus": "Go-live status",
"productionJourney.stage.handoff": "Handoff",
"productionJourney.next.setup": "Start setup",
"productionJourney.next.continueSetup": "Continue resource setup",
"productionJourney.next.ask": "Check access",
"productionJourney.next.fixDenied": "Start permission fix",
"productionJourney.next.completePermissionChange": "Complete permission change",
"productionJourney.next.confirmGoLive": "Confirm go-live status",
"productionJourney.next.resolveBlocker": "Resolve blocker",
```

Add zh-CN keys:

```ts
"productionJourney.aria": "生产旅程",
"productionJourney.kicker": "生产路径",
"productionJourney.nextLabel": "下一步",
"productionJourney.stage.setup": "初始化",
"productionJourney.stage.resources": "资源接入",
"productionJourney.stage.accessQuery": "访问查询",
"productionJourney.stage.permissionChange": "权限变更",
"productionJourney.stage.goLiveStatus": "上线状态",
"productionJourney.stage.handoff": "交接",
"productionJourney.next.setup": "开始配置",
"productionJourney.next.continueSetup": "继续接入资源",
"productionJourney.next.ask": "查询访问",
"productionJourney.next.fixDenied": "发起权限修复",
"productionJourney.next.completePermissionChange": "完成权限变更",
"productionJourney.next.confirmGoLive": "确认上线状态",
"productionJourney.next.resolveBlocker": "处理阻断项",
```

- [x] **Step 4: Implement the checkpoint component**

Create `frontend/src/components/ProductionJourneyCheckpoint.tsx`:

```tsx
import { ArrowRight, Check } from "lucide-react";

import type { Translator } from "../consolePresenters";
import {
  productionJourneyStages,
  type ProductionJourney
} from "../productionJourney";

export function ProductionJourneyCheckpoint({
  journey,
  t
}: {
  journey: ProductionJourney;
  t: Translator;
}) {
  const completed = new Set(journey.completedStageKeys);

  return (
    <section className="production-journey-checkpoint" aria-label={t("productionJourney.aria")}>
      <div className="production-journey-copy">
        <span className="section-kicker">{t("productionJourney.kicker")}</span>
        <strong>{t(journey.nextActionKey)}</strong>
      </div>
      <ol className="production-journey-steps">
        {productionJourneyStages.map((stage) => {
          const complete = completed.has(stage.key);
          const current = journey.currentStageKey === stage.key;
          return (
            <li className={current ? "is-current" : complete ? "is-complete" : ""} key={stage.key}>
              <span aria-hidden="true">{complete ? <Check size={12} /> : null}</span>
              {t(stage.labelKey)}
            </li>
          );
        })}
      </ol>
      <a className="secondary-button production-journey-next" href={journey.nextActionHash}>
        <span>{t("productionJourney.nextLabel")}</span>
        <ArrowRight aria-hidden="true" size={14} />
      </a>
    </section>
  );
}
```

- [x] **Step 5: Add compact styles**

Append to `frontend/src/styles.css` near other production-console structures:

```css
.production-journey-checkpoint {
  align-items: center;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  display: grid;
  gap: var(--space-4);
  grid-template-columns: minmax(160px, 0.9fr) minmax(0, 2fr) auto;
  padding: var(--space-3) var(--space-4);
}

.production-journey-copy {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.production-journey-copy strong {
  color: var(--text);
  font-size: var(--text-sm);
}

.production-journey-steps {
  align-items: center;
  display: grid;
  gap: var(--space-2);
  grid-template-columns: repeat(6, minmax(0, 1fr));
  list-style: none;
  margin: 0;
  padding: 0;
}

.production-journey-steps li {
  align-items: center;
  color: var(--muted);
  display: inline-flex;
  font-size: var(--text-xs);
  font-weight: 600;
  gap: var(--space-1);
  min-width: 0;
  white-space: nowrap;
}

.production-journey-steps li span {
  align-items: center;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-pill);
  display: inline-flex;
  height: 16px;
  justify-content: center;
  width: 16px;
}

.production-journey-steps li.is-complete {
  color: var(--success);
}

.production-journey-steps li.is-complete span {
  background: var(--success-soft);
  border-color: var(--success-border);
}

.production-journey-steps li.is-current {
  color: var(--brand);
}

.production-journey-steps li.is-current span {
  background: var(--brand-soft);
  border-color: var(--brand-border);
}

.production-journey-next {
  justify-self: end;
}
```

Add to the existing responsive section:

```css
@media (max-width: 1120px) {
  .production-journey-checkpoint {
    grid-template-columns: 1fr;
  }

  .production-journey-steps {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .production-journey-next {
    justify-self: start;
  }
}
```

- [x] **Step 6: Run focused component tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs
```

Expected: PASS.

---

### Task 3: Wire Checkpoint Into Primary Workspaces

**Files:**
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/src/components/ConsoleViews.tsx`
- Modify: `frontend/tests/styleTheme.test.mjs`
- Modify: `frontend/tests/consoleNavigation.test.mjs`

- [x] **Step 1: Add failing integration guards**

Extend `frontend/tests/styleTheme.test.mjs`:

```js
test("production journey checkpoint is wired through primary journey views", () => {
  assert.match(app, /from "\.\/productionJourney"/);
  assert.match(app, /from "\.\/components\/ProductionJourneyCheckpoint"/);
  assert.match(app, /deriveProductionJourney\(/);
  assert.match(app, /const productionJourneyCheckpoint = \(/);
  assert.match(consoleViews, /journeyCheckpoint/);
  assert.match(consoleViews, /GettingStartedConsoleView[\s\S]*journeyCheckpoint/);
  assert.match(consoleViews, /AskView[\s\S]*journeyCheckpoint/);
  assert.match(consoleViews, /RegistryView[\s\S]*journeyCheckpoint/);
  assert.match(consoleViews, /AiAdminView[\s\S]*journeyCheckpoint/);
  assert.match(consoleViews, /EvidenceView[\s\S]*journeyCheckpoint/);
});
```

Extend `frontend/tests/consoleNavigation.test.mjs`:

```js
test("production journey acceptance keeps existing primary routes", () => {
  assert.equal(viewForNav("getting-started").primaryPanelKey, "gettingStarted");
  assert.equal(viewForNav("ask").primaryPanelKey, "askAccess");
  assert.equal(viewForNav("registry").primaryPanelKey, "resourceLifecycle");
  assert.equal(viewForNav("ai-admin").primaryPanelKey, "aiAdminPermissionWorkbench");
  assert.equal(viewForNav("evidence").primaryPanelKey, "goLiveAcceptance");
});
```

- [x] **Step 2: Run integration guards and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/consoleNavigation.test.mjs
```

Expected: FAIL because the checkpoint is not wired.

- [x] **Step 3: Update ConsoleViews slots**

In `frontend/src/components/ConsoleViews.tsx`, update view signatures:

```tsx
export function AiAdminView({ aiAdminPanel, journeyCheckpoint }: { aiAdminPanel: ReactNode; journeyCheckpoint?: ReactNode }) {
  return (
    <section className="content-grid">
      {journeyCheckpoint}
      {aiAdminPanel}
    </section>
  );
}

export function GettingStartedConsoleView({ gettingStartedPanel, journeyCheckpoint }: { gettingStartedPanel: ReactNode; journeyCheckpoint?: ReactNode }) {
  return (
    <section className="content-grid">
      {journeyCheckpoint}
      {gettingStartedPanel}
    </section>
  );
}

export function AskView({ askAccessPanel, journeyCheckpoint }: { askAccessPanel: ReactNode; journeyCheckpoint?: ReactNode }) {
  return (
    <section className="content-grid">
      {journeyCheckpoint}
      {askAccessPanel}
    </section>
  );
}
```

Apply the same `journeyCheckpoint?: ReactNode` prop to `RegistryView`, `AccessView`, and `EvidenceView`, rendering `{journeyCheckpoint}` as the first child inside each `content-grid`.

- [x] **Step 4: Derive and render the checkpoint in ConsoleController**

Add imports:

```ts
import {
  deriveProductionJourney
} from "./productionJourney";
import { ProductionJourneyCheckpoint } from "./components/ProductionJourneyCheckpoint";
```

After `aiAdminProductionConsoleSummary` is defined, add:

```ts
  const productionJourney = useMemo(
    () =>
      data
        ? deriveProductionJourney({
            accessOutcome: askAccess.result?.outcome ?? null,
            activeNav,
            data,
            hasPermissionChangeContext: Boolean(handoffContexts.permissionNotice || handoffContexts.permissionChange),
            permissionBlocked: aiAdminProductionReadiness?.status === "blocked" || aiAdminProductionConsoleSummary.status === "blocked",
            permissionReady: aiAdminProductionReadiness?.status === "ready" || aiAdminProductionConsoleSummary.status === "ready"
          })
        : null,
    [
      activeNav,
      aiAdminProductionConsoleSummary.status,
      aiAdminProductionReadiness?.status,
      askAccess.result?.outcome,
      data,
      handoffContexts.permissionChange,
      handoffContexts.permissionNotice
    ]
  );
  const productionJourneyCheckpoint = productionJourney ? (
    <ProductionJourneyCheckpoint journey={productionJourney} t={t} />
  ) : null;
```

Pass `journeyCheckpoint={productionJourneyCheckpoint}` to:

```tsx
<AskView askAccessPanel={askAccessPanel} journeyCheckpoint={productionJourneyCheckpoint} />
<AiAdminView aiAdminPanel={aiAdminPanel} journeyCheckpoint={productionJourneyCheckpoint} />
<GettingStartedConsoleView gettingStartedPanel={...} journeyCheckpoint={productionJourneyCheckpoint} />
<RegistryView ... journeyCheckpoint={productionJourneyCheckpoint} />
<AccessView accessProfilePanel={accessProfilePanel} journeyCheckpoint={productionJourneyCheckpoint} />
<EvidenceView ... journeyCheckpoint={productionJourneyCheckpoint} />
```

- [x] **Step 5: Run focused integration tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/consoleNavigation.test.mjs tests/productionJourney.test.mjs
```

Expected: PASS.

---

### Task 4: User-Facing Wording Cleanup

**Files:**
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/i18n.test.mjs` or create `frontend/tests/productionLanguage.test.mjs`

- [x] **Step 1: Add failing wording guard**

Create `frontend/tests/productionLanguage.test.mjs`:

```js
import assert from "node:assert/strict";
import test from "node:test";

import {
  createTranslator,
  translationKeys
} from "../src/i18n.ts";

test("visible production copy avoids evidence wording", () => {
  const allowedKeyFragments = [
    "exportProductionEvidence",
    "productionEvidence",
    "evidenceLayer"
  ];

  for (const language of ["en", "zh-CN"]) {
    const t = createTranslator(language);
    const offending = translationKeys(language)
      .filter((key) => !allowedKeyFragments.some((fragment) => key.includes(fragment)))
      .map((key) => [key, t(key)])
      .filter(([, value]) => /证据|\bevidence\b/i.test(value));

    assert.deepEqual(offending, []);
  }
});
```

- [x] **Step 2: Run wording guard and confirm RED if visible wording remains**

Run:

```bash
pnpm --dir frontend exec node --test tests/productionLanguage.test.mjs
```

Expected: FAIL if any user-facing translation value still contains visible `evidence` or `证据`; otherwise PASS and proceed to Step 4.

- [x] **Step 3: Replace visible wording**

In `frontend/src/i18n.ts`, replace visible values:

- "Evidence" -> "Records" or "Acceptance" depending on context.
- "evidence" -> "records", "acceptance", "runtime logs", "audit records", or "handoff material".
- "证据" -> "记录", "验收记录", "运行记录", "审计记录", "交接材料", or "验收明细".

Do not rename these internal keys in this slice:

```ts
"action.exportProductionEvidence"
"action.exportingProductionEvidence"
"error.exportProductionEvidence"
"message.productionEvidenceExported"
"message.productionEvidenceRequiresLiveApi"
"ask.evidenceLayer.*"
"journey.aiAdmin.evidence.*"
```

Only change their displayed values if needed.

- [x] **Step 4: Run wording and i18n tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/productionLanguage.test.mjs tests/i18n.test.mjs
```

Expected: PASS.

---

### Task 5: Docs, Verification, and PR

**Files:**
- Modify: `CHANGELOG.md`
- Modify: `docs/superpowers/plans/2026-06-15-production-journey-acceptance.md`

- [x] **Step 1: Update CHANGELOG**

Add under `## Unreleased`:

```md
- Web console now shows a compact production journey checkpoint across setup, resource management, access query, permission changes, and go-live status, so operators can see the current stage and next safe action without reading technical identifiers.
- Web 控制台现在在开始使用、资源管理、访问查询、权限变更和上线状态中展示轻量生产旅程提示，管理员无需阅读技术标识即可确认当前阶段和下一步安全动作。
- User-facing console copy now avoids "evidence/证据" wording in primary labels, using acceptance records, runtime records, audit records, handoff material, and go-live status language instead.
- 控制台用户可见文案不再使用 “evidence/证据” 作为主路径表达，统一改为验收记录、运行记录、审计记录、交接材料和上线状态等业务语言。
```

- [x] **Step 2: Run focused gates**

Run:

```bash
pnpm --dir frontend exec node --test tests/productionJourney.test.mjs tests/productionLanguage.test.mjs tests/styleTheme.test.mjs tests/consoleNavigation.test.mjs
pnpm --dir frontend exec tsc -p tsconfig.json --noEmit
git diff --check
```

Expected: all commands exit 0.

- [x] **Step 3: Run full frontend and repository gates**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
make check
make release-check
```

Expected: all commands exit 0.

- [x] **Step 4: Browser smoke**

Run a temporary isolated demo, then inspect in the browser:

```bash
AGENT_HARBOR_DEMO_API_PORT=9192 MOCK_MCP_PORT=8792 AGENT_HARBOR_DEMO_FRONTEND_PORT=5182 scripts/demo.sh
```

Verify:

1. `http://127.0.0.1:5182/` opens on Getting Started for an empty system and shows the checkpoint.
2. `#registry`, `#ask`, `#ai-admin`, and `#evidence` show the same compact checkpoint without horizontal overflow.
3. User-visible Chinese primary labels do not show `证据`.

Record whether screenshot capture succeeds; if it times out, record DOM/interaction evidence without claiming screenshot proof.

Result:
- `http://127.0.0.1:5182/` opened on `#getting-started` with the production journey checkpoint visible.
- `#registry`, `#ask`, `#ai-admin`, and `#evidence` each showed the same compact checkpoint with no horizontal overflow at the in-app browser viewport.
- Chinese visible text on those pages did not include `证据`.
- Screenshot capture succeeded on `#evidence` (`79258` bytes).

- [x] **Step 5: Commit and create PR**

Run:

```bash
git add CHANGELOG.md docs/superpowers/plans/2026-06-15-production-journey-acceptance.md frontend/src/productionJourney.ts frontend/src/components/ProductionJourneyCheckpoint.tsx frontend/src/components/ConsoleViews.tsx frontend/src/ConsoleController.tsx frontend/src/i18n.ts frontend/src/styles.css frontend/tests/productionJourney.test.mjs frontend/tests/productionLanguage.test.mjs frontend/tests/styleTheme.test.mjs frontend/tests/consoleNavigation.test.mjs
git commit -m "feat: add production journey checkpoint"
git push -u origin codex/production-journey-acceptance
gh pr create --base main --head codex/production-journey-acceptance --title "Add production journey acceptance checkpoint" --body "<summary and test plan>"
```

Expected: PR opens against `main` and CI starts.

---

## Self-Review

- Spec coverage: pure model, compact UI, cross-workspace orientation, wording cleanup, no backend dependency, no frontend package dependency, docs, focused tests, repository gates, and browser smoke are covered.
- Placeholder scan: no placeholder steps remain; code snippets and commands are explicit.
- Type consistency: `ProductionJourney`, `ProductionJourneyStageKey`, `deriveProductionJourney`, and `ProductionJourneyCheckpoint` names are consistent across tasks.
