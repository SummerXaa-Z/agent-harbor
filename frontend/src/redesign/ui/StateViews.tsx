import { LoaderCircle, SearchX } from "lucide-react";
import type { ReactNode } from "react";
import type { ApiErrorPresentation } from "../model/apiErrorCategory";
import { Button } from "./Button";
import { ToneIcon } from "./icons";

export function LoadingState({ label }: { label: string }) {
  return (
    <div aria-live="polite" className="state-view" role="status">
      <span className="state-ico">
        <LoaderCircle aria-hidden="true" className="spin" size={18} />
      </span>
      <div className="state-desc">{label}</div>
    </div>
  );
}

export function EmptyState({ actions, desc, title }: { actions?: ReactNode; desc?: ReactNode; title: ReactNode }) {
  return (
    <div className="state-view">
      <span className="state-ico">
        <SearchX aria-hidden="true" size={18} />
      </span>
      <div className="state-title">{title}</div>
      {desc ? <div className="state-desc">{desc}</div> : null}
      {actions ? <div className="state-actions">{actions}</div> : null}
    </div>
  );
}

export function ErrorState({
  onRetry,
  presentation,
  retryLabel,
}: {
  onRetry?: () => void;
  presentation: ApiErrorPresentation;
  retryLabel?: string;
}) {
  return (
    <div className="state-view state-error" role="alert">
      <span className="state-ico">
        <ToneIcon size={18} tone="danger" />
      </span>
      <div className="state-title">{presentation.title}</div>
      <div className="state-desc">{presentation.next}</div>
      {presentation.detail ? <div className="state-desc small mono">{presentation.detail}</div> : null}
      {onRetry && retryLabel ? (
        <div className="state-actions">
          <Button onClick={onRetry} size="sm" variant="ghost">
            {retryLabel}
          </Button>
        </div>
      ) : null}
    </div>
  );
}
