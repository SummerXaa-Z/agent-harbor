import { Copy } from "lucide-react";

export function shortTechnicalId(value: string) {
  if (value.length <= 24) return value;
  return `${value.slice(0, 12)}...${value.slice(-8)}`;
}

export function TechnicalId({
  copyLabel,
  label,
  value
}: {
  copyLabel?: string;
  label?: string;
  value?: string;
}) {
  const displayValue = value || "-";
  const shortenedValue = shortTechnicalId(displayValue);
  const idNode = (
    <span className="technical-id" title={displayValue} translate="no">
      <code>{shortenedValue}</code>
      {value && copyLabel ? (
        <button
          aria-label={`${copyLabel} ${shortenedValue}`}
          className="technical-id-copy"
          onClick={() => void navigator.clipboard?.writeText(value)}
          title={copyLabel}
          type="button"
        >
          <Copy size={12} />
        </button>
      ) : null}
    </span>
  );

  if (!label) return idNode;

  return (
    <span className="technical-id-field">
      <span>{label}</span>
      {idNode}
    </span>
  );
}
