export const userViews = ["home", "ask", "mine", "apply", "golive"] as const;
export const adminViews = [
  "cockpit",
  "approvals",
  "traces",
  "tenants",
  "registry",
  "capabilities",
  "policies",
  "routes",
  "admin",
] as const;

export type Surface = "user" | "admin";
export type UserView = (typeof userViews)[number];
export type AdminView = (typeof adminViews)[number];
export type RedesignView = UserView | AdminView;
export type RouteParams = Record<string, string>;

export type RedesignRoute =
  | { surface: "entry" }
  | { surface: "notfound"; attempted: string }
  | { surface: "user"; view: UserView; params: RouteParams }
  | { surface: "admin"; view: AdminView; params: RouteParams };

export type SurfaceRoute = Extract<RedesignRoute, { surface: "user" | "admin" }>;

export interface ParsedRedesignHash {
  canonicalHash: string;
  redirected: boolean;
  route: RedesignRoute;
}

export const defaultUserView: UserView = "home";
export const defaultAdminView: AdminView = "cockpit";
export const entryHash = "#/";

const permissionScopeParams = ["template", "caller", "target", "capability", "subject", "approval"] as const;

// Whitelisted query parameters per view, in serialization order.
const viewParams: Record<RedesignView, readonly string[]> = {
  home: [],
  ask: ["caller", "target", "capability", "subject"],
  mine: ["caller"],
  apply: permissionScopeParams,
  golive: permissionScopeParams,
  cockpit: [],
  approvals: ["id", "status"],
  traces: ["range", "type", "result"],
  tenants: ["tenant", "tab"],
  registry: ["agent", "tenant", "workspace"],
  capabilities: ["target"],
  policies: ["template"],
  routes: [],
  admin: [],
};

const enumeratedParams: Partial<Record<RedesignView, Record<string, readonly string[]>>> = {
  tenants: { tab: ["org", "center", "profile"] },
};

export interface NavEntry<V extends RedesignView = RedesignView> {
  labelKey: string;
  view: V;
}

export interface NavSection<V extends RedesignView = RedesignView> {
  items: readonly NavEntry<V>[];
  labelKey?: string;
}

export const userNavSections: readonly NavSection<UserView>[] = [
  {
    items: userViews.map((view) => ({ labelKey: viewLabelKey(view), view })),
  },
];

export const adminNavSections: readonly NavSection<AdminView>[] = [
  { labelKey: "rd.navGroup.overview", items: [{ labelKey: viewLabelKey("cockpit"), view: "cockpit" }] },
  {
    labelKey: "rd.navGroup.review",
    items: [
      { labelKey: viewLabelKey("approvals"), view: "approvals" },
      { labelKey: viewLabelKey("traces"), view: "traces" },
    ],
  },
  {
    labelKey: "rd.navGroup.governance",
    items: (["tenants", "registry", "capabilities", "policies", "routes"] as const).map((view) => ({
      labelKey: viewLabelKey(view),
      view,
    })),
  },
];

// Administrators & boundaries sits in the sidebar footer, as in the prototype.
export const adminFooterNav: NavEntry<AdminView> = { labelKey: viewLabelKey("admin"), view: "admin" };

// Until each redesigned page lands, placeholders link to the legacy page that
// still does the job.
export const legacyHashForView: Record<RedesignView, string> = {
  home: "#getting-started",
  ask: "#ask",
  mine: "#access",
  apply: "#ai-admin",
  golive: "#go-live",
  cockpit: "#cockpit",
  approvals: "#ai-admin",
  traces: "#traces",
  tenants: "#tenants",
  registry: "#registry",
  capabilities: "#capabilities",
  policies: "#policies",
  routes: "#routes",
  admin: "#admin-access",
};

export function viewLabelKey(view: RedesignView): string {
  return `rd.nav.${view}`;
}

export function surfaceViews(surface: Surface): readonly RedesignView[] {
  return surface === "user" ? userViews : adminViews;
}

export function isUserView(value: string): value is UserView {
  return (userViews as readonly string[]).includes(value);
}

export function isAdminView(value: string): value is AdminView {
  return (adminViews as readonly string[]).includes(value);
}

// Returns null for every hash the legacy console owns. Since P5 the empty
// hash belongs to the redesign entry page (decision D1); explicit legacy
// hashes (bare #ask, #getting-started, …) keep loading the legacy console.
// Everything else that is shaped like a redesign route — multi-segment paths
// such as #user/apply, the #/-prefixed form, or an unknown surface/view —
// resolves inside the redesign, ending in its not-found page instead of
// falling through to the legacy console or silently resetting to the default
// view (round 4, #27).
export function parseRedesignHash(hash: string): ParsedRedesignHash | null {
  const raw = hash.startsWith("#") ? hash.slice(1) : hash;
  const queryIndex = raw.indexOf("?");
  const path = queryIndex === -1 ? raw : raw.slice(0, queryIndex);
  const query = queryIndex === -1 ? "" : raw.slice(queryIndex + 1);

  if (path === "/" || path === "") {
    return withCanonicalHash(raw, { surface: "entry" });
  }

  const slashPrefixed = path.startsWith("/");
  const stripped = path.replace(/^\//, "");
  const segments = stripped.split("/");
  if (segments.length === 1) {
    const [only] = segments;
    if (only === "user") {
      return withCanonicalHash(raw, { surface: "user", view: defaultUserView, params: {} });
    }
    if (only === "admin") {
      return withCanonicalHash(raw, { surface: "admin", view: defaultAdminView, params: {} });
    }
    // Bare single-segment hashes belong to the legacy console; the same
    // segment behind a slash is redesign territory and 404s.
    return slashPrefixed ? keepRawHash(raw, { surface: "notfound", attempted: path }) : null;
  }

  const [surface, requestedView = ""] = segments;
  if (surface === "user" || surface === "admin") {
    if (requestedView === "") {
      return surface === "user"
        ? withCanonicalHash(raw, { surface, view: defaultUserView, params: {} })
        : withCanonicalHash(raw, { surface, view: defaultAdminView, params: {} });
    }
    if (surface === "user" && isUserView(requestedView)) {
      return withCanonicalHash(raw, { surface, view: requestedView, params: normalizeParams(requestedView, new URLSearchParams(query)) });
    }
    if (surface === "admin" && isAdminView(requestedView)) {
      return withCanonicalHash(raw, { surface, view: requestedView, params: normalizeParams(requestedView, new URLSearchParams(query)) });
    }
    return keepRawHash(raw, { surface: "notfound", attempted: path });
  }
  return keepRawHash(raw, { surface: "notfound", attempted: path });
}

// A not-found route keeps the attempted hash in the address bar so the user
// (and support) can still read what went wrong.
function keepRawHash(raw: string, route: Extract<RedesignRoute, { surface: "notfound" }>): ParsedRedesignHash {
  return { canonicalHash: `#${raw}`, redirected: false, route };
}

export function isRedesignHash(hash: string): boolean {
  return parseRedesignHash(hash) !== null;
}

export function redesignHash(route: RedesignRoute): string {
  if (route.surface === "entry") {
    return entryHash;
  }
  if (route.surface === "notfound") {
    return `#${route.attempted}`;
  }
  const query = serializeParams(route.view, route.params);
  return `#${route.surface}/${route.view}${query ? `?${query}` : ""}`;
}

export function userHash(view: UserView, params: RouteParams = {}): string {
  return redesignHash({ surface: "user", view, params });
}

export function adminHash(view: AdminView, params: RouteParams = {}): string {
  return redesignHash({ surface: "admin", view, params });
}

export function surfaceHomeHash(surface: Surface): string {
  return surface === "user" ? userHash(defaultUserView) : adminHash(defaultAdminView);
}

export function otherSurface(surface: Surface): Surface {
  return surface === "user" ? "admin" : "user";
}

function withCanonicalHash(raw: string, route: RedesignRoute): ParsedRedesignHash {
  const canonicalHash = redesignHash(route);
  return { canonicalHash, redirected: canonicalHash !== `#${raw}`, route };
}

function normalizeParams(view: RedesignView, search: URLSearchParams): RouteParams {
  const params: RouteParams = {};
  for (const name of viewParams[view]) {
    const value = search.get(name)?.trim() ?? "";
    if (value && isAllowedValue(view, name, value)) {
      params[name] = value;
    }
  }
  return params;
}

function serializeParams(view: RedesignView, params: RouteParams): string {
  const pairs: string[] = [];
  for (const name of viewParams[view]) {
    const value = params[name]?.trim() ?? "";
    if (value && isAllowedValue(view, name, value)) {
      pairs.push(`${name}=${encodeParam(value)}`);
    }
  }
  return pairs.join("&");
}

function isAllowedValue(view: RedesignView, name: string, value: string): boolean {
  const allowed = enumeratedParams[view]?.[name];
  return !allowed || allowed.includes(value);
}

// ":", "@" and "/" are legal in a fragment; keeping them readable makes deep
// links such as subject=user:support-example shareable.
function encodeParam(value: string): string {
  return encodeURIComponent(value).replace(/%3A/gi, ":").replace(/%40/gi, "@").replace(/%2F/gi, "/");
}
