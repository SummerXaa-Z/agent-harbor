import type { Translator } from "../../consolePresenters.ts";
import type { Language } from "../../i18n.ts";
import { localizedApiErrorMessage, localizedUpstreamErrorMessage } from "../../localizedMessages.ts";

export type ApiErrorCategory = "network" | "forbidden" | "notFound" | "unavailable" | "other";

export interface ApiErrorCopy {
  next: string;
  title: string;
}

export interface ApiErrorPresentation extends ApiErrorCopy {
  category: ApiErrorCategory;
  detail: string;
}

const networkMessagePattern = /failed to fetch|load failed|networkerror|network request failed/i;

// Duck-typed on `status` / `code` so node tests can classify plain objects;
// api.ts is not importable there, and ApiRequestError carries both fields.
export function apiErrorCategory(error: unknown): ApiErrorCategory {
  if (error instanceof TypeError && networkMessagePattern.test(error.message)) return "network";
  const status = numericField(error, "status");
  const code = stringField(error, "code");
  if (status === 401 || status === 403) return "forbidden";
  if (status === 404) return "notFound";
  if (status === 502 || status === 503 || status === 504 || code.startsWith("UPSTREAM_")) return "unavailable";
  return "other";
}

export function apiErrorCopy(t: Translator, category: ApiErrorCategory): ApiErrorCopy {
  return {
    next: t(`rd.error.${category}.next`),
    title: t(`rd.error.${category}.title`),
  };
}

// The category decides the headline and the next step; the detail keeps the
// server's code and cause so an operator can still tell DNS from TLS failures.
export function apiErrorPresentation(
  t: Translator,
  language: Language,
  error: unknown,
  fallbackKey: string,
): ApiErrorPresentation {
  const category = apiErrorCategory(error);
  const detail =
    category === "unavailable"
      ? localizedUpstreamErrorMessage(t, language, error, fallbackKey)
      : localizedApiErrorMessage(t, language, error, fallbackKey);
  return { category, detail, ...apiErrorCopy(t, category) };
}

function numericField(value: unknown, name: string): number | undefined {
  if (!value || typeof value !== "object") return undefined;
  const field = (value as Record<string, unknown>)[name];
  return typeof field === "number" ? field : undefined;
}

function stringField(value: unknown, name: string): string {
  if (!value || typeof value !== "object") return "";
  const field = (value as Record<string, unknown>)[name];
  return typeof field === "string" ? field : "";
}
