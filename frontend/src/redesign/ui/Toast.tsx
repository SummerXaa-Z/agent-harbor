import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { ToneIcon } from "./icons";

export type ToastTone = "success" | "warning" | "danger";

interface ToastItem {
  id: number;
  message: string;
  tone: ToastTone;
}

type ShowToast = (message: string, tone?: ToastTone) => void;

const toastDurationMs = 2600;

const ToastContext = createContext<ShowToast>(() => undefined);

export function useToast(): ShowToast {
  return useContext(ToastContext);
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const nextIdRef = useRef(0);
  const timersRef = useRef(new Set<number>());

  useEffect(() => {
    const timers = timersRef.current;
    return () => {
      timers.forEach((timer) => window.clearTimeout(timer));
      timers.clear();
    };
  }, []);

  const showToast = useCallback<ShowToast>((message, tone = "success") => {
    const id = ++nextIdRef.current;
    setToasts((current) => [...current, { id, message, tone }]);
    const timer = window.setTimeout(() => {
      timersRef.current.delete(timer);
      setToasts((current) => current.filter((toast) => toast.id !== id));
    }, toastDurationMs);
    timersRef.current.add(timer);
  }, []);

  return (
    <ToastContext.Provider value={showToast}>
      {children}
      <div aria-live="polite" className="toast-wrap" role="status">
        {toasts.map((toast) => (
          <div className={`toast toast-${toast.tone}`} key={toast.id}>
            <ToneIcon tone={toast.tone} />
            <span>{toast.message}</span>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
