import { capabilityDisplayName } from "../consolePresenters";
import type { Capability } from "../types";
import type { Tone, Translator } from "../permissionWorkbenchPresenters";

export function CapabilityChipList({
  capabilities,
  emptyLabel,
  label,
  tone,
  t
}: {
  capabilities: Capability[];
  emptyLabel: string;
  label: string;
  tone: Tone;
  t: Translator;
}) {
  return (
    <div className={`approval-capability-list tone-${tone}`}>
      <strong>{label}</strong>
      {capabilities.length === 0 ? <span>{emptyLabel}</span> : null}
      <div>
        {capabilities.map((capability) => (
          <span key={capability.id}>
            {capabilityDisplayName(capability, t)} · {t(`value.${capability.action}`, capability.action)}
          </span>
        ))}
      </div>
    </div>
  );
}
