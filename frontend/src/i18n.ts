export type Language = "en" | "zh-CN"

export type Messages = Record<string, string>

const fallbackLanguage: Language = "en"

export function normalizeLanguage(value?: string | null): Language {
  const normalized = value?.toLowerCase()
  if (normalized?.startsWith("zh")) {
    return "zh-CN"
  }
  if (normalized?.startsWith("en")) {
    return "en"
  }
  return fallbackLanguage
}

export function resolveInitialLanguage(storedLanguage?: string | null, browserLanguages: string[] = []): Language {
  if (storedLanguage) {
    return normalizeLanguage(storedLanguage)
  }
  return normalizeLanguage(browserLanguages[0])
}

// Each locale lives in its own module (./i18n/en.ts, ./i18n/zh.ts) so the
// bundler ships them as separate on-demand chunks and the console loads only
// the active language. i18nSync.ts is the eager both-catalogs facade that
// exists for the node --test suites.
const catalogLoaders: Record<Language, () => Promise<{ catalog: Messages }>> = {
  en: () => import("./i18n/en"),
  "zh-CN": () => import("./i18n/zh"),
}

export function translatorFor(catalog: Messages) {
  return (key: string, fallback?: string): string => catalog[key] ?? fallback ?? key
}

// Loads only the requested locale. en/zh-CN key parity is pinned by the
// tests, so the translator no longer falls back across locales at runtime.
export function loadTranslator(language: Language) {
  return catalogLoaders[language]().then((module) => translatorFor(module.catalog))
}

// zh-CN copy for the built-in permission template summaries. The server
// catalog carries English summaries as data; the redesigned console shows the
// translation for the four built-in template ids and falls back to the raw
// summary for anything else (round 4, #35).
export const zhTemplateSummaries: Record<string, string> = {
  "analytics-sandbox": "允许沙箱分析的读取与执行能力，阻止写入、导出和生产管理操作。",
  "audit-readonly": "允许低风险的审计查阅读取，阻止变更、导出和受限数据。",
  "sales-readonly": "允许限定销售租户内的 CRM 读取，阻止导出、删除、管理操作和受限数据。",
  "support-ticket-triage": "允许工单读取和有界更新，阻止导出、删除和管理操作。",
};
