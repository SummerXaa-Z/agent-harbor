# Connection Diagnostics Design

## Goal

Give production operators one clear place to understand whether AgentHarbor is ready for the primary permission journey.

中文目标：管理员打开控制台后，可以在连接设置里一次性确认登录会话、API 兼容、实时数据和 MCP 工具服务是否可用，而不是分别进入权限变更、系统自检和资源页面排查。

## Design Consensus

The existing global Connection popover remains the right location because it already owns API source, session, and technical override context. This increment adds an explicit **Run diagnostics** action to that popover, with async diagnostic state isolated in a small `useConnectionDiagnostics` hook instead of adding more state cells to `ConsoleController`.

Diagnostics will check four production-readiness inputs:

- Console session: the browser is authenticated, or local development explicitly does not require login.
- API compatibility: reuse `checkApiHealth`, including the `/api/v1/system/info` capability contract.
- Live data source: current console data is loaded from API rather than fallback samples.
- MCP tool service: run the existing MCP health check against the current permission-journey endpoint.

Backend system info will also expose a safe `authRequired` boolean so tools and future UI can distinguish production-login and local-development modes without guessing.

## Non-Goals

- No new backend mutation endpoint.
- No persistent diagnostics history.
- No secret, admin key, database URL, or private upstream disclosure.
- No replacement for the lower-level System Self-Check page.

## Error Handling

The diagnostic panel shows each row independently. API compatibility failures reuse the localized compatibility messages from the prior increment. Session failure tells the operator to sign in again. Sample-data mode is a warning, not a crash, but blocks production journey confidence.

## Testing

- Backend test asserts `GET /api/v1/system/info` returns `authRequired=true` when admin auth is configured and `false` in unauthenticated local development mode.
- Frontend pure model tests assert diagnostic row status and summary rules.
- Source/UI tests assert the connection popover contains the diagnostics action and result list.
- i18n tests assert English and Simplified Chinese diagnostic copy.
