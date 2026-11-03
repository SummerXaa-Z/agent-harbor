import { useEffect, useMemo, useState } from "react";
import { RefreshCw } from "lucide-react";
import {
  createInstanceAssignment,
  createTenantEntitlement,
  createWorkspaceAssignment,
  disableInstanceAssignment,
  disableTenantEntitlement,
  disableWorkspaceAssignment,
  refreshTargetCapabilities,
  updateCapability,
} from "../../../api";
import {
  capabilityGrantBlockerKey,
  normalizeCapabilityGrantForm,
  validateCapabilityGrantChain,
  type CapabilityGrantForm,
} from "../../../capabilityGrantChain";
import { tx } from "../../../localizedMessages";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { usePermissionCatalog } from "../../hooks/usePermissionCatalog";
import { apiErrorPresentation } from "../../model/apiErrorCategory";
import { capabilityDomainSegments, capabilityGrantRows } from "../../model/capabilityCatalog";
import { unclassifiedCapabilities } from "../../model/capabilityGovernance";
import { adminHash } from "../../router";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Banner, Notice } from "../../ui/Banner";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip } from "../../ui/Chip";
import { Field } from "../../ui/Field";
import { EmptyState } from "../../ui/StateViews";
import { useToast } from "../../ui/Toast";
import type { Capability } from "../../../types";
import type { AdminViewProps } from "./adminViewProps";
import { CapabilitiesChainDetailModal, type CapabilityChainLayer } from "./CapabilitiesChainDetailModal";
import { CapabilitiesChainFormModal } from "./CapabilitiesChainFormModal";
import { CapabilitiesDetailModal } from "./CapabilitiesDetailModal";
import { CapabilitiesDomainDonut } from "./CapabilitiesDomainDonut";
import { CapabilitiesGrantList } from "./CapabilitiesGrantList";
import { CapabilitiesRefreshModal } from "./CapabilitiesRefreshModal";
import { CapabilitiesTable } from "./CapabilitiesTable";

export function CapabilitiesView({ data, onRetry, params }: AdminViewProps) {
  const { language, t } = useRedesignI18n();
  const toast = useToast();
  const consoleData = data.data;
  const live = data.status === "live";
  const catalog = usePermissionCatalog(live);

  const mcpTargets = useMemo(
    () => (consoleData?.agents ?? []).filter((agent) => agent.channelType === "mcp" && agent.status !== "disabled"),
    [consoleData?.agents]
  );
  // The deep-link target can arrive before the agent list loads, so the URL
  // param is resolved at render time against whatever is loaded: a param no
  // loaded target matches surfaces a notice instead of silently resetting to
  // the first target (round 4, #27). Everything is derived — effect-ordered
  // state would race the hash sync below on the commit where data lands.
  const [selectedTargetId, setSelectedTargetId] = useState("");
  const paramsTarget = params.target ?? "";
  const targetsLoaded = mcpTargets.length > 0;
  const paramsTargetValid = targetsLoaded && mcpTargets.some((agent) => agent.id === paramsTarget);
  const missedTarget =
    targetsLoaded && paramsTarget !== "" && !paramsTargetValid && selectedTargetId === "" ? paramsTarget : "";
  const targetId = selectedTargetId || (paramsTargetValid ? paramsTarget : "") || mcpTargets[0]?.id || "";

  // On a miss the URL keeps the stale target: rewriting it would re-key the
  // view (any re-render re-reads location.hash) and wipe the notice that
  // explains the fallback. Picking a target clears the miss and syncs again.
  useEffect(() => {
    if (!targetId || missedTarget) return;
    const next = adminHash("capabilities", { target: targetId });
    if (window.location.hash !== next) {
      window.history.replaceState(null, "", `${window.location.pathname}${window.location.search}${next}`);
    }
  }, [targetId, missedTarget]);

  const capabilities = consoleData?.capabilities ?? [];
  const targetCapabilities = capabilities.filter((capability) => capability.targetId === targetId);
  const segments = capabilityDomainSegments(targetCapabilities);
  // Approved but unclassified: template matching needs a data domain, so these
  // stay blocked in every permission preview until governance fills one in.
  const unclassifiedApproved = useMemo(
    () => unclassifiedCapabilities(targetCapabilities.filter((capability) => capability.discoveryStatus === "approved")),
    [targetCapabilities],
  );
  const grants = useMemo(
    () =>
      capabilityGrantRows({
        entitlements: (consoleData?.tenantEntitlements ?? []).filter((item) => item.targetId === targetId),
        instanceAssignments: consoleData?.instanceAssignments ?? [],
        workspaceAssignments: consoleData?.workspaceAssignments ?? []
      }),
    [consoleData, targetId]
  );

  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState("");
  const [refreshOpen, setRefreshOpen] = useState(false);
  const [detail, setDetail] = useState<Capability | null>(null);
  const [chainOpen, setChainOpen] = useState(false);
  const [form, setForm] = useState<CapabilityGrantForm>(() => ({
    callerInstanceId: "",
    capabilityId: "",
    subjectSelector: "user:support-*",
    targetId: "",
    tenantId: "",
    workspaceId: ""
  }));
  const [chainError, setChainError] = useState("");
  // Grant-chain management (backend issue 413 console wiring): the detail
  // modal is fully derived from consoleData, so a successful disable plus
  // onRetry() re-renders the layers with their new statuses in place.
  const [chainDetailId, setChainDetailId] = useState("");

  const agents = consoleData?.agents ?? [];
  const tenants = consoleData?.tenants ?? [];
  const selectedCapability = capabilities.find((capability) => capability.id === form.capabilityId) ?? null;
  const blockerKey = capabilityGrantBlockerKey(form, selectedCapability);

  function openChain() {
    if (!consoleData) return;
    setForm((current) => {
      const seeded = normalizeCapabilityGrantForm({ ...current, targetId }, consoleData);
      if (seeded.tenantId !== "" && seeded.workspaceId !== "") return seeded;
      // The shared normalizer never invents a scope; in this view the natural
      // default is the pre-filled caller's own tenant and workspace.
      const caller = consoleData.agents.find((agent) => agent.id === seeded.callerInstanceId);
      return {
        ...seeded,
        tenantId: seeded.tenantId || caller?.tenantId || "",
        workspaceId: seeded.workspaceId || caller?.workspaceId || ""
      };
    });
    setChainError("");
    setChainOpen(true);
  }

  async function runRefresh() {
    if (!targetId) return;
    setActing(true);
    setActionError("");
    try {
      const refreshed = await refreshTargetCapabilities(targetId);
      toast(tx(t, "rd.cap.refreshed", { count: refreshed.length }));
      setRefreshOpen(false);
      await onRetry();
    } catch (error) {
      const presentation = apiErrorPresentation(t, language, error, "rd.cap.refreshFailed");
      setActionError(presentation.detail || presentation.next);
      setRefreshOpen(false);
    } finally {
      setActing(false);
    }
  }

  async function approveCapability(capability: Capability) {
    setActing(true);
    setActionError("");
    try {
      await updateCapability(capability.id, { discoveryStatus: "approved" });
      toast(
        (capability.dataDomains?.length ?? 0) === 0
          ? tx(t, "rd.cap.approvedNeedsDomain", { name: capability.displayName })
          : tx(t, "rd.cap.approved", { name: capability.displayName })
      );
      setDetail(null);
      await onRetry();
    } catch (error) {
      const presentation = apiErrorPresentation(t, language, error, "rd.cap.failed");
      setActionError(presentation.detail || presentation.next);
    } finally {
      setActing(false);
    }
  }

  async function saveDetail(riskLevel: Capability["riskLevel"], dataDomain: string) {
    if (!detail) return;
    setActing(true);
    setActionError("");
    try {
      await updateCapability(detail.id, {
        dataDomains: dataDomain.trim() ? [dataDomain.trim()] : undefined,
        riskLevel
      });
      toast(tx(t, "rd.cap.governanceSaved", { name: detail.displayName }));
      setDetail(null);
      await onRetry();
    } catch (error) {
      const presentation = apiErrorPresentation(t, language, error, "rd.cap.failed");
      setActionError(presentation.detail || presentation.next);
    } finally {
      setActing(false);
    }
  }

  async function submitChain() {
    const outcome = validateCapabilityGrantChain(form, capabilities);
    if ("messageKey" in outcome) {
      setChainError(t(outcome.messageKey));
      return;
    }
    const { capability, dataScopes, tenantId, workspaceId, callerInstanceId, subjectSelector } = outcome.plan;
    setActing(true);
    setChainError("");
    try {
      const entitlement = await createTenantEntitlement(
        {
          capabilityId: capability.id,
          dataScopes,
          effect: "allow",
          priority: 50,
          status: "enabled",
          targetId: capability.targetId,
          tenantId
        }
      );
      const workspaceAssignment = await createWorkspaceAssignment(
        {
          dataScopes,
          effect: "allow",
          status: "enabled",
          tenantEntitlementId: entitlement.id,
          workspaceId
        }
      );
      await createInstanceAssignment(
        {
          callerInstanceId,
          dataScopes,
          effect: "allow",
          status: "enabled",
          subjectSelector,
          workspaceAssignmentId: workspaceAssignment.id
        }
      );
      toast(t("rd.cap.chainCreated"));
      setChainOpen(false);
      await onRetry();
    } catch (error) {
      const presentation = apiErrorPresentation(t, language, error, "rd.cap.chainFailed");
      setChainError(presentation.detail || presentation.next);
    } finally {
      setActing(false);
    }
  }

  async function disableChainLayer(layer: CapabilityChainLayer, id: string): Promise<string | null> {
    setActing(true);
    try {
      if (layer === "entitlement") await disableTenantEntitlement(id);
      else if (layer === "workspace") await disableWorkspaceAssignment(id);
      else await disableInstanceAssignment(id);
      toast(t("rd.cap.chainDisabled"));
      await onRetry();
      return null;
    } catch (error) {
      const code = error && typeof error === "object" && "code" in error
        ? String((error as { code?: unknown }).code)
        : "";
      if (code === "GRANT_CHAIN_CHILDREN_ACTIVE") {
        return t(layer === "entitlement" ? "rd.cap.chainChildrenEntitlement" : "rd.cap.chainChildrenWorkspace");
      }
      const presentation = apiErrorPresentation(t, language, error, "rd.cap.chainFailed");
      return presentation.detail || presentation.next;
    } finally {
      setActing(false);
    }
  }

  const chainDetail = chainDetailId
    ? (consoleData?.tenantEntitlements ?? []).find((entitlement) => entitlement.id === chainDetailId) ?? null
    : null;
  const chainWorkspaces = chainDetail
    ? (consoleData?.workspaceAssignments ?? []).filter((item) => item.tenantEntitlementId === chainDetail.id)
    : [];
  const chainInstancesOf = (workspaceAssignmentId: string) =>
    (consoleData?.instanceAssignments ?? []).filter((item) => item.workspaceAssignmentId === workspaceAssignmentId);

  const domainOptions = useMemo(
    () => [...new Set([...catalog.templates.map((template) => template.defaultDataDomain), ...segments.map((segment) => segment.domain)].filter(Boolean))],
    [catalog.templates, segments]
  );

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t("rd.nav.capabilities")}</h1>
          <p>{t("rd.page.capabilities.desc")}</p>
        </div>
        <div className="page-head-aside">
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      <div className="stack">
        {unclassifiedApproved.length > 0 ? (
          <Notice tone="warn">
            {tx(t, "rd.cap.unclassifiedBanner", { count: unclassifiedApproved.length })}
          </Notice>
        ) : null}
        {missedTarget ? (
          <Notice tone="warn">{tx(t, "rd.cap.targetNotFound", { target: missedTarget })}</Notice>
        ) : null}
        <Card>
          <div className="form-grid form-grid-4">
            <Field htmlFor="cap-target" label={t("rd.cap.target")}>
              <select
                className="select"
                id="cap-target"
                onChange={(event) => setSelectedTargetId(event.target.value)}
                value={targetId}
              >
                {mcpTargets.map((agent) => (
                  <option key={agent.id} value={agent.id}>{agent.name}</option>
                ))}
              </select>
            </Field>
            <div className="audit-count">
              <Chip tone="neutral">{tx(t, "rd.cap.count", { count: targetCapabilities.length })}</Chip>
            </div>
            <div className="form-actions">
              <Button
                disabled={!targetId || acting}
                onClick={() => setRefreshOpen(true)}
                variant="ghost"
              >
                <RefreshCw aria-hidden="true" size={14} />
                {t("rd.cap.refresh")}
              </Button>
            </div>
            <div className="form-actions">
              <Button disabled={!targetId} onClick={openChain} variant="primary">
                {t("rd.cap.createChain")}
              </Button>
            </div>
          </div>
        </Card>

        {actionError ? <Banner desc={actionError} title={t("rd.error.other.title")} tone="danger" /> : null}

        {mcpTargets.length === 0 ? (
          <Card>
            <EmptyState desc={t("rd.cap.noTargetDesc")} title={t("rd.cap.noTarget")} />
          </Card>
        ) : (
          <div className="cockpit-grid">
            <CapabilitiesTable
              acting={acting}
              capabilities={targetCapabilities}
              onApprove={(capability) => void approveCapability(capability)}
              onDetail={(capability) => setDetail(capability)}
              t={t}
            />

            <div className="stack">
              <CapabilitiesDomainDonut segments={segments} t={t} />

              <CapabilitiesGrantList
                capabilities={capabilities}
                grants={grants}
                onManage={(entitlementId) => setChainDetailId(entitlementId)}
                t={t}
              />
            </div>
          </div>
        )}
      </div>

      <CapabilitiesRefreshModal
        acting={acting}
        onCancel={() => setRefreshOpen(false)}
        onConfirm={() => void runRefresh()}
        open={refreshOpen}
        t={t}
      />

      {detail ? (
        <CapabilitiesDetailModal
          acting={acting}
          capability={detail}
          domainOptions={domainOptions}
          key={detail.id}
          onApprove={(capability) => void approveCapability(capability)}
          onClose={() => setDetail(null)}
          onSave={(riskLevel, dataDomain) => void saveDetail(riskLevel, dataDomain)}
          t={t}
        />
      ) : null}

      {chainDetail ? (
        <CapabilitiesChainDetailModal
          acting={acting}
          agents={agents}
          capabilities={capabilities}
          entitlement={chainDetail}
          instancesOf={chainInstancesOf}
          key={chainDetail.id}
          onDisableLayer={disableChainLayer}
          onClose={() => setChainDetailId("")}
          t={t}
          workspaces={chainWorkspaces}
        />
      ) : null}

      <CapabilitiesChainFormModal
        acting={acting}
        agents={agents}
        blockerKey={blockerKey}
        chainError={chainError}
        form={form}
        onFormChange={setForm}
        onClose={() => setChainOpen(false)}
        onSubmit={() => void submitChain()}
        open={chainOpen}
        t={t}
        targetCapabilities={targetCapabilities}
        tenants={tenants}
      />
    </>
  );
}
