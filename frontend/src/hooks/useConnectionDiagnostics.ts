import { useMemo, useState } from "react";
import {
  checkApiHealth,
  checkMockMcpHealth,
  defaultMockMcpHealthUrl,
  fetchConsoleSession
} from "../api";
import {
  buildConnectionDiagnosticRows,
  connectionDiagnosticsSummaryStatus,
  type ConnectionDiagnosticRow,
  type ConnectionDiagnosticStatus
} from "../connectionDiagnostics";

interface UseConnectionDiagnosticsArgs {
  liveDataLoaded: boolean;
  loadError: string;
  mcpEndpoint: string;
}

export function useConnectionDiagnostics({
  liveDataLoaded,
  loadError,
  mcpEndpoint
}: UseConnectionDiagnosticsArgs) {
  const [checking, setChecking] = useState(false);
  const [rows, setRows] = useState<ConnectionDiagnosticRow[]>([]);
  const [checkedAt, setCheckedAt] = useState<Date | null>(null);
  const status = useMemo<ConnectionDiagnosticStatus | null>(() => {
    return rows.length > 0 ? connectionDiagnosticsSummaryStatus(rows) : null;
  }, [rows]);

  function reset() {
    setRows([]);
    setCheckedAt(null);
  }

  async function run() {
    setChecking(true);
    try {
      const [session, apiHealth, mcpHealth] = await Promise.all([
        fetchConsoleSession().catch(() => null),
        checkApiHealth(),
        checkMockMcpHealth(mockMcpHealthUrlFromEndpoint(mcpEndpoint))
      ]);
      setRows(buildConnectionDiagnosticRows({
        apiHealth,
        liveDataLoaded,
        loadError,
        mcpHealth,
        session
      }));
      setCheckedAt(new Date());
    } finally {
      setChecking(false);
    }
  }

  return {
    checkedAt,
    checking,
    reset,
    rows,
    run,
    status
  };
}

function mockMcpHealthUrlFromEndpoint(endpointValue: string) {
  try {
    const endpointUrl = new URL(endpointValue);
    endpointUrl.pathname = "/healthz";
    endpointUrl.search = "";
    endpointUrl.hash = "";
    return endpointUrl.toString();
  } catch {
    return defaultMockMcpHealthUrl;
  }
}
