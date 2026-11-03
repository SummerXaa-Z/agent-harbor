import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useRedesignLanguage } from "../../src/redesign/hooks/useRedesignI18n";
import type { InitialI18n } from "../../src/redesign/hooks/useRedesignI18n";

const loadTranslator = vi.hoisted(() => vi.fn());
vi.mock("../../src/i18n", () => ({
  loadTranslator,
  resolveInitialLanguage: () => "en" as const,
}));

const enT = (key: string) => `en:${key}`;
const zhT = (key: string) => `zh:${key}`;

function mount(initial: InitialI18n = { language: "en", t: enT }) {
  return renderHook((props: InitialI18n) => useRedesignLanguage(props), { initialProps: initial });
}

beforeEach(() => {
  loadTranslator.mockReset();
  window.localStorage.clear();
});

describe("useRedesignLanguage", () => {
  it("renders straight from the preloaded initial translator without fetching", () => {
    const { result } = mount({ language: "zh-CN", t: zhT });
    expect(result.current.t("any.key")).toBe("zh:any.key");
    expect(result.current.language).toBe("zh-CN");
    expect(loadTranslator).not.toHaveBeenCalled();
    expect(window.localStorage.getItem("agent-harbor-language")).toBe("zh-CN");
    expect(document.documentElement.lang).toBe("zh-CN");
  });

  it("keeps the current translator until the new locale chunk resolves", async () => {
    let resolveZh: (t: (key: string) => string) => void = () => undefined;
    loadTranslator.mockImplementation(
      () => new Promise<(key: string) => string>((resolve) => { resolveZh = resolve; }),
    );
    const { result } = mount();
    act(() => result.current.setLanguage("zh-CN"));
    expect(result.current.language).toBe("zh-CN");
    expect(result.current.t("any.key")).toBe("en:any.key");

    await act(async () => { resolveZh(zhT); });
    expect(result.current.t("any.key")).toBe("zh:any.key");
    expect(loadTranslator).toHaveBeenCalledWith("zh-CN");
    expect(window.localStorage.getItem("agent-harbor-language")).toBe("zh-CN");
  });

  it("keeps the old translator when the locale chunk fails to load", async () => {
    loadTranslator.mockRejectedValue(new Error("chunk failed"));
    const { result } = mount();
    await act(async () => { result.current.setLanguage("zh-CN"); });
    expect(result.current.t("any.key")).toBe("en:any.key");
    expect(window.localStorage.getItem("agent-harbor-language")).toBe("zh-CN");
  });
});
