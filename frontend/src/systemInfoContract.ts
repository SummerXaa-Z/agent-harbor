export interface SystemInfo {
  name: string
  apiVersion: string
  authRequired: boolean
  capabilities: string[]
  managementMcpToolCatalog: {
    metadataVersion: number
    requiredMetadata: string[]
    catalogDigest: string
    toolCount: number
    confirmationRequiredTools: number
    toolsWithConfirmationSchema: number
  }
}

export const requiredConsoleCapabilities = [
  'capability_data_domain_classification_v1',
  'permission_package_requested_capability_v1',
  'permission_package_approval_requests',
  'permission_package_approval_withdraw',
  'permission_package_apply_preflight',
  'permission_package_applications',
  'permission_package_application_health',
  'permission_package_application_impact',
  'permission_package_production_readiness',
  'permission_package_access_handoff_v1',
  'permission_package_access_handoff_tokens_v1',
  'permission_package_consumed_approval_recovery',
  'management_mcp_tools_metadata_v4',
]

export const requiredManagementMcpToolCatalogMetadata = ['safety', 'access', 'lifecycle', 'execution']

export function isManagementMcpToolCatalogContractIssue(issue: string): boolean {
  return (
    issue === 'managementMcpToolCatalog.metadataVersion' ||
    issue === 'managementMcpToolCatalog.catalogDigest' ||
    issue === 'managementMcpToolCatalog.toolCount' ||
    issue === 'managementMcpToolCatalog.confirmationRequiredTools' ||
    issue === 'managementMcpToolCatalog.toolsWithConfirmationSchema' ||
    issue.startsWith('managementMcpToolCatalog.requiredMetadata.')
  )
}

export function missingConsoleCapabilities(systemInfo: Pick<SystemInfo, 'capabilities'>): string[] {
  const available = new Set(Array.isArray(systemInfo.capabilities) ? systemInfo.capabilities : [])
  return requiredConsoleCapabilities.filter((capability) => !available.has(capability))
}

export function systemInfoContractIssues(systemInfo: Partial<SystemInfo>): string[] {
  const issues = missingConsoleCapabilities({
    capabilities: Array.isArray(systemInfo.capabilities) ? systemInfo.capabilities : [],
  })

  const catalog = systemInfo.managementMcpToolCatalog
  if (catalog?.metadataVersion !== 4) {
    issues.push('managementMcpToolCatalog.metadataVersion')
  }

  const catalogMetadata = new Set(Array.isArray(catalog?.requiredMetadata) ? catalog.requiredMetadata : [])
  for (const field of requiredManagementMcpToolCatalogMetadata) {
    if (!catalogMetadata.has(field)) {
      issues.push(`managementMcpToolCatalog.requiredMetadata.${field}`)
    }
  }

  if (!Number.isFinite(catalog?.toolCount) || Number(catalog?.toolCount) <= 0) {
    issues.push('managementMcpToolCatalog.toolCount')
  }
  if (!/^[a-f0-9]{64}$/.test(catalog?.catalogDigest ?? '')) {
    issues.push('managementMcpToolCatalog.catalogDigest')
  }
  if (
    !Number.isFinite(catalog?.confirmationRequiredTools) ||
    Number(catalog?.confirmationRequiredTools) <= 0
  ) {
    issues.push('managementMcpToolCatalog.confirmationRequiredTools')
  }
  if (
    !Number.isFinite(catalog?.toolsWithConfirmationSchema) ||
    catalog?.toolsWithConfirmationSchema !== catalog?.confirmationRequiredTools
  ) {
    issues.push('managementMcpToolCatalog.toolsWithConfirmationSchema')
  }

  return issues
}
