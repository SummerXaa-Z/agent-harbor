import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { OverlayRootContext } from "../../src/redesign/ui/Modal";
import { TracePolicyDetail } from "../../src/redesign/views/admin/TracePolicyDetail";
import type { TraceEvent } from "../../src/types";
import { stubT, TestI18n } from "./testI18n";

function renderDetail(trace: Partial<TraceEvent>) {
  const full: TraceEvent = {
    id: "trc_1",
    targetAgentId: "agt_crm",
    routeType: "mcp",
    decision: "denied",
    createdAt: "2026-11-05T00:00:00Z",
    ...trace,
  } as TraceEvent;
  const view = render(
    <TestI18n>
      <OverlayRootContext.Provider value={document.body}>
        <TracePolicyDetail trace={full} t={stubT} />
      </OverlayRootContext.Provider>
    </TestI18n>,
  );
  return view;
}

describe("TracePolicyDetail", () => {
  it("renders the structured policy explanation with red risk chips", () => {
    renderDetail({
      reason: "prod database is read-only",
      ruleRef: "harbor.yaml:13",
      argsDigest: "sha256:9f3e",
      riskFlags: ["delete"],
    });
    expect(screen.getByText("rd.audit.policySection")).toBeTruthy();
    expect(screen.getByText("prod database is read-only")).toBeTruthy();
    expect(screen.getByText("harbor.yaml:13")).toBeTruthy();
    expect(screen.getByText(/rd\.audit\.policyArgsNote/)).toBeTruthy();
    expect(screen.getByText("delete")).toBeTruthy();
  });

  it("shows wrapper source and a failed execution separately from permission", () => {
    renderDetail({ source: "wrap", outcome: "error", decision: "allowed", toolName: "search_notes" });
    expect(screen.getByText("search_notes")).toBeTruthy();
    expect(screen.getByText("rd.audit.source.wrap")).toBeTruthy();
    expect(screen.getByText("rd.audit.result.failed")).toBeTruthy();
  });

  it("renders nothing for legacy traces without policy fields", () => {
    // Pre-policy allowed traces carry no reason/ruleRef/digest at all.
    const { container } = renderDetail({});
    expect(container.textContent).not.toContain("rd.audit.policySection");
  });
});
