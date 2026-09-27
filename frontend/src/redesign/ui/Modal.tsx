import { createContext, useContext, useEffect, useId, useRef, type KeyboardEvent, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { useRedesignI18n } from "../hooks/useRedesignI18n";
import { CloseIcon } from "./icons";

// Dialogs render into a host inside `.ah2` (so tokens apply) but outside the
// page content, which keeps them above the sticky topbar and sidebar.
export const OverlayRootContext = createContext<HTMLElement | null>(null);

const focusableSelector = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  "[tabindex]:not([tabindex='-1'])",
].join(",");

interface ModalProps {
  children: ReactNode;
  footer?: ReactNode;
  onClose: () => void;
  open: boolean;
  size?: "default" | "narrow" | "wide";
  title: ReactNode;
}

export function Modal(props: ModalProps) {
  const overlayRoot = useContext(OverlayRootContext);
  if (!props.open || !overlayRoot) return null;
  return createPortal(<ModalDialog {...props} />, overlayRoot);
}

function ModalDialog({ children, footer, onClose, size = "default", title }: ModalProps) {
  const { t } = useRedesignI18n();
  const titleId = useId();
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const previouslyFocused = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const dialog = dialogRef.current;
    const first = dialog?.querySelector<HTMLElement>(focusableSelector);
    (first ?? dialog)?.focus();
    return () => previouslyFocused?.focus();
  }, []);

  function handleKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key === "Escape") {
      event.stopPropagation();
      onClose();
      return;
    }
    if (event.key !== "Tab" || !dialogRef.current) return;
    const focusable = Array.from(dialogRef.current.querySelectorAll<HTMLElement>(focusableSelector));
    if (focusable.length === 0) {
      event.preventDefault();
      return;
    }
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  const sizeClass = size === "default" ? "modal" : `modal modal-${size}`;
  return (
    <div
      className="modal-mask"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        aria-labelledby={titleId}
        aria-modal="true"
        className={sizeClass}
        onKeyDown={handleKeyDown}
        ref={dialogRef}
        role="dialog"
        tabIndex={-1}
      >
        <div className="modal-h">
          <h3 id={titleId}>{title}</h3>
          <button aria-label={t("rd.common.close")} className="icon-btn sm modal-x" onClick={onClose} type="button">
            <CloseIcon />
          </button>
        </div>
        <div className="modal-b">{children}</div>
        {footer ? <div className="modal-f">{footer}</div> : null}
      </div>
    </div>
  );
}
