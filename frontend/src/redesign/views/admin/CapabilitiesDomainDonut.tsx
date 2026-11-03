import { Card } from "../../ui/Card";
import { capabilityDomainSegments } from "../../model/capabilityCatalog";
import type { useRedesignI18n } from "../../hooks/useRedesignI18n";

export function CapabilitiesDomainDonut({
  segments,
  t
}: {
  segments: ReturnType<typeof capabilityDomainSegments>;
  t: ReturnType<typeof useRedesignI18n>["t"];
}) {
  return (
    <Card title={t("rd.cap.domainsTitle")}>
      <DomainDonut segments={segments} t={t} />
    </Card>
  );
}

function DomainDonut({
  segments,
  t
}: {
  segments: ReturnType<typeof capabilityDomainSegments>;
  t: ReturnType<typeof useRedesignI18n>["t"];
}) {
  const total = segments.reduce((sum, segment) => sum + segment.count, 0);
  const radius = 45;
  const center = 60;
  let offset = 0;

  return (
    <div className="donut-box">
      <svg aria-hidden="true" className="donut-svg" role="img" viewBox="0 0 120 120">
        <circle className="donut-track" cx={center} cy={center} r={radius} />
        {segments.map((segment, index) => {
          const element = (
            <circle
              className={index % 2 === 0 ? "donut-seg-1" : "donut-seg-2"}
              cx={center}
              cy={center}
              key={segment.domain || "unclassified"}
              pathLength={1}
              r={radius}
              strokeDasharray={`${segment.fraction} ${1 - segment.fraction}`}
              strokeDashoffset={-offset}
              transform={`rotate(-90 ${center} ${center})`}
            />
          );
          offset += segment.fraction;
          return element;
        })}
        <text className="donut-center-num" dominantBaseline="central" textAnchor="middle" x={center} y={center - 15}>
          {total}
        </text>
        <text className="donut-center-label" dominantBaseline="central" textAnchor="middle" x={center} y={center + 3}>
          {t("rd.cap.donutCapabilities")}
        </text>
      </svg>
      <div className="donut-legend">
        {segments.map((segment, index) => (
          <div className="chart-legend" key={segment.domain || "unclassified"}>
            <span>
              <span className={index % 2 === 0 ? "leg-dot leg-calls" : "leg-dot leg-audit"} />
              <span className="small">{segment.domain || t("rd.cap.domainUnset")}</span>
            </span>
            <span className="small muted">{segment.count}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
