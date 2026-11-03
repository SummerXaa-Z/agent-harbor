import { translatorFor, type Language, type Messages } from "./i18n.ts";
import { catalog as enCatalog } from "./i18n/en.ts";
import { catalog as zhCatalog } from "./i18n/zh.ts";

// Eager both-catalogs facade for the node --test suites, which need
// synchronous translators. The browser bundle must never import this module:
// it would pull both locale chunks back into the eager graph.
const messages: Record<Language, Messages> = { en: enCatalog, "zh-CN": zhCatalog };

export function createTranslator(language: Language) {
  return translatorFor(messages[language]);
}

export function translationKeys(language: Language): string[] {
  return Object.keys(messages[language]).sort();
}
