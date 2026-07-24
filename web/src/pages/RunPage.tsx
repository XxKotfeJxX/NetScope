import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { api } from "../api/client";
import { DiagnosticRoute } from "../components/DiagnosticRoute";
import { DNSResult } from "../components/DNSResult";
import { HTTPResult } from "../components/HTTPResult";
import { StatusBadge } from "../components/StatusBadge";
import { TCPResult } from "../components/TCPResult";
import { TLSResult } from "../components/TLSResult";

const activeStatuses = new Set(["queued", "running"]);

function formatTimestamp(value?: string) {
  return value ? new Date(value).toLocaleString() : "—";
}

export function RunPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const run = useQuery({
    queryKey: ["run", id],
    queryFn: () => api.getRun(id),
    enabled: Boolean(id),
    refetchInterval: (query) =>
      activeStatuses.has(query.state.data?.status ?? "") ? 2000 : false,
  });
  const cancel = useMutation({
    mutationFn: () => api.cancelRun(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["run", id] }),
  });

  useEffect(() => {
    if (!id || typeof EventSource === "undefined") return;
    const stream = new EventSource(api.eventURL(id));
    const update = () =>
      queryClient.invalidateQueries({ queryKey: ["run", id] });
    [
      "run.started",
      "check.started",
      "check.completed",
      "check.failed",
      "run.completed",
      "run.cancelled",
    ].forEach((event) => stream.addEventListener(event, update));
    return () => stream.close();
  }, [id, queryClient]);

  if (run.isLoading) {
    return (
      <main className="page centered-state">
        <span className="button-spinner dark" />
        <p>Loading diagnostic report…</p>
      </main>
    );
  }
  if (run.error || !run.data) {
    return (
      <main className="page centered-state">
        <div className="error-alert">
          <strong>Diagnostic report unavailable</strong>
          <span>{run.error?.message ?? "Diagnostic run not found."}</span>
        </div>
        <Link to="/">Return to diagnostics</Link>
      </main>
    );
  }

  const data = run.data;
  const dns = data.results.find((result) => result.type === "dns");
  const tcp = data.results.find((result) => result.type === "tcp");
  const http = data.results.find((result) => result.type === "http");
  const tls = data.results.find((result) => result.type === "tls");
  const active = activeStatuses.has(data.status);

  return (
    <main className="page run-page">
      <button className="back-link" onClick={() => navigate(-1)}>
        ← Back to field log
      </button>

      <div className="report-layout">
        <aside className="report-index">
          <p className="section-label">Report / {data.id.slice(0, 8)}</p>
          <h1>{data.normalizedHost}</h1>
          <code className="report-target">{data.target}</code>
          <StatusBadge status={data.status} />

          <dl className="report-facts">
            <div>
              <dt>Created</dt>
              <dd>{formatTimestamp(data.createdAt)}</dd>
            </div>
            <div>
              <dt>Completed</dt>
              <dd>{formatTimestamp(data.completedAt)}</dd>
            </div>
            <div>
              <dt>Checks</dt>
              <dd>
                {data.checks.map((check) => check.toUpperCase()).join(" · ")}
              </dd>
            </div>
            <div>
              <dt>Probe timeout</dt>
              <dd>{data.options.timeoutMs / 1000}s</dd>
            </div>
            <div>
              <dt>IP strategy</dt>
              <dd>{data.options.ipVersion}</dd>
            </div>
          </dl>

          <div className="report-actions">
            {active && (
              <button
                className="text-button danger"
                onClick={() => cancel.mutate()}
                disabled={cancel.isPending}
              >
                Cancel run
              </button>
            )}
            <a className="text-button" href={api.exportURL(id)}>
              Download JSON
            </a>
            <Link className="text-button" to={`/?target=${data.target}`}>
              Run again
            </Link>
          </div>
        </aside>

        <div className="report-body">
          {active && (
            <div className="live-notice">
              <span className="live-pulse" aria-hidden="true" />
              <div>
                <strong>
                  {data.status === "queued"
                    ? "Waiting for an available worker"
                    : "Diagnostic route in progress"}
                </strong>
                <span>Results update live as each stage completes.</span>
              </div>
            </div>
          )}

          <DiagnosticRoute run={data} />

          <div className="result-ledger">
            <div className="ledger-heading">
              <p className="section-label">02–05 / Technical record</p>
              <h2>Detailed results</h2>
              <p>Expand a stage to inspect structured output and raw data.</p>
            </div>
            {dns && <DNSResult result={dns} />}
            {tcp && <TCPResult result={tcp} />}
            {tls && <TLSResult result={tls} />}
            {http && <HTTPResult result={http} />}
            {!active && data.results.length === 0 && (
              <div className="error-alert">
                <strong>No probe output recorded</strong>
                <span>
                  This run finished before any diagnostic stage produced a
                  result. Try running the target again.
                </span>
              </div>
            )}
          </div>
        </div>
      </div>
    </main>
  );
}
