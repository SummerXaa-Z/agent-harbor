import { useEffect, useMemo, useState } from "react";
import { translatedValue } from "../../../consolePresenters";
import { tx } from "../../../localizedMessages";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { usePermissionCatalog } from "../../hooks/usePermissionCatalog";
import { normalizeTemplateId, templateRuleRows } from "../../model/policyTemplates";
import { templateSummary } from "../../model/templateMatch";
import { adminHash, userHash } from "../../router";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip, TagOutline } from "../../ui/Chip";
import { EmptyState } from "../../ui/StateViews";
import type { AdminViewProps } from "./adminViewProps";

// Templates are a static, read-only catalog (difference #7): the only real
// action per template is starting an application derived from it.
export function PoliciesView({ data, onRetry, params }: AdminViewProps) {
  const { language, t } = useRedesignI18n();
  const live = data.status === "live";
  const catalog = usePermissionCatalog(live);
  const [selectedId, setSelectedId] = useState<string | null>(() => normalizeTemplateId(params.template, catalog.templates));

  useEffect(() => {
    const next = adminHash("policies", selectedId ? { template: selectedId } : {});
    if (window.location.hash !== next) {
      window.history.replaceState(null, "", `${window.location.pathname}${window.location.search}${next}`);
    }
  }, [selectedId]);

  const selected = useMemo(
    () => catalog.templates.find((template) => template.id === selectedId) ?? null,
    [catalog.templates, selectedId]
  );
  const rules = useMemo(() => (selected ? templateRuleRows(selected) : []), [selected]);

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t("rd.nav.policies")}</h1>
          <p>{t("rd.page.policies.desc")}</p>
        </div>
        <div className="page-head-aside">
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      <div className="stack">
        {catalog.templates.length === 0 ? (
          <Card>
            <EmptyState desc={t("rd.pol.emptyDesc")} title={t("rd.common.empty")} />
          </Card>
        ) : (
          <div className="tpl-grid">
            {catalog.templates.map((template) => {
              const active = template.id === selectedId;
              return (
                <div className={active ? "card tpl-card tpl-card-active" : "card tpl-card"} key={template.id}>
                  <div className="tpl-head">
                    <span className="tpl-name">{template.name}</span>
                    <span className="mono small">{template.id}</span>
                  </div>
                  <p className="tpl-desc">{templateSummary(template, language)}</p>
                  <div className="chip-row">
                    <TagOutline>{t("rd.pol.meta.domain")}: {template.defaultDataDomain}</TagOutline>
                    <TagOutline>{t("rd.pol.meta.allow")}: {template.allowedActions.map((action) => translatedValue(t, action)).join(" / ")}</TagOutline>
                    {template.blockedActions.length > 0 ? (
                      <TagOutline>{t("rd.pol.meta.block")}: {template.blockedActions.map((action) => translatedValue(t, action)).join(" / ")}</TagOutline>
                    ) : null}
                    {template.blockedRisks.length > 0 ? (
                      <TagOutline>{t("rd.pol.meta.blockRisk")}: {template.blockedRisks.map((risk) => t(`rd.risk.${risk}`)).join(" / ")}</TagOutline>
                    ) : null}
                  </div>
                  <div className="tpl-meta">
                    <Button href={userHash("apply", { template: template.id })} size="sm" variant="primary">
                      {t("rd.pol.apply")}
                    </Button>
                    <Button onClick={() => setSelectedId(template.id)} size="sm" variant="ghost">
                      {active ? t("rd.pol.selected") : t("rd.pol.inspect")}
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {selected ? (
          <Card title={tx(t, "rd.pol.rulesTitle", { name: selected.name })}>
            <div className="rule-list">
              {rules.map((rule) => (
                <div className="rule-line" key={`${rule.capabilityKey}:${rule.decision}`}>
                  <div className="rule-main">
                    <div className="rule-name mono">{rule.capabilityKey}</div>
                    <div className="rule-sub">{t(rule.reasonKey, rule.reason)}</div>
                  </div>
                  <Chip tone={rule.decision === "allow" ? "success" : "danger"}>
                    {rule.decision === "allow" ? t("rd.decision.allowed") : t("rd.decision.blocked")}
                  </Chip>
                </div>
              ))}
            </div>
          </Card>
        ) : null}
      </div>
    </>
  );
}
