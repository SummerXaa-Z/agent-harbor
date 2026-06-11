export type NavKey =
  | "getting-started"
  | "cockpit"
  | "ai-admin"
  | "registry"
  | "routes"
  | "policies"
  | "capabilities"
  | "access"
  | "traces"
  | "evidence"

export type NavGroupKey = "onboarding" | "configuration" | "primary" | "audit"

export interface NavItem {
  detailKey: string
  groupKey: NavGroupKey
  key: NavKey
  label: string
}

export interface ConsoleView {
  key: NavKey
  primaryPanelKey: string
  titleKey: string
}

export const defaultNavKey: NavKey = "ai-admin"
const navHashPrefix = "#"

export const navGroups: Array<{ key: NavGroupKey; labelKey: string }> = [
  { key: "onboarding", labelKey: "navGroup.onboarding" },
  { key: "configuration", labelKey: "navGroup.configuration" },
  { key: "primary", labelKey: "navGroup.primary" },
  { key: "audit", labelKey: "navGroup.audit" },
]

export const navItems: NavItem[] = [
  { detailKey: "navDetail.getting-started", groupKey: "onboarding", key: "getting-started", label: "Getting Started" },
  { detailKey: "navDetail.registry", groupKey: "configuration", key: "registry", label: "Agents & Tools" },
  { detailKey: "navDetail.capabilities", groupKey: "configuration", key: "capabilities", label: "Tool Capabilities" },
  { detailKey: "navDetail.policies", groupKey: "configuration", key: "policies", label: "Access Policies" },
  { detailKey: "navDetail.routes", groupKey: "configuration", key: "routes", label: "Routing Rules" },
  { detailKey: "navDetail.ai-admin", groupKey: "primary", key: "ai-admin", label: "Permission Changes" },
  { detailKey: "navDetail.access", groupKey: "primary", key: "access", label: "Access Profile" },
  { detailKey: "navDetail.traces", groupKey: "audit", key: "traces", label: "Call Logs" },
  { detailKey: "navDetail.evidence", groupKey: "audit", key: "evidence", label: "Go-Live Evidence" },
  { detailKey: "navDetail.cockpit", groupKey: "audit", key: "cockpit", label: "System Check" },
]

const views: Record<NavKey, ConsoleView> = {
  "getting-started": {
    key: "getting-started",
    primaryPanelKey: "gettingStarted",
    titleKey: "page.gettingStarted",
  },
  cockpit: {
    key: "cockpit",
    primaryPanelKey: "runtimeSignals",
    titleKey: "page.cockpit",
  },
  "ai-admin": {
    key: "ai-admin",
    primaryPanelKey: "aiAdminPermissionWorkbench",
    titleKey: "page.aiAdmin",
  },
  registry: {
    key: "registry",
    primaryPanelKey: "agentRegistry",
    titleKey: "page.registry",
  },
  routes: {
    key: "routes",
    primaryPanelKey: "routeGovernance",
    titleKey: "page.routes",
  },
  policies: {
    key: "policies",
    primaryPanelKey: "policyReview",
    titleKey: "page.policies",
  },
  capabilities: {
    key: "capabilities",
    primaryPanelKey: "mcpCapabilities",
    titleKey: "page.capabilities",
  },
  access: {
    key: "access",
    primaryPanelKey: "tenantAccessProfile",
    titleKey: "page.access",
  },
  traces: {
    key: "traces",
    primaryPanelKey: "auditTraces",
    titleKey: "page.traces",
  },
  evidence: {
    key: "evidence",
    primaryPanelKey: "goLiveAcceptance",
    titleKey: "page.evidence",
  },
}

export function viewForNav(key: string): ConsoleView {
  return views[(key as NavKey) in views ? (key as NavKey) : defaultNavKey]
}

export function isNavKey(key: string): key is NavKey {
  return key in views
}

export function navKeyFromHash(hash: string): NavKey | null {
  const normalized = hash.trim().replace(/^#\/?/, "")
  return isNavKey(normalized) ? normalized : null
}

export function navHashFor(key: NavKey): string {
  return `${navHashPrefix}${key}`
}
