export function TechnicalId({ label, value }: { label: string; value?: string }) {
  const displayValue = value || "-";

  return (
    <span>
      <span>{label}</span>
      <span className="technical-id" title={displayValue} translate="no">
        <code>{displayValue}</code>
      </span>
    </span>
  );
}
