# Management MCP Compatibility Aliases

Date: 2026-07-08

This engineering note documents legacy Management MCP tool names kept for old clients. Primary product docs, quickstarts, and new admin-agent flows should use the preferred tool names.

| Legacy alias | Preferred tool | Status |
| --- | --- | --- |
| `export_permission_package_production_evidence` | `export_permission_package_production_report` | `compatibility_alias` |

Rules:

- Keep aliases in `tools/list` with `lifecycle.status: "compatibility_alias"` and `preferredName`.
- New clients should call preferred tools.
- Keep aliases out of primary product copy, quickstarts, and operator-facing examples.
- Treat this file as the compatibility map when old clients need migration help.

## 中文说明

这份工程说明只用于旧客户端兼容排查。主 README、产品旅程、快速开始和新的管理 Agent 示例都应展示首选工具名。

规则：

- `tools/list` 继续返回旧别名，并带上 `lifecycle.status: "compatibility_alias"` 与 `preferredName`。
- 新客户端应调用首选工具。
- 主产品文案、快速开始和操作员示例不展示旧别名。
- 旧客户端迁移时，以这份文件作为兼容映射表。
