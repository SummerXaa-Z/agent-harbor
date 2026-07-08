export type NavKey =
  | "getting-started"
  | "ask"
  | "cockpit"
  | "ai-admin"
  | "tenants"
  | "registry"
  | "routes"
  | "policies"
  | "capabilities"
  | "access"
  | "traces"
  | "go-live"
  | "admin-access"

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

export const defaultNavKey: NavKey = "getting-started"
const navHashPrefix = "#"
const navHashAliases: Record<string, NavKey> = {
  "go-live": "go-live",
  evidence: "go-live",
}

export const navGroups: Array<{ key: NavGroupKey; labelKey: string }> = [
  { key: "onboarding", labelKey: "navGroup.onboarding" },
  { key: "primary", labelKey: "navGroup.primary" },
  { key: "audit", labelKey: "navGroup.audit" },
  { key: "configuration", labelKey: "navGroup.configuration" },
]

export const navItems: NavItem[] = [
  { detailKey: "navDetail.getting-started", groupKey: "onboarding", key: "getting-started", label: "Getting Started" },
  { detailKey: "navDetail.ask", groupKey: "primary", key: "ask", label: "Access Query" },
  { detailKey: "navDetail.ai-admin", groupKey: "primary", key: "ai-admin", label: "Permission Changes" },
  { detailKey: "navDetail.access", groupKey: "primary", key: "access", label: "Access Profile" },
  { detailKey: "navDetail.traces", groupKey: "audit", key: "traces", label: "Call Logs" },
  { detailKey: "navDetail.evidence", groupKey: "audit", key: "go-live", label: "Go-Live Status" },
  { detailKey: "navDetail.cockpit", groupKey: "audit", key: "cockpit", label: "System Check" },
  { detailKey: "navDetail.adminAccess", groupKey: "configuration", key: "admin-access", label: "Administrators & Boundaries" },
  { detailKey: "navDetail.tenants", groupKey: "configuration", key: "tenants", label: "Tenants & Organization" },
  { detailKey: "navDetail.registry", groupKey: "configuration", key: "registry", label: "Resource Management" },
  { detailKey: "navDetail.capabilities", groupKey: "configuration", key: "capabilities", label: "Tool Capabilities" },
  { detailKey: "navDetail.policies", groupKey: "configuration", key: "policies", label: "Access Policies" },
  { detailKey: "navDetail.routes", groupKey: "configuration", key: "routes", label: "Routing Rules" },
]

const views: Record<NavKey, ConsoleView> = {
  "getting-started": {
    key: "getting-started",
    primaryPanelKey: "gettingStarted",
    titleKey: "page.gettingStarted",
  },
  ask: {
    key: "ask",
    primaryPanelKey: "askAccess",
    titleKey: "page.ask",
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
  tenants: {
    key: "tenants",
    primaryPanelKey: "tenantOrganization",
    titleKey: "page.tenants",
  },
  registry: {
    key: "registry",
    primaryPanelKey: "resourceLifecycle",
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
  "go-live": {
    key: "go-live",
    primaryPanelKey: "goLiveAcceptance",
    titleKey: "page.evidence",
  },
  "admin-access": {
    key: "admin-access",
    primaryPanelKey: "adminAccess",
    titleKey: "page.adminAccess",
  },
}

export function viewForNav(key: string): ConsoleView {
  const navKey = navKeyFromHash(key) ?? defaultNavKey
  return views[navKey]
}

export function isNavKey(key: string): key is NavKey {
  return key in views
}

export function navKeyFromHash(hash: string): NavKey | null {
  const normalized = hash.trim().replace(/^#\/?/, "")
  if (normalized in navHashAliases) {
    return navHashAliases[normalized]
  }
  return isNavKey(normalized) ? normalized : null
}

export function navHashFor(key: NavKey): string {
  if (key === "go-live") {
    return `${navHashPrefix}go-live`
  }
  return `${navHashPrefix}${key}`
}

export function canonicalNavHashFromHash(hash: string): string | null {
  const key = navKeyFromHash(hash)
  return key ? navHashFor(key) : null
}
