import type {
  Agent,
  Capability,
  InstanceAssignment,
  Tenant,
  TenantEntitlement,
  WorkspaceAssignment
} from "./types";

export interface TenantOrganizationWorkspaceSummary {
  workspaceId: string;
  agentCount: number;
  assignmentCount: number;
  callerCount: number;
  targetCount: number;
}

export interface TenantOrganizationNode {
  agentCount: number;
  children: TenantOrganizationNode[];
  depth: number;
  permissionCount: number;
  tenant: Tenant;
  workspaceCount: number;
}

export interface TenantOrganizationSelection {
  activeAgentCount: number;
  allowedPermissionCount: number;
  callerCount: number;
  deniedPermissionCount: number;
  instanceAssignmentCount: number;
  path: Tenant[];
  permissionCount: number;
  selectedWorkspaceId: string;
  targetCount: number;
  tenant: Tenant;
  workspaceAssignmentCount: number;
  workspaces: TenantOrganizationWorkspaceSummary[];
}

export interface TenantOrganizationModel {
  flatNodes: TenantOrganizationNode[];
  nodes: TenantOrganizationNode[];
  selected: TenantOrganizationSelection | null;
  selectedTenantId: string;
  totals: {
    activeTenantCount: number;
    agentCount: number;
    permissionCount: number;
    tenantCount: number;
    workspaceCount: number;
  };
}

export interface BuildTenantOrganizationInput {
  agents: Agent[];
  capabilities: Capability[];
  instanceAssignments: InstanceAssignment[];
  selectedTenantId?: string;
  tenantEntitlements: TenantEntitlement[];
  tenants: Tenant[];
  workspaceAssignments: WorkspaceAssignment[];
}

export function buildTenantOrganizationModel(input: BuildTenantOrganizationInput): TenantOrganizationModel {
  const displayInput = displayTenantOrganizationInput(input);
  const tenantById = new Map(displayInput.tenants.map((tenant) => [tenant.id, tenant]));
  const selectedTenant = selectTenant(displayInput.tenants, displayInput.selectedTenantId);
  const nodes = buildTenantNodes(displayInput.tenants, displayInput.agents, displayInput.tenantEntitlements);
  const flatNodes = flattenTenantNodes(nodes);
  const selected = selectedTenant
    ? buildTenantSelection(selectedTenant, tenantById, displayInput)
    : null;

  return {
    flatNodes,
    nodes,
    selected,
    selectedTenantId: selectedTenant?.id ?? "",
    totals: {
      activeTenantCount: displayInput.tenants.filter((tenant) => tenant.status === "active").length,
      agentCount: displayInput.agents.length,
      permissionCount: displayInput.tenantEntitlements.length,
      tenantCount: displayInput.tenants.length,
      workspaceCount: countUnique([
        ...displayInput.agents.map((agent) => agent.workspaceId),
        ...displayInput.workspaceAssignments.map((assignment) => assignment.workspaceId)
      ])
    }
  };
}

function displayTenantOrganizationInput(input: BuildTenantOrganizationInput): BuildTenantOrganizationInput {
  const tenants = collapseRepeatedDemoTenantBatches(input.tenants);
  if (tenants.length === input.tenants.length) return input;

  const visibleTenantIds = new Set(tenants.map((tenant) => tenant.id));
  return {
    ...input,
    agents: input.agents.filter((agent) => visibleTenantIds.has(agent.tenantId)),
    instanceAssignments: input.instanceAssignments.filter((assignment) => visibleTenantIds.has(assignment.tenantId)),
    tenantEntitlements: input.tenantEntitlements.filter((entitlement) => visibleTenantIds.has(entitlement.tenantId)),
    tenants,
    workspaceAssignments: input.workspaceAssignments.filter((assignment) => visibleTenantIds.has(assignment.tenantId))
  };
}

function collapseRepeatedDemoTenantBatches(tenants: Tenant[]) {
  const latestRoots = latestDemoRootsByGroup(tenants);
  if (latestRoots.size === 0) return tenants;

  const tenantByParentId = new Map<string, Tenant[]>();
  for (const tenant of tenants) {
    if (!tenant.parentTenantId) continue;
    const children = tenantByParentId.get(tenant.parentTenantId) ?? [];
    children.push(tenant);
    tenantByParentId.set(tenant.parentTenantId, children);
  }

  const retainedDemoTenantIds = new Set<string>();
  for (const root of latestRoots.values()) {
    const queue = [root];
    while (queue.length > 0) {
      const tenant = queue.shift();
      if (!tenant || retainedDemoTenantIds.has(tenant.id)) continue;
      retainedDemoTenantIds.add(tenant.id);
      queue.push(...(tenantByParentId.get(tenant.id) ?? []).filter(isDemoTenantBatchMember));
    }
  }

  return tenants.filter((tenant) => !isDemoTenantBatchMember(tenant) || retainedDemoTenantIds.has(tenant.id));
}

function latestDemoRootsByGroup(tenants: Tenant[]) {
  const latestRoots = new Map<string, Tenant>();
  for (const tenant of tenants) {
    const group = demoTenantBatchGroup(tenant);
    if (!group || tenant.parentTenantId || !isDemoTenantRoot(tenant)) continue;
    const current = latestRoots.get(group);
    if (!current || compareTenantFreshness(tenant, current) > 0) {
      latestRoots.set(group, tenant);
    }
  }

  return new Map(
    Array.from(latestRoots).filter(([group]) =>
      tenants.filter((tenant) => demoTenantBatchGroup(tenant) === group && isDemoTenantRoot(tenant)).length > 1
    )
  );
}

function demoTenantBatchGroup(tenant: Tenant) {
  if (/^Permission (Package|Request) Approval (Root|Team|Project)$/.test(tenant.name)) return "permission-approval";
  if (/^Core Journey (Root|Team|Project)$/.test(tenant.name)) return "core-journey";
  return "";
}

function isDemoTenantBatchMember(tenant: Tenant) {
  return Boolean(demoTenantBatchGroup(tenant));
}

function isDemoTenantRoot(tenant: Tenant) {
  return / Root$/.test(tenant.name);
}

function compareTenantFreshness(left: Tenant, right: Tenant) {
  return tenantFreshness(left) - tenantFreshness(right)
    || left.id.localeCompare(right.id);
}

function tenantFreshness(tenant: Tenant) {
  return Date.parse(tenant.updatedAt || tenant.createdAt) || Date.parse(tenant.createdAt) || 0;
}

function selectTenant(tenants: Tenant[], selectedTenantId?: string) {
  if (selectedTenantId) {
    const selected = tenants.find((tenant) => tenant.id === selectedTenantId);
    if (selected) return selected;
  }
  return tenants.find((tenant) => tenant.status === "active" && !tenant.parentTenantId)
    ?? tenants.find((tenant) => tenant.status === "active")
    ?? tenants[0]
    ?? null;
}

function buildTenantNodes(
  tenants: Tenant[],
  agents: Agent[],
  tenantEntitlements: TenantEntitlement[]
): TenantOrganizationNode[] {
  const childrenByParent = new Map<string, Tenant[]>();
  const roots: Tenant[] = [];

  for (const tenant of tenants) {
    if (tenant.parentTenantId && tenants.some((candidate) => candidate.id === tenant.parentTenantId)) {
      const children = childrenByParent.get(tenant.parentTenantId) ?? [];
      children.push(tenant);
      childrenByParent.set(tenant.parentTenantId, children);
    } else {
      roots.push(tenant);
    }
  }

  const buildNode = (tenant: Tenant, depth: number): TenantOrganizationNode => {
    const tenantAgents = agents.filter((agent) => agent.tenantId === tenant.id);
    const children = sortTenants(childrenByParent.get(tenant.id) ?? []).map((child) => buildNode(child, depth + 1));
    return {
      agentCount: tenantAgents.length,
      children,
      depth,
      permissionCount: tenantEntitlements.filter((entitlement) => entitlement.tenantId === tenant.id).length,
      tenant,
      workspaceCount: countUnique(tenantAgents.map((agent) => agent.workspaceId))
    };
  };

  return sortTenants(roots).map((tenant) => buildNode(tenant, 0));
}

function buildTenantSelection(
  tenant: Tenant,
  tenantById: Map<string, Tenant>,
  input: BuildTenantOrganizationInput
): TenantOrganizationSelection {
  const agents = input.agents.filter((agent) => agent.tenantId === tenant.id);
  const activeAgents = agents.filter((agent) => agent.status === "active");
  const entitlements = input.tenantEntitlements.filter((entitlement) => entitlement.tenantId === tenant.id);
  const workspaceAssignments = input.workspaceAssignments.filter((assignment) => assignment.tenantId === tenant.id);
  const instanceAssignments = input.instanceAssignments.filter((assignment) => assignment.tenantId === tenant.id);
  const workspaceIds = uniqueValues([
    ...agents.map((agent) => agent.workspaceId),
    ...workspaceAssignments.map((assignment) => assignment.workspaceId)
  ]);
  const workspaces = workspaceIds.map((workspaceId) => {
    const workspaceAgents = agents.filter((agent) => agent.workspaceId === workspaceId);
    return {
      agentCount: workspaceAgents.length,
      assignmentCount: workspaceAssignments.filter((assignment) => assignment.workspaceId === workspaceId).length,
      callerCount: workspaceAgents.filter((agent) => agent.channelType === "local").length,
      targetCount: workspaceAgents.filter((agent) => agent.channelType !== "local").length,
      workspaceId
    };
  });

  return {
    activeAgentCount: activeAgents.length,
    allowedPermissionCount: entitlements.filter((entitlement) => entitlement.effect === "allow").length,
    callerCount: agents.filter((agent) => agent.channelType === "local").length,
    deniedPermissionCount: entitlements.filter((entitlement) => entitlement.effect === "deny").length,
    instanceAssignmentCount: instanceAssignments.length,
    path: tenantPath(tenant, tenantById),
    permissionCount: entitlements.length,
    selectedWorkspaceId: workspaces[0]?.workspaceId ?? "workspace-sandbox",
    targetCount: agents.filter((agent) => agent.channelType !== "local").length,
    tenant,
    workspaceAssignmentCount: workspaceAssignments.length,
    workspaces
  };
}

function tenantPath(tenant: Tenant, tenantById: Map<string, Tenant>) {
  const path: Tenant[] = [];
  let current: Tenant | undefined = tenant;
  const seen = new Set<string>();

  while (current && !seen.has(current.id)) {
    seen.add(current.id);
    path.unshift(current);
    current = current.parentTenantId ? tenantById.get(current.parentTenantId) : undefined;
  }

  return path;
}

function flattenTenantNodes(nodes: TenantOrganizationNode[]): TenantOrganizationNode[] {
  return nodes.flatMap((node) => [node, ...flattenTenantNodes(node.children)]);
}

function sortTenants(tenants: Tenant[]) {
  return [...tenants].sort((left, right) =>
    left.level - right.level
      || left.name.localeCompare(right.name)
      || left.id.localeCompare(right.id)
  );
}

function uniqueValues(values: string[]) {
  return Array.from(new Set(values.filter(Boolean))).sort((left, right) => left.localeCompare(right));
}

function countUnique(values: string[]) {
  return uniqueValues(values).length;
}
