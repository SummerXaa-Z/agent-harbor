import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const viteConfig = readFileSync(new URL("../vite.config.ts", import.meta.url), "utf8");
const consoleController = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const packageJson = JSON.parse(readFileSync(new URL("../package.json", import.meta.url), "utf8"));
const lockfile = readFileSync(new URL("../pnpm-lock.yaml", import.meta.url), "utf8");

test("vite production build keeps the chunk warning budget visible", () => {
  assert.doesNotMatch(viteConfig, /chunkSizeWarningLimit\s*:/);
});

test("vite production build splits stable vendor dependencies", () => {
  assert.match(viteConfig, /rolldownOptions\s*:/);
  assert.match(viteConfig, /output\s*:/);
  assert.match(viteConfig, /react-vendor/);
  assert.match(viteConfig, /icons-vendor/);
  assert.match(viteConfig, /react\|react-dom/);
  assert.match(viteConfig, /lucide-react/);
});

test("frontend entry lazy-loads heavy workspace panels", () => {
  assert.match(consoleController, /lazy\(\(\) => import\("\.\/components\/AiAdminPermissionWorkbench"\)/);
  assert.match(consoleController, /lazy\(\(\) => import\("\.\/components\/CapabilityGovernanceView"\)/);
  assert.match(consoleController, /lazy\(\(\) => import\("\.\/components\/TenantOrganizationView"\)/);
  assert.match(consoleController, /<Suspense/);
});

test("vite esbuild transitive dependency is pinned to the patched line", () => {
  assert.equal(packageJson.pnpm?.overrides?.esbuild, "0.28.1");
  assert.doesNotMatch(lockfile, /esbuild@0\.27\.7/);
  assert.match(lockfile, /esbuild:\s+0\.28\.1/);
});
