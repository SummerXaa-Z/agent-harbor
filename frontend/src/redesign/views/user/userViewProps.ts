import type { ConsoleSession } from "../../../types";
import type { RedesignData } from "../../hooks/useRedesignData";
import type { RouteParams } from "../../router";

export interface UserViewProps {
  data: RedesignData;
  onRetry: () => void;
  params: RouteParams;
  session: ConsoleSession | null;
}
