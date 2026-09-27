import { tx } from "../../../localizedMessages";
import type { PermissionPackageTemplate } from "../../../permissionPackages";
import type { Agent } from "../../../types";
import type { AccessContextState } from "../../hooks/useAccessContext";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { accessContextKey, type AccessContext } from "../../model/accessContext";

export function agentName(agents: readonly Agent[], id: string): string {
  return agents.find((agent) => agent.id === id)?.name ?? id;
}

export function templateName(templates: readonly PermissionPackageTemplate[], id: string): string {
  return templates.find((template) => template.id === id)?.name ?? id;
}

// The request the user is working on, switchable between recent applications.
// Picking one replaces the whole context so region and request text stay with
// the snapshot they belong to.
export function AccessContextSwitcher({
  accessContext,
  agents,
  onSelect,
  templates,
}: {
  accessContext: AccessContextState;
  agents: readonly Agent[];
  onSelect?: (context: AccessContext) => void;
  templates: readonly PermissionPackageTemplate[];
}) {
  const { t } = useRedesignI18n();
  const currentKey = accessContextKey(accessContext.context);
  const options = accessContext.options;
  const hasCurrent = options.some((option) => option.key === currentKey);
  const label = (context: AccessContext) =>
    tx(t, "rd.context.option", {
      caller: agentName(agents, context.callerInstanceId),
      target: agentName(agents, context.targetId),
      template: templateName(templates, context.templateId),
    });

  if (options.length === 0) return null;
  return (
    <div className="ctx-bar">
      <label className="field-label" htmlFor="rd-access-context">
        {t("rd.context.label")}
      </label>
      <select
        className="select"
        id="rd-access-context"
        onChange={(event) => {
          const option = options.find((item) => item.key === event.target.value);
          if (!option) return;
          accessContext.replace(option.context);
          onSelect?.(option.context);
        }}
        value={hasCurrent ? currentKey : ""}
      >
        {hasCurrent ? null : (
          <option value="">{accessContext.context.callerInstanceId ? label(accessContext.context) : t("rd.context.none")}</option>
        )}
        {options.map((option) => (
          <option key={option.key} value={option.key}>
            {label(option.context)}
          </option>
        ))}
      </select>
    </div>
  );
}
