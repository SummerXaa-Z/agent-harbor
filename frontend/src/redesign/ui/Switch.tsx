interface SwitchProps {
  checked: boolean;
  disabled?: boolean;
  label: string;
  onChange: (checked: boolean) => void;
}

// Tiny toggle used inside table cells; the visible label keeps the state
// readable without relying on color alone.
export function Switch({ checked, disabled = false, label, onChange }: SwitchProps) {
  return (
    <button
      aria-checked={checked}
      aria-label={label}
      className={checked ? "switch switch-on" : "switch"}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      role="switch"
      type="button"
    >
      <span className="switch-label">{label}</span>
      <span className="switch-knob" aria-hidden="true" />
    </button>
  );
}
