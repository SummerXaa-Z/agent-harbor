import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CapabilityGrantRow } from "../../src/redesign/model/capabilityCatalog";
import { CapabilitiesGrantList } from "../../src/redesign/views/admin/CapabilitiesGrantList";
import { capability } from "./fixtures";
import { stubT, TestI18n } from "./testI18n";

const rows: CapabilityGrantRow[] = [
  {
    callerCount: 2,
    capabilityId: "cap_lookup",
    effect: "allow",
    entitlementId: "ent_1",
    status: "enabled",
    subjectSelectors: ["user:support-*"],
    tenantId: "tenant_acme",
    workspaceCount: 3,
  },
  {
    callerCount: 0,
    capabilityId: "cap_missing",
    effect: "deny",
    entitlementId: "ent_2",
    status: "enabled",
    subjectSelectors: [],
    tenantId: "tenant_beta",
    workspaceCount: 1,
  },
];

const capabilities = [capability({ id: "cap_lookup", displayName: "Lookup Customer" })];

function renderList(grants: readonly CapabilityGrantRow[], onManage = vi.fn()) {
  render(
    <TestI18n>
      <CapabilitiesGrantList capabilities={capabilities} grants={grants} onManage={onManage} t={stubT} />
    </TestI18n>,
  );
  return { onManage };
}

describe("CapabilitiesGrantList", () => {
  it("renders one row per grant, falling back to the capability id when unknown", () => {
    renderList(rows);
    expect(screen.getByText("Lookup Customer")).toBeTruthy();
    expect(screen.getByText("cap_missing")).toBeTruthy();
    expect(screen.getByText("rd.decision.allowed")).toBeTruthy();
    expect(screen.getByText("rd.decision.blocked")).toBeTruthy();
  });

  it("shows tenant, workspace and caller counts with subject selectors per row", () => {
    renderList(rows);
    const row = screen.getAllByText("rd.cap.manageChain")[0].closest(".cap-row");
    expect(row?.textContent).toContain("tenant_acme");
    expect(row?.textContent).toContain("user:support-*");
    const denyRow = screen.getAllByText("rd.cap.manageChain")[1].closest(".cap-row");
    expect(denyRow?.textContent).toContain("tenant_beta");
  });

  it("reports the entitlement id when manage is clicked", () => {
    const { onManage } = renderList(rows, vi.fn());
    fireEvent.click(screen.getAllByText("rd.cap.manageChain")[1]);
    expect(onManage).toHaveBeenCalledWith("ent_2");
  });

  it("renders the empty state when there are no grants", () => {
    renderList([]);
    expect(screen.getByText("rd.cap.grantsEmpty")).toBeTruthy();
    expect(screen.queryByText("rd.cap.manageChain")).toBeNull();
  });
});
