import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import { StatusBadge } from "../components/StatusBadge";

function eventTime(value?: string) {
  if (!value) return "pending";
  return new Date(value).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function MonitoringPage() {
  const overview = useQuery({
    queryKey: ["monitoring-overview"],
    queryFn: api.monitoringOverview,
    refetchInterval: 5000,
  });
  const data = overview.data;

  return (
    <main className="page monitoring-page">
      <div className="page-heading">
        <div>
          <p className="section-label">Scheduled operations</p>
          <h1>Live monitoring</h1>
          <p>A chronological journal of target health and latency.</p>
        </div>
        <Link className="text-button" to="/targets">
          Manage targets
        </Link>
      </div>

      <div className="monitoring-summary">
        <span>{data?.activeTargets ?? 0} active targets</span>
        <span>{data?.warningTargets ?? 0} warning</span>
        <span>{data?.unavailableTargets ?? 0} unavailable</span>
        <i className={overview.isSuccess ? "is-live" : ""}>
          {overview.isSuccess ? "Live" : "Connecting"}
        </i>
      </div>

      {overview.error && (
        <div className="error-alert">
          <strong>Monitoring journal unavailable</strong>
          <span>{overview.error.message}</span>
        </div>
      )}

      <section className="operations-journal">
        <div className="operations-head">
          <span>Time</span>
          <span>Target</span>
          <span>Status</span>
          <span>Latency</span>
          <span>Context</span>
        </div>
        {data?.recentChecks.map((entry) => (
          <Link
            className="operation-row"
            to={`/targets/${entry.targetId}`}
            key={entry.id}
          >
            <time>{eventTime(entry.checkedAt ?? entry.createdAt)}</time>
            <div>
              <strong>{entry.targetName}</strong>
              <code>{entry.targetAddress}</code>
            </div>
            <StatusBadge status={entry.status} />
            <strong>
              {entry.latencyMs === undefined ? "—" : `${entry.latencyMs} ms`}
            </strong>
            <span>{entry.errorMessage || "Scheduled check completed"}</span>
          </Link>
        ))}
        {!overview.isLoading && data?.recentChecks.length === 0 && (
          <div className="empty-state">
            <div className="empty-copy">
              <p className="section-label">Journal is empty</p>
              <h3>Monitoring events appear after the first scheduled check</h3>
            </div>
            <Link className="text-button" to="/targets">
              Save a target
            </Link>
          </div>
        )}
      </section>
    </main>
  );
}
