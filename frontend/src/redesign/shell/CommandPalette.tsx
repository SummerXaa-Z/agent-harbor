import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { useRedesignI18n } from "../hooks/useRedesignI18n";
import { filterPaletteItems, paletteItems, type PaletteItem } from "../model/paletteIndex";
import { Modal } from "../ui/Modal";

interface CommandPaletteProps {
  onClose: () => void;
  open: boolean;
  recent: readonly PaletteItem[];
}

const groupOrder = ["recent", "actions", "pages"] as const;

// ⌘K palette: pages of both surfaces, high-frequency action entries and
// recent resources. Arrow keys move, Enter navigates, Esc closes (Modal).
export function CommandPalette({ onClose, open, recent }: CommandPaletteProps) {
  const { t } = useRedesignI18n();
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);

  const base = useMemo(() => [...recent, ...paletteItems()], [recent]);
  const resolveLabel = useMemo(
    () => (item: PaletteItem) => (item.group === "recent" ? item.label : t(item.labelKey)),
    [t],
  );
  const filtered = useMemo(() => filterPaletteItems(base, query, resolveLabel), [base, query, resolveLabel]);

  useEffect(() => {
    if (!open) {
      setQuery("");
      setActive(0);
    }
  }, [open]);
  useEffect(() => {
    setActive((current) => Math.min(current, Math.max(filtered.length - 1, 0)));
  }, [filtered.length]);

  useEffect(() => {
    listRef.current?.querySelector<HTMLElement>("[data-active='true']")?.scrollIntoView({ block: "nearest" });
  }, [active, filtered]);

  function commit(item: PaletteItem | undefined) {
    if (!item) return;
    window.location.hash = item.hash;
    onClose();
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActive((current) => (filtered.length === 0 ? 0 : (current + 1) % filtered.length));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActive((current) => (filtered.length === 0 ? 0 : (current - 1 + filtered.length) % filtered.length));
    } else if (event.key === "Enter") {
      event.preventDefault();
      commit(filtered[active]);
    }
  }

  let renderedIndex = -1;
  return (
    <Modal autoFocusQuery=".pal-input" onClose={onClose} open={open} title={t("rd.palette.title")}>
      <input
        aria-label={t("rd.palette.placeholder")}
        autoFocus
        className="pal-input"
        onChange={(event) => {
          setQuery(event.target.value);
          setActive(0);
        }}
        onKeyDown={handleKeyDown}
        placeholder={t("rd.palette.placeholder")}
        type="text"
        value={query}
      />
      <div aria-label={t("rd.palette.title")} className="pal-list" ref={listRef} role="listbox">
        {filtered.length === 0 ? <p className="pal-empty muted">{t("rd.palette.empty")}</p> : null}
        {groupOrder.map((group) => {
          const groupItems = filtered.filter((item) => item.group === group);
          if (groupItems.length === 0) return null;
          return (
            <div key={group}>
              <div className="pal-group">{t(`rd.palette.group.${group}`)}</div>
              {groupItems.map((item) => {
                renderedIndex += 1;
                const itemIndex = renderedIndex;
                const isActive = itemIndex === active;
                return (
                  <a
                    aria-selected={isActive}
                    className={isActive ? "pal-item pal-active" : "pal-item"}
                    data-active={isActive}
                    href={item.hash}
                    key={item.id}
                    onClick={(event) => {
                      event.preventDefault();
                      commit(item);
                    }}
                    onMouseEnter={() => setActive(itemIndex)}
                    role="option"
                    tabIndex={-1}
                  >
                    <span className="pal-label">
                      {item.group === "recent" ? item.label : t(item.labelKey)}
                      <span className="pal-surface">{t(`rd.surface.${item.surface}`)}</span>
                    </span>
                    {item.group === "recent" ? <span className="pal-sub mono">{item.sub}</span> : null}
                  </a>
                );
              })}
            </div>
          );
        })}
      </div>
      <p className="pal-hint muted small">{t("rd.palette.hint")}</p>
    </Modal>
  );
}
