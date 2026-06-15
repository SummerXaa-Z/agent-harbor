# Connection Diagnostics Smoke Gate Design

## Goal

Make the release smoke gate verify that the served web console still exposes the production connection diagnostics contract.

中文目标：把“连接诊断”纳入发布验收，避免未来改动让主旅程前置检查只在源码测试里通过，却在实际启动的控制台中缺失。

## Design Consensus

The right scope is the existing `scripts/scenario-web-console-production-journey.sh` gate. It already starts an isolated API, official SDK MCP demo service, and Vite console. This increment extends that gate with lightweight checks instead of adding a browser automation dependency.

The smoke gate should verify three things:

- The running API exposes `authRequired=true` in the production-journey smoke's fail-closed management-auth posture through `GET /api/v1/system/info`.
- The served console source still contains the connection diagnostics action and result-list contract.
- The release smoke runs the focused connection diagnostics model test together with the existing production journey route tests.

## Non-Goals

- No Playwright or browser automation dependency in release checks.
- No new backend endpoint.
- No persistent diagnostics history.
- No UI redesign.

## Pressure Test

- **Why not click the button in release-check?** The repository has intentionally kept the web-console production smoke dependency-free. Source-route smoke plus unit tests gives stable coverage without adding flaky browser setup.
- **Why check `authRequired=true`?** The smoke API runs in a production-shaped fail-closed management posture, not demo bypass mode. If the field disappears or flips unexpectedly, the connection diagnostics panel can no longer distinguish production login requirements from explicit local development bypass.
- **Why include the model test in the scenario?** It keeps the diagnostic row rules on the same release path as the production journey smoke, not only in the broader frontend test suite.

## Testing

- Add source-level guard assertions to `tests/makefile_targets_test.sh` so the production smoke script must contain the diagnostics checks.
- Run `bash tests/makefile_targets_test.sh` before implementation and confirm it fails.
- Update `scripts/scenario-web-console-production-journey.sh`.
- Run the focused guard, the scenario with isolated ports, `make check`, and `make release-check`.
