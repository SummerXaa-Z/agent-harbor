export interface SystemInfo {
  name: string
  apiVersion: string
  authRequired: boolean
  capabilities: string[]
  managementMcpToolCatalog: {
    metadataVersion: number
    requiredMetadata: string[]
  }
}

export const requiredConsoleCapabilities = [
  'permission_package_approval_requests',
  'permission_package_approval_withdraw',
  'permission_package_apply_preflight',
  'permission_package_applications',
  'permission_package_application_health',
  'permission_package_application_impact',
  'permission_package_production_readiness',
  'permission_package_consumed_approval_recovery',
  'management_mcp_tools_metadata_v2',
]

export const requiredManagementMcpToolCatalogMetadata = ['safety', 'access', 'lifecycle']

export function missingConsoleCapabilities(systemInfo: Pick<SystemInfo, 'capabilities'>): string[] {
  const available = new Set(Array.isArray(systemInfo.capabilities) ? systemInfo.capabilities : [])
  return requiredConsoleCapabilities.filter((capability) => !available.has(capability))
}

export function systemInfoContractIssues(systemInfo: Partial<SystemInfo>): string[] {
  const issues = missingConsoleCapabilities({
    capabilities: Array.isArray(systemInfo.capabilities) ? systemInfo.capabilities : [],
  })

  const catalog = systemInfo.managementMcpToolCatalog
  if (catalog?.metadataVersion !== 2) {
    issues.push('managementMcpToolCatalog.metadataVersion')
  }

  const catalogMetadata = new Set(Array.isArray(catalog?.requiredMetadata) ? catalog.requiredMetadata : [])
  for (const field of requiredManagementMcpToolCatalogMetadata) {
    if (!catalogMetadata.has(field)) {
      issues.push(`managementMcpToolCatalog.requiredMetadata.${field}`)
    }
  }

  return issues
}
