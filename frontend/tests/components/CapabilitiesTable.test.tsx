import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CapabilitiesTable } from "../../src/redesign/views/admin/CapabilitiesTable";
import type { Capability } from "../../src/types";
import { capability } from "./fixtures";
import { stubT, TestI18n } from "./testI18n";

const pending = capability({
  id: "cap_pending",
  key: "crm.exportAll",
  displayName: "Export All",
  riskLevel: "critical",
  discoveryStatus: "pending_review",
});
const approved = capability({
  id: "cap_approved",
  key: "crm.lookup",
  displayName: "Lookup",
  riskLevel: "low",
  discoveryStatus: "approved",
  dataDomains: ["orders"],
});

function renderTable(capabilities: readonly Capability[], acting = false) {
  const onDetail = vi.fn();
  const onApprove = vi.fn();
  render(
    <TestI18n>
      <CapabilitiesTable
        acting={acting}
        capabilities={capabilities}
        onApprove={onApprove}
        onDetail={onDetail}
        t={stubT}
      />
    </TestI18n>,
  );
  return { onDetail, onApprove };
}

describe("CapabilitiesTable", () => {
  it("renders one row per capability with key, risk, domain and status", () => {
    renderTable([pending, approved]);
    expect(screen.getByText("Export All")).toBeTruthy();
    expect(screen.getByText("crm.exportAll")).toBeTruthy();
    expect(screen.getByText("rd.risk.critical")).toBeTruthy();
    expect(screen.getByText("status.capabilityPendingReview")).toBeTruthy();
    expect(screen.getByText("status.capabilityApproved")).toBeTruthy();
    expect(screen.getByText("orders")).toBeTruthy();
  });

  it("offers approve only on pending rows and reports the row on click", () => {
    const { onApprove } = renderTable([pending, approved]);
    const approveButtons = screen.getAllByText("rd.cap.approve");
    expect(approveButtons).toHaveLength(1);
    fireEvent.click(approveButtons[0]);
    expect(onApprove).toHaveBeenCalledWith(pending);
  });

  it("offers detail on every row and disables approve while acting", () => {
    const { onDetail } = renderTable([pending, approved], true);
    expect(screen.getAllByText("rd.cap.detail")).toHaveLength(2);
    fireEvent.click(screen.getAllByText("rd.cap.detail")[0]);
    expect(onDetail).toHaveBeenCalledWith(pending);
    const approve = screen.getByText("rd.cap.approve").closest("button");
    expect(approve?.hasAttribute("disabled")).toBe(true);
  });

  it("renders the empty state when the target has no capabilities", () => {
    renderTable([]);
    expect(screen.getByText("rd.cap.empty")).toBeTruthy();
    expect(screen.queryByText("rd.cap.detail")).toBeNull();
  });
});
