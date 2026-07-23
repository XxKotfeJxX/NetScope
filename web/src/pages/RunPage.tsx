import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  Ban,
  Download,
  LoaderCircle,
  RotateCcw,
} from "lucide-react";
import { useEffect } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { api } from "../api/client";
import { DNSResult } from "../components/DNSResult";
import { HTTPResult } from "../components/HTTPResult";
import { StatusBadge } from "../components/StatusBadge";
import { TCPResult } from "../components/TCPResult";
import { TLSResult } from "../components/TLSResult";

const activeStatuses = new Set(["queued", "running"]);

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
        <LoaderCircle className="spin" />
        Loading diagnostic run…
      </main>
    );
  }
  if (run.error || !run.data) {
    return (
      <main className="page centered-state">
        <div className="error-alert">
          {run.error?.message ?? "Diagnostic run not found."}
        </div>
        <Link to="/">Return to dashboard</Link>
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
        <ArrowLeft size={16} /> Back
      </button>
      <section className="run-summary">
        <div>
          <p className="card-kicker">Diagnostic run</p>
          <h1>{data.normalizedHost}</h1>
          <p className="run-input">{data.target}</p>
        </div>
        <StatusBadge status={data.status} />
      </section>
      <div className="summary-bar">
        <div>
          <span>Created</span>
          <strong>{new Date(data.createdAt).toLocaleString()}</strong>
        </div>
        <div>
          <span>Checks</span>
          <strong>
            {data.checks.map((check) => check.toUpperCase()).join(" · ")}
          </strong>
        </div>
        <div>
          <span>Timeout</span>
          <strong>{data.options.timeoutMs / 1000}s</strong>
        </div>
        <div className="summary-actions">
          {active && (
            <button
              className="secondary-button danger"
              onClick={() => cancel.mutate()}
              disabled={cancel.isPending}
            >
              <Ban size={16} /> Cancel
            </button>
          )}
          <a className="secondary-button" href={api.exportURL(id)}>
            <Download size={16} /> JSON
          </a>
          <Link className="secondary-button" to={`/?target=${data.target}`}>
            <RotateCcw size={16} /> Rerun
          </Link>
        </div>
      </div>

      {active && (
        <section className="waiting-card">
          <span className="scanner-line" />
          <LoaderCircle className="spin" size={22} />
          <div>
            <h2>{data.status === "queued" ? "Queued" : "Running checks"}</h2>
            <p>Live updates are connected through Server-Sent Events.</p>
          </div>
        </section>
      )}
      {dns && <DNSResult result={dns} />}
      {tcp && <TCPResult result={tcp} />}
      {http && <HTTPResult result={http} />}
      {tls && <TLSResult result={tls} />}
      {!active && data.results.length === 0 && (
        <div className="error-alert">This run finished without results.</div>
      )}
    </main>
  );
}
