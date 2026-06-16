import assert from "node:assert/strict";
import test from "node:test";

import {
  createTranslator,
  translationKeys
} from "../src/i18n.ts";

test("visible production copy avoids evidence wording", () => {
  for (const language of ["en", "zh-CN"]) {
    const t = createTranslator(language);
    const offending = translationKeys(language)
      .map((key) => [key, t(key)])
      .filter(([, value]) => /证据|\bevidence\b/i.test(value));

    assert.deepEqual(offending, []);
  }
});
