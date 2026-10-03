import { Component, type ErrorInfo, type ReactNode } from "react";
import { useRedesignI18n } from "../hooks/useRedesignI18n";
import { Button } from "./Button";
import { Card } from "./Card";

type ErrorBoundaryScope = "app" | "view";

interface ErrorBoundaryProps {
  children: ReactNode;
  scope: ErrorBoundaryScope;
}

interface ErrorBoundaryState {
  error: Error | null;
}

// Render failures in one view must not blank the whole console: the app-level
// boundary keeps the shell alive and the per-view boundary (keyed by route)
// lets the operator navigate elsewhere and come back to a fresh page.
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("render failed", error, info.componentStack);
  }

  render() {
    if (this.state.error !== null) {
      return <ErrorBoundaryFallback error={this.state.error} scope={this.props.scope} />;
    }
    return this.props.children;
  }
}

function ErrorBoundaryFallback({ error, scope }: { error: Error; scope: ErrorBoundaryScope }) {
  const { t } = useRedesignI18n();
  return (
    <Card title={t("rd.error.boundary.title")}>
      <p className="muted">{t(scope === "app" ? "rd.error.boundary.appHint" : "rd.error.boundary.viewHint")}</p>
      <p className="mono small">{error.message || error.name}</p>
      <Button onClick={() => window.location.reload()}>{t("rd.error.boundary.reload")}</Button>
    </Card>
  );
}
