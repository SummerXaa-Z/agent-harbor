export type NavKey =
  | "cockpit"
  | "ai-admin"
  | "registry"
  | "routes"
  | "policies"
  | "capabilities"
  | "access"
  | "traces"
  | "evidence"

export type NavGroupKey = "primary" | "audit" | "configuration"

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

export const navGroups: Array<{ key: NavGroupKey; labelKey: string }> = [
  { key: "primary", labelKey: "navGroup.primary" },
  { key: "audit", labelKey: "navGroup.audit" },
  { key: "configuration", labelKey: "navGroup.configuration" },
]

export const navItems: NavItem[] = [
  { detailKey: "navDetail.ai-admin", groupKey: "primary", key: "ai-admin", label: "Permission Changes" },
  { detailKey: "navDetail.access", groupKey: "primary", key: "access", label: "Access Profile" },
  { detailKey: "navDetail.evidence", groupKey: "primary", key: "evidence", label: "Go-Live Evidence" },
  { detailKey: "navDetail.traces", groupKey: "audit", key: "traces", label: "Call Logs" },
  { detailKey: "navDetail.cockpit", groupKey: "audit", key: "cockpit", label: "System Check" },
  { detailKey: "navDetail.registry", groupKey: "configuration", key: "registry", label: "Agents & Tools" },
  { detailKey: "navDetail.capabilities", groupKey: "configuration", key: "capabilities", label: "Tool Capabilities" },
  { detailKey: "navDetail.policies", groupKey: "configuration", key: "policies", label: "Access Policies" },
  { detailKey: "navDetail.routes", groupKey: "configuration", key: "routes", label: "Routing Rules" },
]

const views: Record<NavKey, ConsoleView> = {
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
    primaryPanelKey: "evidenceRuns",
    titleKey: "page.evidence",
  },
}

export function viewForNav(key: string): ConsoleView {
  return views[(key as NavKey) in views ? (key as NavKey) : defaultNavKey]
}
