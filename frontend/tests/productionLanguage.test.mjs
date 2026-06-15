import assert from "node:assert/strict";
import test from "node:test";

import {
  createTranslator,
  translationKeys
} from "../src/i18n.ts";

test("visible production copy avoids evidence wording", () => {
  const allowedKeyFragments = [
    "exportProductionEvidence",
    "productionEvidence",
    "evidenceLayer"
  ];

  for (const language of ["en", "zh-CN"]) {
    const t = createTranslator(language);
    const offending = translationKeys(language)
      .filter((key) => !allowedKeyFragments.some((fragment) => key.includes(fragment)))
      .map((key) => [key, t(key)])
      .filter(([, value]) => /证据|\bevidence\b/i.test(value));

    assert.deepEqual(offending, []);
  }
});
