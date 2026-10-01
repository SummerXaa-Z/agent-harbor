import { useCallback, useMemo, useState } from "react";
import { callMcpRpc, createAgentKey, revokeAgentKey } from "../../api";
import { tx } from "../../localizedMessages";
import type { PermissionPackageWorkbenchPreview } from "../../permissionPackages";
import type { AccessContext } from "../model/accessContext";
import { apiErrorPresentation } from "../model/apiErrorCategory";
import {
  buildRuntimeValidationReadiness,
  classifyRuntimeValidationRun,
  mcpToolCallPayload,
  mcpToolsListPayload,
  type RuntimeValidationBlocker,
  type RuntimeValidationPlan,
} from "../model/runtimeValidation";
import { useToast } from "../ui/Toast";
import { useRedesignI18n } from "./useRedesignI18n";

export interface RuntimeValidationState {
  blockers: RuntimeValidationBlocker[];
  plan: RuntimeValidationPlan | null;
  running: boolean;
  run: () => Promise<boolean>;
}

const runtimeValidationKeyName = "runtime validation key";

// Runs the go-live runtime validation with a throwaway agent key: one
// tools/list, one denied tools/call (when the draft blocks something), one
// allowed tools/call — then revokes the key either way. Returns true when
// any evidence landed so the caller can refresh readiness. A denied probe
// the gateway does not reject (a pre-existing grant wider than the package,
// round 6 finding #41) does not abort the run: the allowed probe still
// executes and the mismatch is diagnosed by name instead of dead-ending
// readiness with no explanation.
export function useRuntimeValidation({
  context,
  live,
  preview,
}: {
  context: AccessContext;
  live: boolean;
  preview: PermissionPackageWorkbenchPreview | null;
}): RuntimeValidationState {
  const { language, t } = useRedesignI18n();
  const showToast = useToast();
  const [running, setRunning] = useState(false);

  const readiness = useMemo(
    () => buildRuntimeValidationReadiness({
      allowedCapabilities: preview?.draft.allowedCapabilities ?? [],
      blockedCapabilities: preview?.draft.blockedCapabilities ?? [],
      context,
      hasApplication: Boolean(preview?.summary.applied || preview?.latestApplication),
      liveDataAvailable: live,
      runId: "",
    }),
    [context, live, preview],
  );

  const run = useCallback(async () => {
    if (running) return false;
    const plan = readiness.plan;
    if (!plan) return false;
    const runPlan = { ...plan, runId: `ui-validation-${Date.now().toString(36)}` };
    setRunning(true);
    let callerKeyId = "";
    try {
      const callerKey = await createAgentKey(
        { agentId: runPlan.callerInstanceId, expiresInSeconds: 900, name: runtimeValidationKeyName },
        "",
      );
      callerKeyId = callerKey.id;
      const toolList = await callMcpRpc(runPlan.targetId, mcpToolsListPayload(), callerKey.key, runPlan.runId, "", runPlan.subjectId);
      let deniedStatus: number | null = null;
      if (runPlan.blockedCapabilityKey) {
        const deniedCall = await callMcpRpc(
          runPlan.targetId,
          mcpToolCallPayload(runPlan.blockedCapabilityKey),
          callerKey.key,
          runPlan.runId,
          "",
          runPlan.subjectId,
        );
        deniedStatus = deniedCall.status;
      }
      const allowedCall = await callMcpRpc(
        runPlan.targetId,
        mcpToolCallPayload(runPlan.allowedCapabilityKey),
        callerKey.key,
        runPlan.runId,
        "",
        runPlan.subjectId,
      );
      const outcome = classifyRuntimeValidationRun({
        allowedOk: allowedCall.ok,
        allowedStatus: allowedCall.status,
        blockedCapabilityKey: runPlan.blockedCapabilityKey,
        deniedStatus,
        toolListOk: toolList.ok,
        toolListStatus: toolList.status,
      });
      if (outcome.kind === "toolListFailed") {
        showToast(tx(t, "rd.golive.validationRpcUnexpected", { status: outcome.status }), "danger");
        return false;
      }
      if (outcome.kind === "allowedFailed") {
        showToast(tx(t, "rd.golive.validationRpcUnexpected", { status: outcome.status }), "danger");
        return false;
      }
      if (outcome.kind === "denyUnexpected") {
        showToast(
          tx(t, "rd.golive.validationDeniedUnexpected", { capability: outcome.blockedCapabilityKey, status: outcome.deniedStatus }),
          "danger",
        );
        showToast(tx(t, "rd.golive.validationDoneDenyMissing", { allowed: outcome.allowedStatus }));
        return true;
      }
      showToast(
        outcome.deniedStatus
          ? tx(t, "rd.golive.validationDone", { allowed: outcome.allowedStatus, denied: outcome.deniedStatus })
          : tx(t, "rd.golive.validationDoneNoDenied", { allowed: outcome.allowedStatus }),
      );
      return true;
    } catch (error) {
      showToast(apiErrorPresentation(t, language, error, "error.permissionRuntimeValidationFailed").detail, "danger");
      return false;
    } finally {
      if (callerKeyId) {
        try {
          await revokeAgentKey(callerKeyId, "");
        } catch {
          // The key expires in 15 minutes on its own; the run already failed.
        }
      }
      setRunning(false);
    }
  }, [language, readiness.plan, running, showToast, t]);

  return { blockers: readiness.blockers, plan: readiness.plan, running, run };
}
