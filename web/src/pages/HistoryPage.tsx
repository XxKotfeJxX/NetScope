import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import type { DiagnosticRun } from "../api/types";
import { api } from "../api/client";
import { StatusBadge } from "../components/StatusBadge";

function dayKey(value: string) {
  return new Date(value).toLocaleDateString("en-CA");
}

function dayLabel(value: string) {
  const date = new Date(value);
  const today = new Date();
  const yesterday = new Date();
  yesterday.setDate(today.getDate() - 1);
  if (dayKey(value) === dayKey(today.toISOString())) return "Today";
  if (dayKey(value) === dayKey(yesterday.toISOString())) return "Yesterday";
  return date.toLocaleDateString([], {
    weekday: "long",
    month: "long",
    day: "numeric",
  });
}

function durationFor(run: DiagnosticRun) {
  if (run.startedAt && run.completedAt) {
    return Math.max(
      new Date(run.completedAt).getTime() - new Date(run.startedAt).getTime(),
      0,
    );
  }
  return Math.max(...run.results.map((result) => result.durationMs), 0);
}

export function HistoryPage() {
  const runs = useQuery({
    queryKey: ["runs", 1, 50],
    queryFn: () => api.listRuns(1, 50),
  });

  const groups = (runs.data?.items ?? []).reduce<
    Array<{ key: string; label: string; runs: DiagnosticRun[] }>
  >((result, run) => {
    const key = dayKey(run.createdAt);
    const group = result.find((item) => item.key === key);
    if (group) group.runs.push(run);
    else
      result.push({
        key,
        label: dayLabel(run.createdAt),
        runs: [run],
      });
    return result;
  }, []);

  return (
    <main className="page history-page">
      <div className="page-heading">
        <div>
          <p className="section-label">Network field log</p>
          <h1>Diagnostic history</h1>
          <p>
            Persisted reports, ordered by the moment each route was accepted.
          </p>
        </div>
        <code>{runs.data?.totalItems ?? 0} reports</code>
      </div>

      {runs.error && (
        <div className="error-alert">
          <strong>History unavailable</strong>
          <span>{runs.error.message}</span>
        </div>
      )}

      <div className="history-journal">
        {groups.map((group) => (
          <section className="journal-day" key={group.key}>
            <h2>{group.label}</h2>
            <div>
              {group.runs.map((run) => (
                <Link
                  className="history-row"
                  to={`/runs/${run.id}`}
                  key={run.id}
                >
                  <time dateTime={run.createdAt}>
                    {new Date(run.createdAt).toLocaleTimeString([], {
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </time>
                  <div>
                    <strong>{run.normalizedHost}</strong>
                    <small>{run.target}</small>
                  </div>
                  <span className="history-checks">
                    {run.checks.map((check) => check.toUpperCase()).join(" · ")}
                  </span>
                  <StatusBadge status={run.status} />
                  <code>{durationFor(run)} ms</code>
                  <span className="row-action">Open report →</span>
                </Link>
              ))}
            </div>
          </section>
        ))}
        {!runs.isLoading && groups.length === 0 && (
          <div className="empty-state journal-empty">
            <div className="empty-copy">
              <p className="section-label">No field notes</p>
              <h3>Completed diagnostics will be recorded here</h3>
            </div>
            <Link className="text-button" to="/">
              Run a diagnostic
            </Link>
          </div>
        )}
      </div>
    </main>
  );
}
