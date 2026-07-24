import type { CheckStatus, RunStatus, TargetStatus } from "../api/types";

type DisplayStatus = RunStatus | CheckStatus | TargetStatus | "skipped";

const statusMeta: Record<DisplayStatus, { symbol: string; label: string }> = {
  queued: { symbol: "○", label: "Queued" },
  running: { symbol: "↻", label: "Running" },
  completed: { symbol: "●", label: "Passed" },
  partial: { symbol: "△", label: "Warning" },
  failed: { symbol: "×", label: "Failed" },
  cancelled: { symbol: "—", label: "Cancelled" },
  interrupted: { symbol: "×", label: "Interrupted" },
  pending: { symbol: "○", label: "Pending" },
  passed: { symbol: "●", label: "Passed" },
  warning: { symbol: "△", label: "Warning" },
  skipped: { symbol: "○", label: "Skipped" },
  operational: { symbol: "●", label: "Operational" },
  unavailable: { symbol: "×", label: "Unavailable" },
  paused: { symbol: "○", label: "Paused" },
  maintenance: { symbol: "◇", label: "Maintenance" },
};

export function StatusBadge({ status }: { status: DisplayStatus }) {
  const meta = statusMeta[status];
  return (
    <span
      className={`status status-${status}`}
      aria-label={`Status: ${meta.label}`}
    >
      <span aria-hidden="true">{meta.symbol}</span>
      {meta.label}
    </span>
  );
}
