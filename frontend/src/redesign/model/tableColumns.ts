export interface TableColumn {
  key: string;
  labelKey: string;
  // Percentage of the table width; every table sums to 100.
  width: number;
}

// Every redesigned table declares its columns here so widths stay reviewable
// in one place and the Table primitive never guesses a layout.
export const tableColumns = {
  userResources: [
    { key: "resource", labelKey: "rd.col.resource", width: 34 },
    { key: "type", labelKey: "rd.col.type", width: 26 },
    { key: "status", labelKey: "rd.col.status", width: 20 },
    { key: "lastActivity", labelKey: "rd.col.lastActivity", width: 20 },
  ],
  userCapabilities: [
    { key: "capability", labelKey: "rd.col.capability", width: 22 },
    { key: "type", labelKey: "rd.col.type", width: 12 },
    { key: "risk", labelKey: "rd.col.risk", width: 10 },
    { key: "dataScope", labelKey: "rd.col.dataScope", width: 26 },
    { key: "decision", labelKey: "rd.col.decision", width: 14 },
    { key: "approval", labelKey: "rd.col.approval", width: 16 },
  ],
  userTokens: [
    { key: "token", labelKey: "rd.col.maskedToken", width: 30 },
    { key: "status", labelKey: "rd.col.status", width: 18 },
    { key: "expiresAt", labelKey: "rd.col.expiresAt", width: 24 },
    { key: "subject", labelKey: "rd.col.subject", width: 14 },
    { key: "actions", labelKey: "rd.col.actions", width: 14 },
  ],
  adminRecentAudit: [
    { key: "time", labelKey: "rd.col.time", width: 16 },
    { key: "operation", labelKey: "rd.col.operation", width: 20 },
    { key: "resource", labelKey: "rd.col.resource", width: 24 },
    { key: "result", labelKey: "rd.col.result", width: 14 },
    { key: "subject", labelKey: "rd.col.subject", width: 26 },
  ],
  adminTraces: [
    { key: "time", labelKey: "rd.col.time", width: 13 },
    { key: "operation", labelKey: "rd.col.operation", width: 17 },
    { key: "resource", labelKey: "rd.col.resource", width: 26 },
    { key: "subject", labelKey: "rd.col.subject", width: 16 },
    { key: "result", labelKey: "rd.col.result", width: 11 },
    { key: "summary", labelKey: "rd.col.summaryActions", width: 17 },
  ],
  adminTenants: [
    { key: "tenant", labelKey: "rd.col.tenant", width: 20 },
    { key: "workspace", labelKey: "rd.col.workspace", width: 24 },
    { key: "resourceCount", labelKey: "rd.col.resourceCount", width: 12 },
    { key: "grants", labelKey: "rd.col.grants", width: 20 },
    { key: "status", labelKey: "rd.col.status", width: 12 },
    { key: "actions", labelKey: "rd.col.actions", width: 12 },
  ],
  adminRegistry: [
    { key: "resource", labelKey: "rd.col.resource", width: 24 },
    { key: "type", labelKey: "rd.col.type", width: 13 },
    { key: "endpoint", labelKey: "rd.col.endpoint", width: 31 },
    { key: "status", labelKey: "rd.col.status", width: 14 },
    { key: "actions", labelKey: "rd.col.actions", width: 18 },
  ],
  adminCapabilities: [
    { key: "capability", labelKey: "rd.col.capability", width: 22 },
    { key: "type", labelKey: "rd.col.type", width: 13 },
    { key: "risk", labelKey: "rd.col.risk", width: 10 },
    { key: "dataScope", labelKey: "rd.col.dataScope", width: 30 },
    { key: "status", labelKey: "rd.col.status", width: 13 },
    { key: "actions", labelKey: "rd.col.actions", width: 12 },
  ],
  adminRoutes: [
    { key: "rule", labelKey: "rd.col.rule", width: 18 },
    { key: "match", labelKey: "rd.col.match", width: 30 },
    { key: "target", labelKey: "rd.col.target", width: 22 },
    { key: "priority", labelKey: "rd.col.priority", width: 10 },
    { key: "status", labelKey: "rd.col.status", width: 11 },
    { key: "actions", labelKey: "rd.col.actions", width: 9 },
  ],
  adminAdmins: [
    { key: "member", labelKey: "rd.col.member", width: 28 },
    { key: "role", labelKey: "rd.col.role", width: 22 },
    { key: "scope", labelKey: "rd.col.scope", width: 20 },
    { key: "status", labelKey: "rd.col.status", width: 14 },
    { key: "actions", labelKey: "rd.col.actions", width: 16 },
  ],
} as const satisfies Record<string, readonly TableColumn[]>;

export type TableId = keyof typeof tableColumns;

export function tableColumnsFor(tableId: TableId): readonly TableColumn[] {
  return tableColumns[tableId];
}
