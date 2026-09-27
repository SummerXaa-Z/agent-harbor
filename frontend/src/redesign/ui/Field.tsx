import type { ReactNode } from "react";

interface FieldProps {
  children: ReactNode;
  error?: ReactNode;
  hint?: ReactNode;
  htmlFor: string;
  label: ReactNode;
}

export function Field({ children, error, hint, htmlFor, label }: FieldProps) {
  return (
    <div>
      <label className="field-label" htmlFor={htmlFor}>
        {label}
      </label>
      {children}
      {error ? (
        <div className="field-error" role="alert">
          {error}
        </div>
      ) : hint ? (
        <div className="hint">{hint}</div>
      ) : null}
    </div>
  );
}

export function Switch({
  checked,
  disabled = false,
  label,
  onChange,
}: {
  checked: boolean;
  disabled?: boolean;
  label: string;
  onChange: (checked: boolean) => void;
}) {
  return (
    <button
      aria-checked={checked}
      aria-label={label}
      className="switch"
      disabled={disabled}
      onClick={() => onChange(!checked)}
      role="switch"
      title={label}
      type="button"
    />
  );
}
