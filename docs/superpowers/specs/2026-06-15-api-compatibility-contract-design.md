# API Compatibility Contract Design

## Goal

Prevent the web console from treating an old or incompatible AgentHarbor API as a random permission-journey failure.

中文目标：当控制台连接到旧版或能力不完整的 API 时，先在状态检查阶段明确提示升级 API，而不是让用户在权限变更、审批或应用阶段看到零散的 404 或业务错误。

## Design Consensus

The backend will expose a public, read-only `GET /api/v1/system/info` endpoint. It returns the control-plane name, a stable API compatibility version, and a list of backend capabilities that the current console depends on.

The frontend `checkApiHealth` flow will continue to verify `/healthz`, then verify `/api/v1/system/info`. If the endpoint is missing or required capabilities are absent, the readiness check reports an API contract failure with localized Chinese and English messages.

## Required Console Capabilities

- `permission_package_approval_requests`
- `permission_package_approval_withdraw`
- `permission_package_apply_preflight`
- `permission_package_applications`
- `permission_package_application_health`
- `permission_package_application_impact`
- `permission_package_production_readiness`
- `permission_package_consumed_approval_recovery`

## Non-Goals

- No backend mutation behavior changes.
- No database migration.
- No new dependency.
- No release publishing or repository visibility changes.

## Testing

- Backend unit test confirms the public system info endpoint returns a version and all required capabilities without an admin key.
- Frontend source tests confirm the console health check calls the system info endpoint and handles incompatible contracts.
- i18n tests confirm Chinese and English compatibility messages exist.
- Full gates remain `pnpm --dir frontend test`, `pnpm --dir frontend build`, `make check`, and `make release-check`.
