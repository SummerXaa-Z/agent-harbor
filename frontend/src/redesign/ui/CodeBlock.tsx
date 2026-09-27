import { Copy } from "lucide-react";
import { useRedesignI18n } from "../hooks/useRedesignI18n";
import { redactTokens } from "../model/secretMask";
import { useToast } from "./Toast";

// Code blocks show and copy the same redacted text: handoff snippets carry
// `${AGENT_HARBOR_TOKEN}`, and any full key that slips in is cut to its prefix.
export function CodeBlock({ code, label }: { code: string; label: string }) {
  const { t } = useRedesignI18n();
  const showToast = useToast();
  const safeCode = redactTokens(code);

  async function copy() {
    try {
      await navigator.clipboard.writeText(safeCode);
      showToast(t("rd.common.copied"));
    } catch {
      showToast(t("rd.common.copyFailed"), "danger");
    }
  }

  return (
    <div className="codeblock-wrap">
      <pre aria-label={label} className="codeblock" tabIndex={0}>
        {safeCode.split("\n").map((line, index) => (
          <span className={line.trimStart().startsWith("#") ? "ck-cm" : undefined} key={index}>
            {line}
            {"\n"}
          </span>
        ))}
      </pre>
      <button className="copy-mini" onClick={() => void copy()} type="button">
        <Copy aria-hidden="true" size={13} />
        {t("rd.common.copy")}
      </button>
    </div>
  );
}
