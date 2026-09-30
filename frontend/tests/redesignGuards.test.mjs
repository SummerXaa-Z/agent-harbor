import assert from "node:assert/strict";
import { readdirSync, readFileSync } from "node:fs";
import test from "node:test";

import { createTranslator, translationKeys } from "../src/i18n.ts";
import { adminViews, userViews } from "../src/redesign/router.ts";
import { tableColumns } from "../src/redesign/model/tableColumns.ts";

const read = (path) => readFileSync(new URL(path, import.meta.url), "utf8");

const redesignRoot = new URL("../src/redesign/", import.meta.url);
const redesignSources = readdirSync(redesignRoot, { recursive: true })
  .filter((path) => /\.tsx?$/.test(path))
  .sort()
  .map((path) => ({ path: `redesign/${path}`, source: readFileSync(new URL(path, redesignRoot), "utf8") }));
const redesignTsx = redesignSources.filter(({ path }) => path.endsWith(".tsx"));

const css = read("../src/styles/redesign.css").replace(/\/\*[\s\S]*?\*\//g, "");
const tokenBlock = css.match(/^\s*([^{]+)\{([^}]*)\}/);
const cssAfterTokens = css.slice(tokenBlock.index + tokenBlock[0].length);

const rawColorPatterns = [/#[0-9a-fA-F]{3,8}\b/, /\brgba?\(/, /\bhsla?\(/, /(?<![\w-])(?:white|black)(?![\w-])/];
const cjkPattern = /[\u3000-\u303f\u3400-\u9fff\uf900-\ufaff\uff00-\uffef]/;

const rdKeys = (language) => translationKeys(language).filter((key) => key.startsWith("rd."));

function withoutWhitespace(value) {
  return value.replace(/\s+/g, "");
}

// Splits on commas that are not inside parentheses, e.g. `:where(a, button)`.
function splitSelectors(selectorText) {
  const selectors = [];
  let depth = 0;
  let current = "";
  for (const char of selectorText) {
    if (char === "(") depth += 1;
    if (char === ")") depth -= 1;
    if (char === "," && depth === 0) {
      selectors.push(current.trim());
      current = "";
    } else {
      current += char;
    }
  }
  selectors.push(current.trim());
  return selectors.filter(Boolean);
}

// Returns the full opening tag starting at `start`, skipping `>` inside JSX
// expressions and string literals (arrow functions, comparisons).
function openingTag(source, start) {
  let depth = 0;
  let quote = "";
  for (let index = start + 1; index < source.length; index += 1) {
    const char = source[index];
    if (quote) {
      if (char === quote && source[index - 1] !== "\\") quote = "";
      continue;
    }
    if (char === '"' || char === "'" || char === "`") quote = char;
    else if (char === "{") depth += 1;
    else if (char === "}") depth -= 1;
    else if (char === ">" && depth === 0) return source.slice(start, index + 1);
  }
  throw new Error(`unterminated tag at ${start}`);
}

function openingTags(source, name) {
  const tags = [];
  const pattern = new RegExp(`<${name}(?=[\\s>/])`, "g");
  for (const match of source.matchAll(pattern)) tags.push(openingTag(source, match.index));
  return tags;
}

function braceExpression(source, start) {
  let depth = 0;
  for (let index = start; index < source.length; index += 1) {
    if (source[index] === "{") depth += 1;
    if (source[index] === "}") {
      depth -= 1;
      if (depth === 0) return source.slice(start + 1, index);
    }
  }
  throw new Error(`unterminated expression at ${start}`);
}

// Static class tokens from className="..." and the string or template
// literals inside className={...}; comparison operands, method arguments and
// interpolated suffixes are skipped.
function staticClassTokens(source) {
  const literals = [];
  for (const match of source.matchAll(/className=("([^"]*)"|\{)/g)) {
    if (match[2] !== undefined) {
      literals.push(match[2]);
      continue;
    }
    const expression = braceExpression(source, match.index + "className=".length)
      .replace(/[!=]==?\s*(?:"[^"]*"|`[^`]*`)/g, "")
      .replace(/\.\w+\(\s*(?:"[^"]*"|`[^`]*`)\s*\)/g, "");
    for (const literal of expression.matchAll(/"([^"]*)"|`([^`]*)`/g)) literals.push(literal[1] ?? literal[2]);
  }
  return literals
    .flatMap((literal) => literal.split(/\s+/))
    .filter((token) => /^[A-Za-z][\w-]*[A-Za-z0-9]$/.test(token));
}

test("redesign tokens match the handoff values exactly", () => {
  assert.ok(splitSelectors(tokenBlock[1]).includes(".ah2"), "the first rule is the .ah2 token block");
  const declared = Object.fromEntries(
    [...tokenBlock[2].matchAll(/(--[\w-]+)\s*:\s*([^;]+);/g)].map((match) => [match[1], withoutWhitespace(match[2])]),
  );
  const expected = {
    "--bg": "#F6F8FB",
    "--card": "#FFFFFF",
    "--border": "#E6EAF1",
    "--border-strong": "#D8DEE9",
    "--primary": "#2456E6",
    "--primary-hover": "#1E47C4",
    "--primary-soft": "#EAF0FE",
    "--text": "#0F1B2D",
    "--text-2": "#4A5A70",
    "--text-3": "#8A97A8",
    "--success": "#16A36A",
    "--success-soft": "#E6F6EF",
    "--warning": "#D98324",
    "--warning-soft": "#FCF1E2",
    "--danger": "#D64545",
    "--danger-soft": "#FBEAEA",
    "--radius": "12px",
    "--radius-sm": "8px",
    "--sidebar-w": "232px",
    "--topbar-h": "58px",
    "--font": '-apple-system, BlinkMacSystemFont, "PingFang SC", "Microsoft YaHei", "Segoe UI", sans-serif',
    "--mono": '"SF Mono", Menlo, Consolas, monospace',
  };
  for (const [name, value] of Object.entries(expected)) {
    assert.equal(declared[name], withoutWhitespace(value), name);
  }
});

test("raw colors only appear in the redesign token block", () => {
  for (const pattern of rawColorPatterns) {
    assert.doesNotMatch(cssAfterTokens, pattern);
  }
});

test("every redesign rule and keyframe is scoped to .ah2", () => {
  const keyframes = [...css.matchAll(/@keyframes\s+([\w-]+)/g)].map((match) => match[1]);
  assert.ok(keyframes.length > 0);
  for (const name of keyframes) assert.match(name, /^ah2-/, name);

  const rules = css
    .replace(/@keyframes[^{]+\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}/g, "")
    .replace(/@media[^{]+\{/g, "");
  const allowedRoots = new Set(["body:has(.ah2)", ":root:has(.ah2)"]);
  for (const match of rules.matchAll(/([^{}]+)\{[^{}]*\}/g)) {
    for (const selector of splitSelectors(match[1])) {
      assert.ok(/^\.ah2(?![\w-])/.test(selector) || allowedRoots.has(selector), `unscoped selector: ${selector}`);
    }
  }
});

test("redesign sources keep copy in i18n and colors in tokens", () => {
  for (const { path, source } of redesignSources) {
    assert.doesNotMatch(source, cjkPattern, `${path} contains a CJK literal`);
  }
  for (const { path, source } of redesignTsx) {
    for (const pattern of rawColorPatterns) {
      assert.doesNotMatch(source, pattern, `${path} contains a raw color`);
    }
    assert.doesNotMatch(source, /style=\{\{[^}]*(?:color|background|border|fill|stroke)/i, `${path} inlines a color style`);
  }
});

test("redesign TSX only uses classes that redesign.css defines", () => {
  const defined = new Set([...css.matchAll(/\.([A-Za-z][\w-]*)/g)].map((match) => match[1]));
  for (const { path, source } of redesignTsx) {
    for (const token of staticClassTokens(source)) {
      assert.ok(defined.has(token), `${path} uses .${token}, which redesign.css does not define`);
    }
  }
});

test("tables take their columns only from tableColumns.ts", () => {
  for (const [tableId, columns] of Object.entries(tableColumns)) {
    assert.equal(columns.reduce((sum, column) => sum + column.width, 0), 100, tableId);
  }
  const table = read("../src/redesign/ui/Table.tsx");
  assert.match(table, /tableColumns\[tableId\]/);
  assert.doesNotMatch(table, /\bcolumns\??:/);
  for (const { path, source } of redesignTsx) {
    for (const tag of openingTags(source, "Table")) {
      assert.match(tag, /\btableId=/, `${path}: ${tag}`);
      assert.doesNotMatch(tag, /\bcolumns=/, `${path}: ${tag}`);
    }
  }
});

test("every button and link in the redesign has a real action", () => {
  let checked = 0;
  for (const { path, source } of redesignTsx) {
    for (const tag of openingTags(source, "button")) {
      checked += 1;
      assert.ok(/\bonClick=/.test(tag) || /\btype="submit"/.test(tag), `${path}: action-less ${tag}`);
    }
    for (const tag of openingTags(source, "Button")) {
      checked += 1;
      assert.ok(/\b(?:onClick|href)=/.test(tag) || /\btype="submit"/.test(tag), `${path}: action-less ${tag}`);
    }
    for (const tag of openingTags(source, "a")) {
      checked += 1;
      assert.match(tag, /\bhref=/, `${path}: ${tag}`);
    }
    assert.doesNotMatch(source, /onClick=\{\(\)\s*=>\s*(?:\{\s*\}|undefined|null)\}/, `${path} has an empty click handler`);
  }
  assert.ok(checked > 10);
});

test("redesign never renders secrets or real bearer values", () => {
  const i18nValues = ["en", "zh-CN"].flatMap((language) => {
    const t = createTranslator(language);
    return rdKeys(language).map((key) => ({ path: `i18n ${language} ${key}`, source: t(key) }));
  });
  for (const { path, source } of [...redesignSources, ...i18nValues]) {
    assert.doesNotMatch(source, /\b(?:response|result|created|token)\??\.key\b/, `${path} reads a one-time key`);
    assert.doesNotMatch(source, /Bearer (?!\$\{AGENT_HARBOR_TOKEN\})/, `${path} writes a bearer value`);
    assert.doesNotMatch(source, /\b(?:ahadm|ah)_[A-Za-z0-9_-]{12,}(?![A-Za-z0-9_\-…])/u, `${path} contains a full key`);
  }
});

test("the legacy console is fully retired (D1)", () => {
  const main = read("../src/main.tsx");
  const rootEntry = read("../src/RootEntry.tsx");
  const redesignApp = read("../src/redesign/RedesignApp.tsx");

  assert.match(main, /<RootEntry \/>/);
  assert.doesNotMatch(main, /["']\.\/App["']|styles\.css/);
  assert.match(rootEntry, /import RedesignApp from "\.\/redesign\/RedesignApp";/);
  assert.match(rootEntry, /<RedesignApp hash=\{hash\} \/>/);
  assert.match(redesignApp, /import "\.\.\/styles\/redesign\.css";/);

  // No legacy entry, controller, or stylesheet may creep back into src/.
  const retired = ["App.tsx", "LegacyEntry.tsx", "ConsoleController.tsx", "styles.css", "permission-workbench.css"];
  const srcEntries = readdirSync(new URL("../src", import.meta.url), { recursive: true }).map(String);
  for (const name of retired) {
    assert.ok(!srcEntries.some((entry) => entry === name || entry.endsWith(`/${name}`)), `${name} still exists under src/`);
  }
  assert.ok(!srcEntries.some((entry) => entry === "components" || entry.startsWith("components/")), "src/components should not exist");

  for (const { path, source } of redesignSources) {
    assert.doesNotMatch(source, /["'](?:\.\.\/)+styles(?:\/permission-workbench)?\.css["']/, `${path} imports legacy styles`);
    assert.doesNotMatch(source, /from ["'](?:\.\.\/)+(?:App|ConsoleController|LegacyEntry)["']/, `${path} imports the legacy tree`);
  }
});

test("refresh only reports success after live data arrived", () => {
  const redesignApp = read("../src/redesign/RedesignApp.tsx");
  assert.match(redesignApp, /const live = await data\.reload\(\);/);
  assert.match(redesignApp, /live \? t\("rd\.toast\.refreshed"\) : t\("rd\.toast\.refreshFailed"\)/);
  const hook = read("../src/redesign/hooks/useRedesignData.ts");
  assert.match(hook, /return dataResult\.status === "fulfilled" && dataResult\.value\.loadedFromApi;/);
});

test("every rd.* key used by the redesign exists in both languages", () => {
  const en = new Set(translationKeys("en"));
  const zh = new Set(translationKeys("zh-CN"));
  const used = new Set();
  for (const { source } of redesignSources) {
    for (const match of source.matchAll(/["'`](rd\.[A-Za-z0-9_.]+[A-Za-z0-9_])["'`]/g)) used.add(match[1]);
  }
  for (const view of [...userViews, ...adminViews]) used.add(`rd.page.${view}.desc`);
  for (const surface of ["user", "admin"]) {
    used.add(`rd.surface.${surface}`);
    for (const suffix of ["desc", "item1", "item2", "item3", "item4", "go"]) used.add(`rd.entry.${surface}.${suffix}`);
  }
  for (const state of ["done", "current", "blocked", "skipped", "pending"]) used.add(`rd.step.${state}`);
  assert.ok(used.size > 40);
  for (const key of used) {
    assert.ok(en.has(key), `missing en ${key}`);
    assert.ok(zh.has(key), `missing zh-CN ${key}`);
  }
  assert.deepEqual(rdKeys("zh-CN"), rdKeys("en"));
});
