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

export function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}
