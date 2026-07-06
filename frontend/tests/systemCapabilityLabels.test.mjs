import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import test from "node:test";

import { createTranslator } from "../src/i18n.ts";
import { requiredConsoleCapabilities } from "../src/systemInfoContract.ts";

const moduleUrl = new URL("../src/systemCapabilityLabels.ts", import.meta.url);
const connectionDiagnosticsSource = readFileSync(new URL("../src/connectionDiagnostics.ts", import.meta.url), "utf8");
const healthCheckPresentationSource = readFileSync(new URL("../src/healthCheckPresentation.ts", import.meta.url), "utf8");

test("system capability labels cover every required console capability", async () => {
  assert.equal(existsSync(moduleUrl), true, "systemCapabilityLabels.ts centralizes capability label mappings");

  const { systemCapabilityLabelKeyByName } = await import("../src/systemCapabilityLabels.ts");
  const missing = requiredConsoleCapabilities.filter((capability) => !systemCapabilityLabelKeyByName[capability]);

  assert.deepEqual(missing, []);

  for (const language of ["en", "zh-CN"]) {
    const t = createTranslator(language);
    for (const capability of requiredConsoleCapabilities) {
      const key = systemCapabilityLabelKeyByName[capability];
      assert.notEqual(t(key), key, `${language} translation is missing for ${capability}`);
    }
  }
});

test("system capability label mapping is shared by diagnostics and health presentation", () => {
  assert.match(connectionDiagnosticsSource, /from ['"]\.\/systemCapabilityLabels\.ts['"]/);
  assert.match(healthCheckPresentationSource, /from ['"]\.\/systemCapabilityLabels\.ts['"]/);
  assert.doesNotMatch(connectionDiagnosticsSource, /const systemCapabilityLabelKeyByName/);
  assert.doesNotMatch(healthCheckPresentationSource, /const systemCapabilityLabelKeyByName/);
});
