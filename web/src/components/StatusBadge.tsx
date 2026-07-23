import type { CheckStatus, RunStatus } from "../api/types";

export function StatusBadge({ status }: { status: RunStatus | CheckStatus }) {
  return <span className={`status status-${status}`}>{status}</span>;
}
