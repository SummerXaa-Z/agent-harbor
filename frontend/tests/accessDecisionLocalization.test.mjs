import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import { accessTraceReasonLabel } from "../src/consolePresenters.ts";
import { createTranslator } from "../src/i18n.ts";
import { accessNextActionKeys } from "../src/askJourney.ts";

// Guard for the console message localization contract
// (docs/product/0.3.x-console-message-localization.md, Phase 2): the access
// decision explain API emits evidence with layer/status codes plus a
// messageKey contract; the frontend must keep localized copy for every one of
// them. Reading the Go source keeps this test honest when new evidence rows
// appear server-side.
const source = readFileSync(
  new URL("../../internal/httpapi/management_mcp.go", import.meta.url),
  "utf8"
);

function functionBody(marker) {
  const start = source.indexOf(marker);
  assert.ok(start >= 0, `${marker} not found in management_mcp.go`);
  const next = source.indexOf("\nfunc ", start + 1);
  return source.slice(start, next < 0 ? undefined : next);
}

const evidenceBody = functionBody("func (s *Server) managementMCPAccessEvidence");
const appendDecisionBody = functionBody("func appendDecisionEvidence");
const nextActionCodesBody = functionBody("func managementMCPAccessNextActionCodes");

function unique(matches) {
  return [...new Set(matches)];
}

function translatorResolves(key) {
  for (const language of ["en", "zh-CN"]) {
    const t = createTranslator(language);
    assert.notEqual(t(key), key, `${key} is missing ${language} copy`);
  }
}

test("every backend evidence message key has localized copy", () => {
  const literalKeys = unique([...evidenceBody.matchAll(/MessageKey: "([a-z_]+\.[a-z_]+)"/g)].map((m) => m[1]));
  assert.ok(literalKeys.length >= 9, `expected caller/target/capability keys, found ${literalKeys.length}`);
  const assignmentKeys = unique(
    [...evidenceBody.matchAll(/appendDecisionEvidence\(([^)]*)\)/g)]
      .flatMap((call) => [...call[1].matchAll(/"((?:tenant_entitlement|workspace_assignment|instance_assignment)\.(?:matched|blocked))"/g)].map((m) => m[1]))
  );
  const keys = [...literalKeys, ...assignmentKeys];
  assert.ok(keys.length >= 15, `expected a real evidence inventory, found ${keys.length}`);
  for (const key of keys) {
    translatorResolves(`ask.evidence.${key}`);
  }
});

test("every backend evidence layer has a localized node label", () => {
  const layers = unique([
    ...[...evidenceBody.matchAll(/Layer: "([a-z_]+)"/g)].map((m) => m[1]),
    ...[...evidenceBody.matchAll(/appendDecisionEvidence\(([^)]*)\)/g)].map((call) => call[1].match(/"([a-z_]+)"/)?.[1])
  ].filter(Boolean));
  assert.ok(layers.length >= 6, `expected a real layer inventory, found ${layers.length}`);
  for (const layer of layers) {
    translatorResolves(`ask.recordLayer.${layer}`);
  }
});

test("every evidence status has a localized label", () => {
  const statuses = unique([
    ...[...evidenceBody.matchAll(/Status: "([a-z_]+)"/g)].map((m) => m[1]),
    ...[...appendDecisionBody.matchAll(/"([a-z_]+)"/g)].map((m) => m[1])
  ]);
  assert.ok(statuses.length >= 6, `expected a real status inventory, found ${statuses.length}`);
  for (const status of statuses) {
    assert.match(status, /^[a-z_]+$/, `status literals only, got ${status}`);
    translatorResolves(`status.${status}`);
  }
});

test("every backend access next action code maps to localized copy", () => {
  const codes = unique([...nextActionCodesBody.matchAll(/\[\]string\{"([a-z_]+)"\}/g)].map((m) => m[1]));
  assert.ok(codes.length >= 9, `expected a real next-action inventory, found ${codes.length}`);
  for (const code of codes) {
    const key = accessNextActionKeys[code];
    assert.ok(key, `next action code ${code} has no copy mapping`);
    translatorResolves(key);
  }
});

test("the ask view resolves record messages by messageKey, not sentence text", () => {
  const askJourney = readFileSync(new URL("../src/askJourney.ts", import.meta.url), "utf8");
  assert.match(askJourney, /ask\.evidence\.\$\{row\.messageKey\}/);
  const askView = readFileSync(new URL("../src/components/AskAccessView.tsx", import.meta.url), "utf8");
  assert.match(askView, /accessNextActionLabelByCode\(result\.nextActionCodes\?\.\[index\]/);
});


test("every backend trace deny reason maps to localized runtime audit copy", () => {
  const storeSources = ["../../internal/store/memory.go", "../../internal/store/postgres.go", "../../internal/httpapi/access_profile.go"]
    .map((name) => readFileSync(new URL(name, import.meta.url), "utf8"))
    .join("\n");
  const reasons = unique([...storeSources.matchAll(/Reason: "([^"]+)"/g)].map((m) => m[1]));
  assert.ok(reasons.length >= 11, `expected a real deny-reason inventory, found ${reasons.length}`);
  for (const reason of reasons) {
    for (const language of ["en", "zh-CN"]) {
      const t = createTranslator(language);
      const label = accessTraceReasonLabel(reason, "deny", t);
      assert.notEqual(label, reason, `${reason} is not localized in ${language}`);
    }
  }
});

test("every backend management audit action, resource, and summary maps to localized copy", () => {
  const serverSource = readFileSync(new URL("../../internal/httpapi/server.go", import.meta.url), "utf8");
  const literalPairs = [
    ...serverSource.matchAll(/"([a-z_]+\.[a-z_]+)", "([a-z_]+)", [^,]+, "([^"]+)"/g)
  ].map((m) => ({ action: m[1], resource: m[2], summary: m[3] }));
  // The approval resolution handler writes its action/summary through variables.
  const variablePairs = [
    { action: "permission_package.approval_approved", resource: "permission_package_approval_request", summary: "Permission package approval approved" },
    { action: "permission_package.approval_rejected", resource: "permission_package_approval_request", summary: "Permission package approval rejected" }
  ];
  const pairs = [...literalPairs, ...variablePairs];
  assert.ok(pairs.length >= 20, `expected a real audit-event inventory, found ${pairs.length}`);
  for (const pair of pairs) {
    const summaryKey = pair.summary.trim().replaceAll(" ", "_").toLowerCase();
    for (const base of [`auditAction.${pair.action}`, `auditResource.${pair.resource}`, `auditSummary.${summaryKey}`]) {
      translatorResolves(base);
    }
  }
});
