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

export interface NavItem {
  key: NavKey
  label: string
}

export interface ConsoleView {
  key: NavKey
  primaryPanelKey: string
  titleKey: string
}

export const defaultNavKey: NavKey = "ai-admin"

export const navItems: NavItem[] = [
  { key: "ai-admin", label: "Permission Packages" },
  { key: "access", label: "Access" },
  { key: "traces", label: "Traces" },
  { key: "evidence", label: "Evidence" },
  { key: "cockpit", label: "Self-Check" },
  { key: "registry", label: "Registry" },
  { key: "routes", label: "Routes" },
  { key: "policies", label: "Policies" },
  { key: "capabilities", label: "Capabilities" },
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
