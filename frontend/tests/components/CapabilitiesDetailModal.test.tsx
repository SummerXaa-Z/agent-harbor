import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { OverlayRootContext } from "../../src/redesign/ui/Modal";
import { CapabilitiesDetailModal } from "../../src/redesign/views/admin/CapabilitiesDetailModal";
import { capability } from "./fixtures";
import { stubT, TestI18n } from "./testI18n";

type ModalProps = Parameters<typeof CapabilitiesDetailModal>[0];

function modalProps(overrides: Partial<ModalProps> = {}): ModalProps {
  return {
    capability: capability(),
    domainOptions: ["billing", "operations"],
    acting: false,
    t: stubT,
    onClose: vi.fn(),
    onApprove: vi.fn(),
    onSave: vi.fn(),
    ...overrides,
  };
}

function renderModal(overrides: Partial<ModalProps> = {}) {
  const props = modalProps(overrides);
  render(
    <TestI18n>
      <OverlayRootContext.Provider value={document.body}>
        <CapabilitiesDetailModal {...props} />
      </OverlayRootContext.Provider>
    </TestI18n>,
  );
  return props;
}

describe("CapabilitiesDetailModal", () => {
  it("seeds drafts from the capability and passes them to save untouched", () => {
    const { onSave } = renderModal({ capability: capability({ riskLevel: "medium", dataDomains: ["orders"] }) });
    fireEvent.click(screen.getByText("rd.cap.save"));
    expect(onSave).toHaveBeenCalledWith("medium", "orders");
  });

  it("falls back to the first data scope domain when data domains are unset", () => {
    const scoped = capability({ dataDomains: undefined, dataScopes: [{ dataDomain: "operations" }] });
    const { onSave } = renderModal({ capability: scoped });
    fireEvent.click(screen.getByText("rd.cap.save"));
    expect(onSave).toHaveBeenCalledWith("low", "operations");
  });

  it("saves edited risk and domain drafts", () => {
    const { onSave } = renderModal({ capability: capability({ riskLevel: "low", dataDomains: ["orders"] }) });
    fireEvent.change(screen.getByLabelText("rd.col.risk"), { target: { value: "critical" } });
    fireEvent.change(screen.getByLabelText("rd.cap.domain"), { target: { value: "billing" } });
    fireEvent.click(screen.getByText("rd.cap.save"));
    expect(onSave).toHaveBeenCalledWith("critical", "billing");
  });

  it("offers approve only while the capability awaits review", () => {
    renderModal({ capability: capability() });
    expect(screen.queryByText("rd.cap.approve")).toBeNull();

    const pending = capability({ discoveryStatus: "pending_review" });
    const props = renderModal({ capability: pending });
    fireEvent.click(screen.getByText("rd.cap.approve"));
    expect(props.onApprove).toHaveBeenCalledWith(pending);
  });

  it("keeps edited drafts when the parent re-renders with a refreshed capability object", () => {
    // After save the view reloads and consoleData replaces every object; the
    // modal keeps its own draft state, so an in-progress edit survives.
    const props = modalProps({ capability: capability({ riskLevel: "high", dataDomains: ["orders"] }) });
    const view = render(
      <TestI18n>
        <OverlayRootContext.Provider value={document.body}>
          <CapabilitiesDetailModal {...props} />
        </OverlayRootContext.Provider>
      </TestI18n>,
    );
    fireEvent.change(screen.getByLabelText("rd.col.risk"), { target: { value: "low" } });
    const refreshed = capability({ riskLevel: "critical", dataDomains: ["billing"] });
    view.rerender(
      <TestI18n>
        <OverlayRootContext.Provider value={document.body}>
          <CapabilitiesDetailModal {...props} capability={refreshed} />
        </OverlayRootContext.Provider>
      </TestI18n>,
    );
    fireEvent.click(screen.getByText("rd.cap.save"));
    expect(props.onSave).toHaveBeenCalledWith("low", "orders");
  });
});
