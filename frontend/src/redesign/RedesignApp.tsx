import { Suspense, lazy, useEffect, useMemo, useState } from "react";
import { apiBase } from "../api";
import { useConsoleAuth } from "../hooks/useConsoleAuth";
import "../styles/redesign.css";
import { RedesignI18nContext, useRedesignI18n, useRedesignLanguage, type InitialI18n } from "./hooks/useRedesignI18n";
import { useNotifications } from "./hooks/useNotifications";
import { useRedesignData, type RedesignData } from "./hooks/useRedesignData";
import { recentItems } from "./model/paletteIndex";
import { otherSurface, parseRedesignHash, type RedesignRoute, type SurfaceRoute, type UserView as UserViewName } from "./router";
import { LoginCard } from "./shell/LoginCard";
import { CommandPalette } from "./shell/CommandPalette";
import { Sidebar } from "./shell/Sidebar";
import { Topbar } from "./shell/Topbar";
import { ErrorBoundary } from "./ui/ErrorBoundary";
import { OverlayRootContext } from "./ui/Modal";
import { LoadingState } from "./ui/StateViews";
import { ToastProvider, useToast } from "./ui/Toast";
import { EntryPage } from "./views/EntryPage";
import { NotFoundView } from "./views/NotFoundView";
import type { UserViewProps } from "./views/user/userViewProps";

// Views load on demand so each route ships as its own chunk; the shell,
// login, and entry surfaces stay eager for first paint. Chunk failures
// surface through the view-scoped ErrorBoundary around the Suspense.
const AdminView = lazy(() => import("./views/admin/AdminView").then((m) => ({ default: m.AdminView })));
const ApplyView = lazy(() => import("./views/user/ApplyView").then((m) => ({ default: m.ApplyView })));
const AskView = lazy(() => import("./views/user/AskView").then((m) => ({ default: m.AskView })));
const GoLiveView = lazy(() => import("./views/user/GoLiveView").then((m) => ({ default: m.GoLiveView })));
const HomeView = lazy(() => import("./views/user/HomeView").then((m) => ({ default: m.HomeView })));
const MineView = lazy(() => import("./views/user/MineView").then((m) => ({ default: m.MineView })));

type ConsoleAuth = ReturnType<typeof useConsoleAuth>;

export default function RedesignApp({ hash, initialI18n }: { hash: string; initialI18n: InitialI18n }) {
  const parsed = parseRedesignHash(hash);
  const i18n = useRedesignLanguage(initialI18n);
  const auth = useConsoleAuth();
  const data = useRedesignData(auth.accessReady);
  const [overlayRoot, setOverlayRoot] = useState<HTMLDivElement | null>(null);
  const canonicalHash = parsed?.canonicalHash;
  const redirected = parsed?.redirected ?? false;

  useEffect(() => {
    if (!redirected || !canonicalHash) return;
    window.history.replaceState(null, "", `${window.location.pathname}${window.location.search}${canonicalHash}`);
  }, [canonicalHash, redirected]);

  return (
    <RedesignI18nContext.Provider value={i18n}>
      <div className="ah2">
        <OverlayRootContext.Provider value={overlayRoot}>
          <ToastProvider>
            {parsed ? <RouteOutlet auth={auth} data={data} route={parsed.route} routeKey={parsed.canonicalHash} /> : null}
          </ToastProvider>
        </OverlayRootContext.Provider>
        <div ref={setOverlayRoot} />
      </div>
    </RedesignI18nContext.Provider>
  );
}

function RouteOutlet({
  auth,
  data,
  route,
  routeKey,
}: {
  auth: ConsoleAuth;
  data: RedesignData;
  route: RedesignRoute;
  routeKey: string;
}) {
  const { t } = useRedesignI18n();
  if (route.surface === "entry") {
    return <EntryPage apiBase={apiBase} session={auth.session} />;
  }
  if (route.surface === "notfound") {
    return <NotFoundView attempted={route.attempted} />;
  }
  if (!auth.session && auth.sessionLoading) {
    return (
      <div className="login">
        <LoadingState label={t("auth.sessionLoading")} />
      </div>
    );
  }
  if (!auth.accessReady) {
    return (
      <LoginCard
        loginKey={auth.loginKey}
        message={auth.loginMessage}
        onLoginKeyChange={auth.setLoginKey}
        onSubmit={() => void auth.login()}
        submitting={auth.loginSubmitting}
      />
    );
  }
  return <SurfaceShell auth={auth} data={data} route={route} routeKey={routeKey} />;
}

function SurfaceShell({
  auth,
  data,
  route,
  routeKey,
}: {
  auth: ConsoleAuth;
  data: RedesignData;
  route: SurfaceRoute;
  routeKey: string;
}) {
  const { t } = useRedesignI18n();
  const showToast = useToast();
  const notifications = useNotifications(data, auth.session);
  const [paletteOpen, setPaletteOpen] = useState(false);

  useEffect(() => {
    window.scrollTo({ left: 0, top: 0 });
  }, [routeKey]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setPaletteOpen((open) => !open);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  async function refresh() {
    const live = await data.reload();
    showToast(live ? t("rd.toast.refreshed") : t("rd.toast.refreshFailed"), live ? "success" : "danger");
  }

  const recent = useMemo(
    () => recentItems({ agents: data.data?.agents ?? [], approvals: notifications.approvals }),
    [data.data, notifications.approvals],
  );
  const otherUnread = notifications.unread[otherSurface(route.surface)];

  return (
    <>
      <Sidebar
        activeView={route.view}
        onSignOut={() => void auth.logout()}
        session={auth.session}
        surface={route.surface}
      />
      <div className="main">
        <Topbar
          notifications={notifications}
          onOpenPalette={() => setPaletteOpen(true)}
          onRefresh={() => void refresh()}
          otherSurfaceUnread={otherUnread}
          refreshing={data.loading}
          surface={route.surface}
          view={route.view}
        />
        <main className="content">
          <div className="view" key={routeKey}>
            <ErrorBoundary scope="view">
              <Suspense fallback={<LoadingState label={t("rd.common.loading")} />}>
                {route.surface === "user" ? (
                  <UserView data={data} onRetry={() => void refresh()} params={route.params} session={auth.session} view={route.view} />
                ) : (
                  <AdminView data={data} onRetry={() => void refresh()} params={route.params} session={auth.session} view={route.view} />
                )}
              </Suspense>
            </ErrorBoundary>
          </div>
        </main>
      </div>
      <CommandPalette onClose={() => setPaletteOpen(false)} open={paletteOpen} recent={recent} />
    </>
  );
}

function UserView({ view, ...props }: UserViewProps & { view: UserViewName }) {
  switch (view) {
    case "home":
      return <HomeView {...props} />;
    case "ask":
      return <AskView {...props} />;
    case "mine":
      return <MineView {...props} />;
    case "apply":
      return <ApplyView {...props} />;
    case "golive":
      return <GoLiveView {...props} />;
  }
}
