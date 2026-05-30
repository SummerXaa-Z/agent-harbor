interface RetryFields {
  backoffMsText: string;
  maxAttemptsText: string;
}

export type RetryFieldResult =
  | {
      ok: true;
      requested: boolean;
      backoffMs: number;
      maxAttempts: number;
    }
  | {
      ok: false;
      message: string;
    };

export function parseRetryFields(fields: RetryFields): RetryFieldResult {
  const maxAttemptsText = fields.maxAttemptsText.trim();
  const backoffMsText = fields.backoffMsText.trim();
  const maxAttempts = Number(maxAttemptsText);
  const backoffMs = Number(backoffMsText);

  if (!maxAttemptsText || !Number.isInteger(maxAttempts) || maxAttempts < 1 || maxAttempts > 4) {
    return {
      ok: false,
      message: "Retry attempts must be an integer between 1 and 4."
    };
  }
  if (!backoffMsText || !Number.isInteger(backoffMs) || backoffMs < 0 || backoffMs > 1000) {
    return {
      ok: false,
      message: "Retry backoff must be an integer between 0 and 1000 ms."
    };
  }

  return {
    ok: true,
    requested: maxAttempts !== 1 || backoffMs !== 0,
    backoffMs,
    maxAttempts
  };
}
