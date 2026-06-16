import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { ChevronRight, ExternalLink, MoreHorizontal, X } from "lucide-react";

import type { Tone } from "../consolePresenters";

export function MetricCard({
  icon,
  label,
  value,
  detail,
  tone
}: {
  icon: ReactNode;
  label: string;
  value: string;
  detail: string;
  tone: Tone;
}) {
  return (
    <article className={`metric-card tone-${tone}`}>
      <div className="metric-icon">{icon}</div>
      <div>
        <span>{label}</span>
        <strong>{value}</strong>
        <small>{detail}</small>
      </div>
    </article>
  );
}

export function Panel({
  title,
  icon,
  action,
  className,
  children
}: {
  title: string;
  icon: ReactNode;
  action?: ReactNode;
  className?: string;
  children: ReactNode;
}) {
  return (
    <section className={`panel ${className ?? ""}`}>
      <header className="panel-header">
        <div>
          {icon}
          <h2>{title}</h2>
        </div>
        {action}
      </header>
      {children}
    </section>
  );
}

export function ActionModalButton({
  title,
  icon,
  openLabel,
  closeLabel,
  id,
  onClose,
  openToken,
  tone = "secondary",
  variant = "compact",
  children
}: {
  title: string;
  icon: ReactNode;
  openLabel: string;
  closeLabel: string;
  id?: string;
  onClose?: () => void;
  openToken?: number;
  tone?: "primary" | "secondary";
  variant?: "compact" | "command";
  children: ReactNode;
}) {
  const triggerClassName = variant === "command"
    ? `action-modal-trigger action-modal-trigger-command is-${tone}`
    : "action-modal-trigger action-modal-trigger-compact";

  return (
    <ActionModalLauncher
      className="action-modal-inline"
      closeLabel={closeLabel}
      icon={icon}
      id={id}
      onClose={onClose}
      openLabel={openLabel}
      openToken={openToken}
      title={title}
      triggerClassName={triggerClassName}
    >
      {children}
    </ActionModalLauncher>
  );
}

function ActionModalLauncher({
  title,
  icon,
  openLabel,
  closeLabel,
  className,
  triggerClassName,
  id,
  onClose,
  openToken,
  children
}: {
  title: string;
  icon: ReactNode;
  openLabel: string;
  closeLabel: string;
  className: string;
  triggerClassName: string;
  id?: string;
  onClose?: () => void;
  openToken?: number;
  children: ReactNode;
}) {
  const generatedId = useId();
  const dialogId = `${id ?? "action-modal"}-${generatedId}`;
  const [open, setOpen] = useState(false);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  function closeModal() {
    if (!open) return;
    setOpen(false);
    onClose?.();
  }

  useEffect(() => {
    if (openToken === undefined) return;
    setOpen(true);
  }, [openToken]);

  useEffect(() => {
    if (!open) return;

    const activeElement = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const previousOverflow = document.body.style.overflow;
    const focusTimer = window.setTimeout(() => closeButtonRef.current?.focus(), 0);
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") closeModal();
    };

    document.body.style.overflow = "hidden";
    document.addEventListener("keydown", onKeyDown);

    return () => {
      window.clearTimeout(focusTimer);
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", onKeyDown);
      activeElement?.focus();
    };
  }, [open]);

  return (
    <div className={className} id={id}>
      <button
        aria-label={`${title} ${openLabel}`}
        aria-controls={dialogId}
        aria-expanded={open}
        aria-haspopup="dialog"
        className={triggerClassName}
        type="button"
        onClick={() => setOpen(true)}
      >
        <span className="action-modal-trigger-label">
          {icon}
          <strong>{title}</strong>
        </span>
        <span aria-hidden="true" className="action-modal-trigger-affordance">
          {openLabel}
          <ChevronRight size={15} />
        </span>
      </button>
      {open ? (
        <div
          className="action-modal-backdrop"
          role="presentation"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) closeModal();
          }}
        >
          <section
            aria-labelledby={`${dialogId}-title`}
            aria-modal="true"
            className="action-modal-panel"
            id={dialogId}
            role="dialog"
          >
            <header className="action-modal-header">
              <div>
                {icon}
                <h2 id={`${dialogId}-title`}>{title}</h2>
              </div>
              <button
                ref={closeButtonRef}
                aria-label={closeLabel}
                className="icon-button compact"
                title={closeLabel}
                type="button"
                onClick={closeModal}
              >
                <X size={15} />
              </button>
            </header>
            <div className="action-modal-body">{children}</div>
          </section>
        </div>
      ) : null}
    </div>
  );
}

export function IconMore({ title = "More" }: { title?: string }) {
  return (
    <button aria-label={title} className="icon-button compact" title={title} type="button">
      <MoreHorizontal size={16} />
    </button>
  );
}

export function IconOpen({ title = "Open" }: { title?: string }) {
  return (
    <button aria-label={title} className="icon-button compact" title={title} type="button">
      <ExternalLink size={15} />
    </button>
  );
}
