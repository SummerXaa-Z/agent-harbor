import { defineConfig } from "vitest/config";

// Component and hook tests run here under happy-dom; the plain
// tests/*.test.mjs suites stay on node --test (see the test script).
// The include list keeps the two runners from picking up each other's files.
export default defineConfig({
  test: {
    environment: "happy-dom",
    globals: true,
    include: ["tests/components/**/*.test.tsx", "tests/hooks/**/*.test.tsx"],
  },
});
