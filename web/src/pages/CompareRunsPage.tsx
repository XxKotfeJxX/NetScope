import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { api } from "../api/client";
import type { CheckResult, CheckType, DiagnosticRun } from "../api/types";
import { StatusBadge } from "../components/StatusBadge";

const checkOrder: CheckType[] = [
  "dns",
  "ping",
  "traceroute",
  "tcp",
  "tls",
  "http",
];

function runLabel(run: DiagnosticRun) {
  const time = new Date(run.createdAt).toLocaleString([], {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
  return `${run.normalizedHost} · ${time}`;
}

function signed(value: number) {
  return `${value > 0 ? "+" : ""}${value} ms`;
}

function deltaClass(value: number) {
  if (value > 0) return "comparison-regression";
  if (value < 0) return "comparison-improvement";
  return "";
}

function resultFor(run: DiagnosticRun, type: CheckType) {
  return run.results.find((result) => result.type === type);
}

function ResultCell({ result }: { result?: CheckResult }) {
  if (!result) return <span className="empty-value">Not requested</span>;
  return (
    <div className="comparison-result">
      <StatusBadge status={result.status} />
      <strong>{result.durationMs} ms</strong>
      <span>{result.summary ?? result.errorMessage ?? "No summary"}</span>
    </div>
  );
}

export function RunComparison({
  left,
  right,
}: {
  left: DiagnosticRun;
  right: DiagnosticRun;
}) {
  const types = checkOrder.filter(
    (type) => left.checks.includes(type) || right.checks.includes(type),
  );

  return (
    <section className="comparison-ledger" aria-label="Run comparison">
      <div className="comparison-head comparison-grid">
        <span>Check</span>
        <div>
          <small>Previous run</small>
          <strong>
            {new Date(left.createdAt).toLocaleTimeString([], {
              hour: "2-digit",
              minute: "2-digit",
            })}
          </strong>
        </div>
        <div>
          <small>Current run</small>
          <strong>
            {new Date(right.createdAt).toLocaleTimeString([], {
              hour: "2-digit",
              minute: "2-digit",
            })}
          </strong>
        </div>
        <span>Change</span>
      </div>
      {types.map((type) => {
        const previous = resultFor(left, type);
        const current = resultFor(right, type);
        const delta =
          previous && current
            ? current.durationMs - previous.durationMs
            : undefined;
        return (
          <div className="comparison-row comparison-grid" key={type}>
            <strong>
              {type === "traceroute" ? "TRACE" : type.toUpperCase()}
            </strong>
            <ResultCell result={previous} />
            <ResultCell result={current} />
            <code className={delta === undefined ? "" : deltaClass(delta)}>
              {delta === undefined ? "—" : signed(delta)}
            </code>
          </div>
        );
      })}
    </section>
  );
}

export function CompareRunsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const history = useQuery({
    queryKey: ["runs", 1, 50],
    queryFn: () => api.listRuns(1, 50),
  });
  const items = useMemo(() => history.data?.items ?? [], [history.data?.items]);
  const leftID = searchParams.get("left") ?? "";
  const rightID = searchParams.get("right") ?? "";

  useEffect(() => {
    if (items.length === 0 || (leftID && rightID)) return;
    const next = new URLSearchParams(searchParams);
    if (!leftID) next.set("left", items[1]?.id ?? items[0].id);
    if (!rightID) next.set("right", items[0].id);
    setSearchParams(next, { replace: true });
  }, [items, leftID, rightID, searchParams, setSearchParams]);

  const left = useQuery({
    queryKey: ["run", leftID],
    queryFn: () => api.getRun(leftID),
    enabled: Boolean(leftID),
  });
  const right = useQuery({
    queryKey: ["run", rightID],
    queryFn: () => api.getRun(rightID),
    enabled: Boolean(rightID),
  });

  function choose(side: "left" | "right", id: string) {
    const next = new URLSearchParams(searchParams);
    next.set(side, id);
    setSearchParams(next);
  }

  return (
    <main className="page compare-page">
      <div className="page-heading">
        <div>
          <p className="section-label">Run differential</p>
          <h1>Compare diagnostic runs</h1>
          <p>
            Inspect timing and status changes between two persisted reports.
          </p>
        </div>
        <Link className="text-button" to="/history">
          Open history
        </Link>
      </div>

      <div className="comparison-selectors">
        <label>
          <span className="option-label">Previous run</span>
          <select
            value={leftID}
            onChange={(event) => choose("left", event.target.value)}
          >
            {items.map((run) => (
              <option value={run.id} key={run.id}>
                {runLabel(run)}
              </option>
            ))}
          </select>
        </label>
        <span aria-hidden="true">→</span>
        <label>
          <span className="option-label">Current run</span>
          <select
            value={rightID}
            onChange={(event) => choose("right", event.target.value)}
          >
            {items.map((run) => (
              <option value={run.id} key={run.id}>
                {runLabel(run)}
              </option>
            ))}
          </select>
        </label>
      </div>

      {(history.isLoading || left.isLoading || right.isLoading) && (
        <div className="centered-state compact-state">
          <span className="button-spinner dark" />
          <p>Loading run differential…</p>
        </div>
      )}
      {(history.error || left.error || right.error) && (
        <div className="error-alert">
          <strong>Comparison unavailable</strong>
          <span>
            {history.error?.message ??
              left.error?.message ??
              right.error?.message}
          </span>
        </div>
      )}
      {!history.isLoading && items.length === 0 && (
        <div className="empty-state">
          <div className="empty-copy">
            <p className="section-label">No reports</p>
            <h3>Run two diagnostics to compare their results</h3>
          </div>
          <Link className="text-button" to="/">
            Run a diagnostic
          </Link>
        </div>
      )}
      {left.data && right.data && (
        <>
          {left.data.normalizedHost !== right.data.normalizedHost && (
            <div className="comparison-notice">
              Comparing different targets: {left.data.normalizedHost} and{" "}
              {right.data.normalizedHost}.
            </div>
          )}
          <RunComparison left={left.data} right={right.data} />
        </>
      )}
    </main>
  );
}
