import type { ConsoleSession } from "../../../types";
import type { RedesignData } from "../../hooks/useRedesignData";
import type { RouteParams } from "../../router";

export interface AdminViewProps {
  data: RedesignData;
  onRetry: () => void;
  params: RouteParams;
  session: ConsoleSession | null;
}
