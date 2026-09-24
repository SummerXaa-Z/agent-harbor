import type { Translator } from "./consolePresenters";
import type { Language } from "./i18n";

export type LocalizedMessage =
  | {
      key: string;
      params?: Record<string, string | number>;
    }
  | {
      render: (t: Translator, language: Language) => string;
    };

export function localizedMessageText(message: LocalizedMessage | null, t: Translator, language: Language) {
  if (!message) return "";
  if ("render" in message) return message.render(t, language);
  return message.params ? tx(t, message.key, message.params) : t(message.key);
}

export function localizedErrorMessageState(error: unknown, fallbackKey: string): LocalizedMessage {
  return {
    render: (t, language) => localizedErrorMessage(t, language, error, fallbackKey)
  };
}

export function localizedErrorMessage(t: Translator, language: Language, error: unknown, fallbackKey: string) {
  const fallback = t(fallbackKey);
  if (!(error instanceof Error) || !error.message.trim()) return fallback;
  if (language === "en" || /[\u4e00-\u9fa5]/.test(error.message)) {
    return error.message;
  }
  return fallback;
}

const upstreamErrorCodeKeys: Record<string, string> = {
  UPSTREAM_CONNECT_ERROR: "error.upstream.connect",
  UPSTREAM_DNS_ERROR: "error.upstream.dns",
  UPSTREAM_ERROR: "error.upstream.generic",
  UPSTREAM_TIMEOUT: "error.upstream.timeout",
  UPSTREAM_TLS_ERROR: "error.upstream.tls"
};

export function localizedUpstreamErrorMessageState(error: unknown, fallbackKey: string): LocalizedMessage {
  return {
    render: (t, language) => localizedUpstreamErrorMessage(t, language, error, fallbackKey)
  };
}

// Swallowing the API error behind the fallback key hides exactly the detail an
// administrator needs (connection refused vs DNS vs TLS), so upstream failures
// render the localized classification plus the technical cause in parentheses.
export function localizedUpstreamErrorMessage(t: Translator, language: Language, error: unknown, fallbackKey: string) {
  const fallback = localizedErrorMessage(t, language, error, fallbackKey);
  if (!(error instanceof Error) || language === "en") return fallback;
  const code = typeof (error as { code?: unknown }).code === "string"
    ? (error as { code?: unknown }).code as string
    : "";
  const key = code ? upstreamErrorCodeKeys[code] : undefined;
  if (!key) return fallback;
  const detail = error.message.trim();
  return detail
    ? `${t(fallbackKey)}${t(key)}\uff08${detail}\uff09`
    : `${t(fallbackKey)}${t(key)}`;
}

export function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}
