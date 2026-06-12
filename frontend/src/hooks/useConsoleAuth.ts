import { useEffect, useState } from "react";
import {
  fetchConsoleSession,
  loginConsole,
  logoutConsole
} from "../api";
import type { ConsoleSession } from "../types";

export interface ConsoleAuthMessage {
  key: string;
  params?: Record<string, string | number>;
}

interface ConsoleAuthState {
  loginKey: string;
  loginMessage: ConsoleAuthMessage | null;
  loginSubmitting: boolean;
  session: ConsoleSession | null;
  sessionLoading: boolean;
}

const initialConsoleAuthState: ConsoleAuthState = {
  loginKey: "",
  loginMessage: null,
  loginSubmitting: false,
  session: null,
  sessionLoading: true
};

export function useConsoleAuth() {
  const [state, setState] = useState<ConsoleAuthState>(initialConsoleAuthState);
  const accessReady = Boolean(state.session?.authenticated || state.session?.requiresLogin === false);

  useEffect(() => {
    let active = true;
    const controller = new AbortController();
    setState((current) => ({ ...current, sessionLoading: true }));
    fetchConsoleSession(controller.signal)
      .then((session) => {
        if (!active) return;
        setState((current) => ({ ...current, loginMessage: null, session }));
      })
      .catch((error) => {
        if (!active || isAbortError(error)) return;
        setState((current) => ({
          ...current,
          loginMessage: { key: "error.consoleSessionUnavailable" },
          session: { authenticated: false, requiresLogin: true }
        }));
      })
      .finally(() => {
        if (active) setState((current) => ({ ...current, sessionLoading: false }));
      });
    return () => {
      active = false;
      controller.abort();
    };
  }, []);

  async function login(onSuccess?: () => void) {
    const nextAdminKey = state.loginKey.trim();
    if (!nextAdminKey) {
      setState((current) => ({ ...current, loginMessage: { key: "message.consoleLoginRequired" } }));
      return;
    }
    setState((current) => ({ ...current, loginMessage: null, loginSubmitting: true }));
    try {
      const nextSession = await loginConsole(nextAdminKey);
      setState((current) => ({
        ...current,
        loginKey: "",
        loginMessage: { key: "message.consoleLoginSucceeded" },
        session: nextSession
      }));
      onSuccess?.();
    } catch {
      setState((current) => ({ ...current, loginMessage: { key: "error.consoleLoginFailed" } }));
    } finally {
      setState((current) => ({ ...current, loginSubmitting: false }));
    }
  }

  async function logout(onComplete?: (session: ConsoleSession) => void) {
    setState((current) => ({ ...current, loginMessage: null }));
    try {
      const nextSession = await logoutConsole();
      setState((current) => ({ ...current, session: nextSession }));
      onComplete?.(nextSession);
    } catch {
      setState((current) => ({ ...current, loginMessage: { key: "error.consoleLogoutFailed" } }));
    }
  }

  function setLoginKey(loginKey: string) {
    setState((current) => ({ ...current, loginKey }));
  }

  return {
    accessReady,
    login,
    logout,
    setLoginKey,
    ...state
  };
}

function isAbortError(error: unknown) {
  return error instanceof Error && error.name === "AbortError";
}
