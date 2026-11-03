import { useState } from "react";
import { Banner } from "../../ui/Banner";
import { Button } from "../../ui/Button";
import { Chip } from "../../ui/Chip";
import { Modal } from "../../ui/Modal";
import type { Agent, Capability, InstanceAssignment, TenantEntitlement, WorkspaceAssignment } from "../../../types";
import type { useRedesignI18n } from "../../hooks/useRedesignI18n";

export type CapabilityChainLayer = "entitlement" | "workspace" | "instance";

// The modal is fully derived from consoleData, so a successful disable plus a
// data reload re-renders the layers with their new statuses in place. Layer
// failures surface as the string onDisableLayer resolves to.
export function CapabilitiesChainDetailModal({
  entitlement,
  workspaces,
  instancesOf,
  capabilities,
  agents,
  acting,
  t,
  onClose,
  onDisableLayer
}: {
  entitlement: TenantEntitlement;
  workspaces: readonly WorkspaceAssignment[];
  instancesOf: (workspaceAssignmentId: string) => readonly InstanceAssignment[];
  capabilities: readonly Capability[];
  agents: readonly Agent[];
  acting: boolean;
  t: ReturnType<typeof useRedesignI18n>["t"];
  onClose: () => void;
  onDisableLayer: (layer: CapabilityChainLayer, id: string) => Promise<string | null>;
}) {
  const [actionError, setActionError] = useState("");

  async function disableLayer(layer: CapabilityChainLayer, id: string) {
    setActionError((await onDisableLayer(layer, id)) ?? "");
  }

  function chainStatusChip(status: "enabled" | "disabled") {
    return (
      <Chip tone={status === "enabled" ? "success" : "neutral"}>
        {t(status === "enabled" ? "rd.cap.chainStatusEnabled" : "rd.cap.chainStatusDisabled")}
      </Chip>
    );
  }

  return (
    <Modal
      footer={
        <Button onClick={onClose} variant="primary">{t("rd.common.close")}</Button>
      }
      onClose={onClose}
      open
      size="wide"
      title={t("rd.cap.chainDetailTitle")}
    >
      <div className="stack">
        <p className="hint">{t("rd.cap.chainDetailHelp")}</p>
        {actionError ? <Banner desc={actionError} title={t("rd.error.other.title")} tone="danger" /> : null}
        <div className="cap-list">
          <div className="cap-row">
            <div className="cap-main">
              <div className="cap-name">{t("rd.cap.chainEntitlement")}</div>
              <div className="cap-sub mono">
                {capabilities.find((capability) => capability.id === entitlement.capabilityId)?.displayName ?? entitlement.capabilityId}
                {" · "}
                {entitlement.tenantId}
              </div>
            </div>
            <div className="apr-button-row">
              {chainStatusChip(entitlement.status)}
              {entitlement.status === "enabled" ? (
                <Button
                  disabled={acting}
                  onClick={() => void disableLayer("entitlement", entitlement.id)}
                  size="sm"
                  variant="danger-ghost"
                >
                  {t("rd.cap.chainDisable")}
                </Button>
              ) : null}
            </div>
          </div>
          {workspaces.length === 0 ? (
            <p className="muted small">{t("rd.cap.chainNoWorkspaces")}</p>
          ) : (
            workspaces.map((workspace) => {
              const instances = instancesOf(workspace.id);
              return (
                <div className="chain-group" key={workspace.id}>
                  <div className="cap-row">
                    <div className="cap-main">
                      <div className="cap-name">{t("rd.cap.chainWorkspace")}</div>
                      <div className="cap-sub mono">{workspace.workspaceId}</div>
                    </div>
                    <div className="apr-button-row">
                      {chainStatusChip(workspace.status)}
                      {workspace.status === "enabled" ? (
                        <Button
                          disabled={acting}
                          onClick={() => void disableLayer("workspace", workspace.id)}
                          size="sm"
                          variant="danger-ghost"
                        >
                          {t("rd.cap.chainDisable")}
                        </Button>
                      ) : null}
                    </div>
                  </div>
                  {instances.length === 0 ? (
                    <p className="muted small chain-indent">{t("rd.cap.chainNoCallers")}</p>
                  ) : (
                    <div className="cap-list chain-indent">
                      {instances.map((instance) => (
                        <div className="cap-row" key={instance.id}>
                          <div className="cap-main">
                            <div className="cap-name">{t("rd.cap.chainCaller")}</div>
                            <div className="cap-sub mono">
                              {agents.find((agent) => agent.id === instance.callerInstanceId)?.name ?? instance.callerInstanceId}
                              {instance.subjectSelector ? ` · ${instance.subjectSelector}` : ""}
                            </div>
                          </div>
                          <div className="apr-button-row">
                            {chainStatusChip(instance.status)}
                            {instance.status === "enabled" ? (
                              <Button
                                disabled={acting}
                                onClick={() => void disableLayer("instance", instance.id)}
                                size="sm"
                                variant="danger-ghost"
                              >
                                {t("rd.cap.chainDisable")}
                              </Button>
                            ) : null}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              );
            })
          )}
        </div>
      </div>
    </Modal>
  );
}
