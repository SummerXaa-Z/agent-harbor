# CI Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a GitHub Actions CI workflow that verifies backend, PostgreSQL integration, and frontend changes on pull requests and pushes.

**Architecture:** Use one workflow with three independent jobs: backend Go verification, PostgreSQL-backed store integration tests, and frontend pnpm verification. Keep the workflow small and explicit so PR #1 gets a reliable baseline without introducing release/deploy behavior.

**Tech Stack:** GitHub Actions, Go 1.25 from `go.mod`, PostgreSQL 16 service container, Node 24, pnpm 10.

---

### Task 1: Add CI Workflow

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Create workflow**

Create `.github/workflows/ci.yml` with:

```yaml
name: CI

on:
  pull_request:
  push:
    branches:
      - main
      - "codex/**"

permissions:
  contents: read

jobs:
  backend:
    name: Backend
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v6
      - name: Set up Go
        uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache-dependency-path: go.sum
      - name: Test
        run: go test ./...
      - name: Vet
        run: go vet ./...
      - name: Build
        run: go build ./...

  postgres:
    name: PostgreSQL integration
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_USER: agent_harbor
          POSTGRES_PASSWORD: agent_harbor
          POSTGRES_DB: agent_harbor
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U agent_harbor -d agent_harbor"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    env:
      AGENT_HARBOR_TEST_DATABASE_URL: postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable
    steps:
      - name: Checkout
        uses: actions/checkout@v6
      - name: Set up Go
        uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache-dependency-path: go.sum
      - name: Store integration tests
        run: go test ./internal/store -count=1

  frontend:
    name: Frontend
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v6
      - name: Set up pnpm
        uses: pnpm/action-setup@v6
        with:
          version: 10
      - name: Set up Node
        uses: actions/setup-node@v6
        with:
          node-version: 24
          cache: pnpm
          cache-dependency-path: frontend/pnpm-lock.yaml
      - name: Install dependencies
        run: pnpm --dir frontend install --frozen-lockfile
      - name: Test
        run: pnpm --dir frontend test
      - name: Build
        run: pnpm --dir frontend build
```

- [ ] **Step 2: Verify locally**

Run:

```bash
go test ./...
go vet ./...
go build ./...
pnpm --dir frontend test
pnpm --dir frontend build
git diff --check
```

Expected: PASS.

- [ ] **Step 3: Commit and push**

Run:

```bash
git add .github/workflows/ci.yml docs/superpowers/plans/2026-05-31-ci-workflow.md CHANGELOG.md
git commit -m "ci: add pull request verification workflow"
git push
```

Expected: Draft PR #1 updates and GitHub Actions checks appear.
