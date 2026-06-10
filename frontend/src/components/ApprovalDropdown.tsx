import { useId, useState, type FocusEvent, type KeyboardEvent } from "react";
import { Check, ChevronDown } from "lucide-react";
import { dropdownKeyAction, nextDropdownActiveIndex } from "../dropdownKeyboard";

export interface ApprovalDropdownOption {
  label: string;
  value: string;
}

export function ApprovalDropdown({
  id,
  label,
  onChange,
  options,
  value
}: {
  id?: string;
  label: string;
  onChange: (value: string) => void;
  options: ApprovalDropdownOption[];
  value: string;
}) {
  const generatedId = useId();
  const [open, setOpen] = useState(false);
  const selectedIndex = Math.max(0, options.findIndex((option) => option.value === value));
  const [activeIndex, setActiveIndex] = useState(selectedIndex);
  const selectedOption = options.find((option) => option.value === value) ?? options[0];
  const rootId = id ?? generatedId.replaceAll(":", "");
  const labelId = `${rootId}-label`;
  const listboxId = `${rootId}-listbox`;
  const activeOptionId = open && activeIndex >= 0 ? `${rootId}-option-${activeIndex}` : undefined;
  const openMenu = () => {
    setActiveIndex(selectedIndex);
    setOpen(true);
  };
  const closeOnBlur = (event: FocusEvent<HTMLDivElement>) => {
    if (!event.currentTarget.contains(event.relatedTarget)) {
      setOpen(false);
    }
  };
  const handleKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    const action = dropdownKeyAction(event.key, open);
    if (action === "ignore") return;
    event.preventDefault();
    if (action === "close") {
      setOpen(false);
      return;
    }
    if (action === "open") {
      openMenu();
      return;
    }
    if (action === "move") {
      setActiveIndex((current) => nextDropdownActiveIndex(current, options.length, event.key));
      return;
    }
    const option = options[activeIndex] ?? selectedOption;
    if (option) {
      onChange(option.value);
      setOpen(false);
    }
  };

  return (
    <div className={`approval-dropdown ${open ? "is-open" : ""}`} onBlur={closeOnBlur}>
      <span className="sr-only" id={labelId}>{label}</span>
      <button
        aria-activedescendant={activeOptionId}
        aria-controls={listboxId}
        aria-expanded={open}
        aria-haspopup="listbox"
        aria-labelledby={labelId}
        className="approval-dropdown-trigger"
        onClick={() => {
          if (open) {
            setOpen(false);
            return;
          }
          openMenu();
        }}
        onKeyDown={handleKeyDown}
        role="combobox"
        type="button"
      >
        <span>{selectedOption?.label ?? "-"}</span>
        <ChevronDown aria-hidden="true" size={15} />
      </button>
      {open ? (
        <div className="approval-dropdown-menu" id={listboxId} role="listbox">
          {options.map((option, index) => {
            const selected = option.value === value;
            return (
              <button
                aria-selected={selected}
                className={`approval-dropdown-option ${selected ? "is-selected" : ""} ${index === activeIndex ? "is-active" : ""}`}
                id={`${rootId}-option-${index}`}
                key={option.value || "empty-option"}
                onClick={() => {
                  onChange(option.value);
                  setOpen(false);
                }}
                onMouseEnter={() => setActiveIndex(index)}
                role="option"
                type="button"
              >
                <span>{option.label}</span>
                {selected ? <Check aria-hidden="true" size={14} /> : null}
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
