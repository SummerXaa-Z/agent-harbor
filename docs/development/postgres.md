# Local PostgreSQL setup / 本地 PostgreSQL 配置

This guide runs a PostgreSQL-backed AgentHarbor developer preview on one machine. It starts **only** PostgreSQL in Docker; AgentHarbor continues to run from the checked-out source on the host. It is suitable for trying persisted tenants, permissions, audit records, and encrypted Agent credentials.

本指南用于在一台机器上运行带 PostgreSQL 持久化存储的 AgentHarbor 开发预览环境。Docker 只启动 PostgreSQL；AgentHarbor 仍从宿主机的源码运行。它适合验证租户、授权、审计记录和加密后的 Agent 凭据在重启后仍然保留。

This is not a production deployment recipe. It does not configure HTTPS, backups, high availability, network isolation, secret distribution, or production administrator identities. For deployment-style configuration checks, use `make production-hardening` and the Runtime Configuration section in the [README](../../README.md#runtime-configuration).

这不是生产部署方案。它没有配置 HTTPS、备份、高可用、网络隔离、密钥分发或生产管理员身份。需要检查部署式配置时，请运行 `make production-hardening`，并阅读 [README 的运行时配置](../../README.md#runtime-configuration)。

## Prerequisites / 前置条件

- Docker Desktop or another Docker installation with the `docker compose` command.
- Go and the repository dependencies described in the [README](../../README.md#quick-start).
- `openssl` to generate a local credential-encryption key.

The checked-in Compose file uses PostgreSQL 16, the same major version as the repository CI service. Its username, password, database name, and exposed `5432` port are intentionally local defaults; do not reuse them outside a disposable developer environment.

仓库内的 Compose 文件使用 PostgreSQL 16，与 CI 服务的主版本一致。用户名、密码、数据库名和暴露的 `5432` 端口仅是本地默认值；不要在可共享或生产环境中复用。

## 1. Start the local database / 启动本地数据库

From the repository root:

```bash
docker compose -f docs/development/postgres.compose.yaml up -d
docker compose -f docs/development/postgres.compose.yaml ps
```

Wait until the `postgres` service is healthy. Docker stores the data in the named `agent-harbor-postgres` volume, so a normal container restart does not reset it.

等待 `postgres` 服务变为 healthy。Docker 会把数据保存在命名卷 `agent-harbor-postgres` 中，因此普通的容器重启不会清空数据。

## 2. Configure a persisted local run / 配置持久化本地运行

Generate a credential key once and keep the same value while this database contains Agent credentials. The application accepts 32 random raw bytes or base64-encoded 32-byte values; the command below produces the latter.

为凭据密钥生成一次随机值，并在该数据库仍保存 Agent 凭据时保持不变。应用接受 32 字节随机原始值或其 base64 编码；下面的命令生成后者。

```bash
export AGENT_HARBOR_CREDENTIAL_KEY="$(openssl rand -base64 32)"
export AGENT_HARBOR_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable'

go run ./cmd/agent-harbor
```

Do not commit the generated key, put it in an example file, or replace it casually after credentials have been saved. AgentHarbor encrypts persisted Agent credentials with this key; changing it without a tested re-encryption plan can prevent stored credentials from being read. Missing or weak keys are rejected when `AGENT_HARBOR_DATABASE_URL` is set.

不要提交生成的 key、不要把它写入示例文件，也不要在已经保存凭据后随意替换它。AgentHarbor 使用这个 key 加密持久化的 Agent 凭据；没有经过验证的重加密方案就更换 key，可能导致无法读取已有凭据。设置了 `AGENT_HARBOR_DATABASE_URL` 后，缺失或弱 key 会被拒绝。

For a repeatable local shell, copy the relevant values into an untracked environment file based on [`.env.example`](../../.env.example), then load it with your preferred local secret workflow.

若要重复使用本地环境，请以 [`.env.example`](../../.env.example) 为基础，将这些值写入未跟踪的环境文件，再用你惯用的本地密钥加载方式导入。

## 3. Understand migrations / 了解迁移行为

There is no separate migration command to run. On startup, AgentHarbor connects to the configured database and automatically applies the embedded SQL files from `internal/db/migrations/` in filename order. Each migration is recorded in `schema_migrations` and runs in a database transaction, so a recorded migration is not run again on the next start.

不需要单独执行迁移命令。启动时，AgentHarbor 会连接已配置的数据库，并按文件名顺序自动执行 `internal/db/migrations/` 内嵌的 SQL 文件。每条迁移都会记录在 `schema_migrations` 中，并在数据库事务里执行；已记录的迁移不会在下次启动时重复执行。

After the application starts, you can inspect the recorded migrations without exposing application credentials:

应用启动后，可以查看已记录的迁移，而无需暴露应用凭据：

```bash
docker compose -f docs/development/postgres.compose.yaml exec -T postgres \
  psql -U agent_harbor -d agent_harbor -c 'select version, applied_at from schema_migrations order by version;'
```

If startup fails during a migration, stop and review the error before retrying. Do not manually mark a migration as applied or edit an already-applied migration in a database you want to keep.

如果启动在迁移过程中失败，请先查看错误再重试。不要在需要保留的数据库里手动把迁移标记为已完成，也不要修改已经执行过的迁移文件。

## 4. Run PostgreSQL integration tests / 运行 PostgreSQL 集成测试

Use a disposable local database for the integration suite. The tests run against the exact URL you provide, apply migrations if needed, and create test data; never point this variable at shared or production data.

集成测试应使用可丢弃的本地数据库。测试会连接你提供的确切 URL、按需执行迁移并创建测试数据；绝不要把这个变量指向共享或生产数据。

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  make test-postgres
```

`make check` does not require Docker or PostgreSQL. The repository CI runs the PostgreSQL integration job separately, while the local target remains opt-in.

`make check` 不需要 Docker 或 PostgreSQL。仓库 CI 会单独运行 PostgreSQL 集成任务；本地目标仍按需执行。

## 5. Stop or reset the local database / 停止或重置本地数据库

Stop the container but retain the data volume:

停止容器但保留数据卷：

```bash
docker compose -f docs/development/postgres.compose.yaml down
```

To permanently delete this local database and every persisted record in its named volume, use the following command only for this disposable developer setup:

如需永久删除该本地数据库及命名卷中的全部持久化记录，只能对这个可丢弃的开发环境使用以下命令：

```bash
docker compose -f docs/development/postgres.compose.yaml down -v
```
