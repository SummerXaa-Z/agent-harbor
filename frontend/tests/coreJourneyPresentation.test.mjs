import assert from "node:assert/strict";
import test from "node:test";

import { coreJourneyStepDetailLabel } from "../src/coreJourneyPresentation.ts";
import { createTranslator } from "../src/i18n.ts";

test("core journey step details use operator-readable copy instead of raw ids", () => {
  const t = createTranslator("zh-CN");
  const detail = coreJourneyStepDetailLabel(
    {
      detail: "tenant-root-ui-core-demo -> tenant-child-ui-core-demo -> tenant-grandchild-ui-core-demo",
      key: "tenantTree",
      metric: "0",
      status: "missing"
    },
    t
  );

  assert.equal(detail, "等待创建三级租户范围。");
  assert.doesNotMatch(detail, /tenant-/);
});

test("core journey step details describe the current state for each stage", () => {
  const t = createTranslator("en");

  assert.equal(
    coreJourneyStepDetailLabel(
      { detail: "http://127.0.0.1:8787/mcp", key: "agentPair", metric: "0/0", status: "missing" },
      t
    ),
    "Waiting to register the caller and target MCP service."
  );
  assert.equal(
    coreJourneyStepDetailLabel(
      { detail: "search_customer scoped, export_contracts unassigned", key: "capabilityDiscovery", metric: "2/2", status: "complete" },
      t
    ),
    "Allowed tool is scoped; blocked tool remains unassigned."
  );
  assert.equal(
    coreJourneyStepDetailLabel(
      { detail: "ui-core-demo allowed=1 denied=0", key: "runtimeEvidence", metric: "1/0", status: "partial" },
      t
    ),
    "Only part of the runtime decision has been captured."
  );
});
