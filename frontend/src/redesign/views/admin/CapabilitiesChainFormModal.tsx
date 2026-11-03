import { Banner } from "../../ui/Banner";
import { Button } from "../../ui/Button";
import { Field } from "../../ui/Field";
import { Modal } from "../../ui/Modal";
import {
  accessSubjectOptionForId,
  accessSubjectOptionForSelector,
  accessSubjectOptions,
  customAccessSubjectOption,
} from "../../../accessSubjects";
import type { CapabilityGrantForm } from "../../../capabilityGrantChain";
import type { Agent, Capability, Tenant } from "../../../types";
import type { useRedesignI18n } from "../../hooks/useRedesignI18n";

export function CapabilitiesChainFormModal({
  open,
  form,
  blockerKey,
  chainError,
  tenants,
  agents,
  targetCapabilities,
  acting,
  t,
  onClose,
  onFormChange,
  onSubmit
}: {
  open: boolean;
  form: CapabilityGrantForm;
  blockerKey: string | null;
  chainError: string;
  tenants: readonly Tenant[];
  agents: readonly Agent[];
  targetCapabilities: readonly Capability[];
  acting: boolean;
  t: ReturnType<typeof useRedesignI18n>["t"];
  onClose: () => void;
  onFormChange: (next: CapabilityGrantForm) => void;
  onSubmit: () => void;
}) {
  return (
    <Modal
      footer={
        <>
          <Button onClick={onClose} variant="ghost">{t("rd.apr.cancel")}</Button>
          <Button disabled={acting || blockerKey !== null} onClick={onSubmit} variant="primary">
            {t("rd.cap.createChain")}
          </Button>
        </>
      }
      onClose={onClose}
      open={open}
      title={t("rd.cap.chainTitle")}
    >
      <div className="stack">
        <p className="hint">{t("rd.cap.chainHelp")}</p>
        <div className="form-grid">
          <Field htmlFor="chain-tenant" label={t("rd.cap.field.tenant")}>
            <select
              className="select"
              id="chain-tenant"
              onChange={(event) => onFormChange({ ...form, tenantId: event.target.value })}
              value={form.tenantId}
            >
              <option value="">—</option>
              {tenants.map((tenant) => (
                <option key={tenant.id} value={tenant.id}>{tenant.name}</option>
              ))}
              {form.tenantId && !tenants.some((tenant) => tenant.id === form.tenantId) ? (
                <option value={form.tenantId}>{form.tenantId}</option>
              ) : null}
            </select>
          </Field>
          <Field htmlFor="chain-workspace" label={t("rd.cap.field.workspace")}>
            <select
              className="select"
              id="chain-workspace"
              onChange={(event) => onFormChange({ ...form, workspaceId: event.target.value })}
              value={form.workspaceId}
            >
              <option value="">—</option>
              {[...new Set(agents.map((agent) => agent.workspaceId).filter(Boolean))].map((workspaceId) => (
                <option key={workspaceId} value={workspaceId}>{workspaceId}</option>
              ))}
              {form.workspaceId && !agents.some((agent) => agent.workspaceId === form.workspaceId) ? (
                <option value={form.workspaceId}>{form.workspaceId}</option>
              ) : null}
            </select>
          </Field>
        </div>
        <Field htmlFor="chain-capability" label={t("rd.cap.field.capability")}>
          <select
            className="select"
            id="chain-capability"
            onChange={(event) => onFormChange({ ...form, capabilityId: event.target.value })}
            value={form.capabilityId}
          >
            <option value="">—</option>
            {targetCapabilities.map((capability) => (
              <option key={capability.id} value={capability.id}>
                {capability.displayName} ({t(`rd.risk.${capability.riskLevel}`)})
              </option>
            ))}
          </select>
        </Field>
        <Field htmlFor="chain-caller" label={t("rd.cap.field.caller")}>
          <select
            className="select"
            id="chain-caller"
            onChange={(event) => onFormChange({ ...form, callerInstanceId: event.target.value })}
            value={form.callerInstanceId}
          >
            <option value="">—</option>
            {agents
              .filter((agent) => agent.status === "active" && agent.channelType === "local")
              .map((agent) => (
                <option key={agent.id} value={agent.id}>{agent.name}</option>
              ))}
          </select>
        </Field>
        <SubjectPicker form={form} onFormChange={onFormChange} t={t} />
        {blockerKey ? <p className="field-error" role="alert">{t(blockerKey)}</p> : null}
        {chainError ? <Banner desc={chainError} title={t("rd.error.other.title")} tone="danger" /> : null}
      </div>
    </Modal>
  );
}

function SubjectPicker({
  form,
  onFormChange,
  t
}: {
  form: CapabilityGrantForm;
  onFormChange: (next: CapabilityGrantForm) => void;
  t: ReturnType<typeof useRedesignI18n>["t"];
}) {
  const selected = accessSubjectOptionForSelector(form.subjectSelector);
  const options = [...accessSubjectOptions, customAccessSubjectOption];
  return (
    <>
      <Field htmlFor="chain-subject" label={t("rd.cap.field.subject")}>
        <select
          className="select"
          id="chain-subject"
          onChange={(event) => {
            const option = accessSubjectOptionForId(event.target.value);
            if (option) onFormChange({ ...form, subjectSelector: option.subjectSelector });
          }}
          value={selected.id}
        >
          {options.map((option) => (
            <option key={option.id} value={option.id}>
              {t(`accessSubject.kind.${option.kind}`)} · {t(option.labelKey)}
            </option>
          ))}
        </select>
      </Field>
      {selected.id === customAccessSubjectOption.id ? (
        <Field
          hint={t("rd.cap.field.subjectCustomHint")}
          htmlFor="chain-subject-raw"
          label={t("rd.cap.field.subjectCustom")}
        >
          <input
            className="input mono"
            id="chain-subject-raw"
            onChange={(event) => onFormChange({ ...form, subjectSelector: event.target.value })}
            value={form.subjectSelector}
          />
        </Field>
      ) : null}
    </>
  );
}
