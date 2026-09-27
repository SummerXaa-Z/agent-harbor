import { useEffect, useState } from "react";
import { apiBase } from "../api";
import { useConsoleAuth } from "../hooks/useConsoleAuth";
import "../styles/redesign.css";
import { RedesignI18nContext, useRedesignI18n, useRedesignLanguage } from "./hooks/useRedesignI18n";
import { useRedesignData, type RedesignData } from "./hooks/useRedesignData";
import { parseRedesignHash, type RedesignRoute, type SurfaceRoute, type UserView as UserViewName } from "./router";
import { LoginCard } from "./shell/LoginCard";
import { Sidebar } from "./shell/Sidebar";
import { Topbar } from "./shell/Topbar";
import { OverlayRootContext } from "./ui/Modal";
import { LoadingState } from "./ui/StateViews";
import { ToastProvider, useToast } from "./ui/Toast";
import { EntryPage } from "./views/EntryPage";
import { PlaceholderView } from "./views/PlaceholderView";
import { ApplyView } from "./views/user/ApplyView";
import { AskView } from "./views/user/AskView";
import { GoLiveView } from "./views/user/GoLiveView";
import { HomeView } from "./views/user/HomeView";
import { MineView } from "./views/user/MineView";
import type { UserViewProps } from "./views/user/userViewProps";

type ConsoleAuth = ReturnType<typeof useConsoleAuth>;

export default function RedesignApp({ hash }: { hash: string }) {
  const parsed = parseRedesignHash(hash);
  const i18n = useRedesignLanguage();
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

  useEffect(() => {
    window.scrollTo({ left: 0, top: 0 });
  }, [routeKey]);

  async function refresh() {
    const live = await data.reload();
    showToast(live ? t("rd.toast.refreshed") : t("rd.toast.refreshFailed"), live ? "success" : "danger");
  }

  return (
    <>
      <Sidebar activeView={route.view} onSignOut={() => void auth.logout()} session={auth.session} surface={route.surface} />
      <div className="main">
        <Topbar onRefresh={() => void refresh()} refreshing={data.loading} surface={route.surface} view={route.view} />
        <main className="content">
          <div className="view" key={routeKey}>
            {route.surface === "user" ? (
              <UserView data={data} onRetry={() => void refresh()} params={route.params} session={auth.session} view={route.view} />
            ) : (
              <PlaceholderView data={data} onRetry={() => void refresh()} view={route.view} />
            )}
          </div>
        </main>
      </div>
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
