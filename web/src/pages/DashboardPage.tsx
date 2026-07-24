import { useMutation, useQuery } from "@tanstack/react-query";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import type { DiagnosticRun } from "../api/types";
import { api } from "../api/client";
import { RunForm } from "../components/RunForm";
import { StatusBadge } from "../components/StatusBadge";

function durationFor(run: DiagnosticRun) {
  if (run.startedAt && run.completedAt) {
    return Math.max(
      new Date(run.completedAt).getTime() - new Date(run.startedAt).getTime(),
      0,
    );
  }
  return Math.max(...run.results.map((result) => result.durationMs), 0);
}

export function DashboardPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const capabilities = useQuery({
    queryKey: ["capabilities"],
    queryFn: api.capabilities,
  });
  const recent = useQuery({
    queryKey: ["runs", 1, 6],
    queryFn: () => api.listRuns(1, 6),
  });
  const createRun = useMutation({
    mutationFn: api.createRun,
    onSuccess: (run) => navigate(`/runs/${run.id}`),
  });

  return (
    <main className="page dashboard-page">
      <section className="hero">
        <div className="hero-copy">
          <p className="section-label">Network field manual / 01</p>
          <h1>Run a network diagnostic</h1>
          <p>
            Inspect resolution, connection timing, transport security, and the
            final HTTP response for one explicit target.
          </p>
        </div>
        <div className="route-preview" aria-label="Diagnostic sequence">
          <span>target</span>
          <i />
          <span>DNS</span>
          <i />
          <span>TCP</span>
          <i />
          <span>TLS</span>
          <i />
          <span>HTTP</span>
        </div>
      </section>

      <RunForm
        capabilities={capabilities.data}
        initialTarget={searchParams.get("target") ?? ""}
        pending={createRun.isPending}
        onSubmit={(payload) => createRun.mutate(payload)}
      />
      {createRun.error && (
        <div className="error-alert form-alert">
          <strong>Diagnostic could not be started</strong>
          <span>{createRun.error.message}</span>
        </div>
      )}

      <section className="recent-section">
        <div className="section-heading">
          <div>
            <p className="section-label">02 / Field log</p>
            <h2>Recent diagnostics</h2>
          </div>
          <Link to="/history">Open full history →</Link>
        </div>

        <div className="run-log">
          {recent.isLoading && (
            <>
              <div className="skeleton-row" />
              <div className="skeleton-row" />
            </>
          )}
          {recent.data?.items.map((run) => (
            <Link className="run-row" to={`/runs/${run.id}`} key={run.id}>
              <time dateTime={run.createdAt}>
                {new Date(run.createdAt).toLocaleTimeString([], {
                  hour: "2-digit",
                  minute: "2-digit",
                })}
              </time>
              <div className="run-target">
                <strong>{run.normalizedHost}</strong>
                <span>
                  {run.checks.map((check) => check.toUpperCase()).join(" · ")}
                </span>
              </div>
              <StatusBadge status={run.status} />
              <code>{durationFor(run)} ms</code>
              <span className="row-action">Open report →</span>
            </Link>
          ))}
          {recent.data?.items.length === 0 && (
            <div className="empty-state">
              <div className="empty-copy">
                <p className="section-label">No diagnostics yet</p>
                <h3>Start with a hostname, URL, or IP address</h3>
                <p>
                  NetScope will follow the route from DNS resolution to the
                  final HTTP response.
                </p>
              </div>
              <div className="empty-route" aria-hidden="true">
                target ── DNS ── TCP ── TLS ── HTTP
              </div>
            </div>
          )}
        </div>
      </section>
    </main>
  );
}
