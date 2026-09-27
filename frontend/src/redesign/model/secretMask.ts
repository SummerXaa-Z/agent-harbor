// Handoff snippets always reference the token through this placeholder; the
// real value never enters rendered text (handoff red line, D4).
export const AGENT_HARBOR_TOKEN_PLACEHOLDER = "${AGENT_HARBOR_TOKEN}";
export const bearerPlaceholderHeader = "Authorization: Bearer ${AGENT_HARBOR_TOKEN}";
export const subjectHeaderExample = "X-AgentHarbor-Subject-Id: user:support-example";

const maskedSuffix = "…";
// Agent keys are `ah_` + 43 base64url characters and admin keys `ahadm_` + 43;
// twelve or more key characters without the ellipsis means a full secret.
const unmaskedTokenPattern = /\b(?:ahadm|ah)_[A-Za-z0-9_-]{12,}(?![A-Za-z0-9_\-…])/gu;

// The UI only ever holds a stored prefix; the ellipsis marks the value as
// masked wherever it is shown.
export function maskSecret(value: string, visible = 12): string {
  const trimmed = value.trim();
  if (!trimmed) return "";
  return `${trimmed.slice(0, visible)}${maskedSuffix}`;
}

export function containsUnmaskedToken(text: string): boolean {
  unmaskedTokenPattern.lastIndex = 0;
  const found = unmaskedTokenPattern.test(text);
  unmaskedTokenPattern.lastIndex = 0;
  return found;
}

// A last line of defence for text that reaches a code block or log view: any
// full key is cut down to the prefix the backend stores for it (12 characters
// for agent keys, `ahadm_` + 8 for admin keys) before it can be displayed.
export function redactTokens(text: string): string {
  return text.replace(unmaskedTokenPattern, (token) =>
    maskSecret(token, token.startsWith("ahadm_") ? 14 : 12),
  );
}
