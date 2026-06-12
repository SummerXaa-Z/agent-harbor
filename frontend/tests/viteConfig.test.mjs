import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const viteConfig = readFileSync(new URL("../vite.config.ts", import.meta.url), "utf8");

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
