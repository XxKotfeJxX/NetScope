import { useMutation, useQuery } from "@tanstack/react-query";
import { ArrowUpRight, Clock3 } from "lucide-react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { api } from "../api/client";
import { RunForm } from "../components/RunForm";
import { StatusBadge } from "../components/StatusBadge";

export function DashboardPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const capabilities = useQuery({
    queryKey: ["capabilities"],
    queryFn: api.capabilities,
  });
  const recent = useQuery({
    queryKey: ["runs", 1, 5],
    queryFn: () => api.listRuns(1, 5),
  });
  const createRun = useMutation({
    mutationFn: api.createRun,
    onSuccess: (run) => navigate(`/runs/${run.id}`),
  });

  return (
    <main className="page dashboard-page">
      <section className="hero">
        <div className="eyebrow">
          <span className="pulse" />
          Live diagnostic workspace
        </div>
        <h1>
          See what the network
          <span> is actually doing.</span>
        </h1>
        <p>
          Run a focused DNS inspection against one explicit target. Results are
          persisted and streamed back as soon as the worker finishes.
        </p>
      </section>

      <RunForm
        capabilities={capabilities.data}
        initialTarget={searchParams.get("target") ?? ""}
        pending={createRun.isPending}
        onSubmit={(payload) => createRun.mutate(payload)}
      />
      {createRun.error && (
        <div className="error-alert form-alert">{createRun.error.message}</div>
      )}

      <section className="recent-section">
        <div className="section-heading">
          <div>
            <p className="card-kicker">PostgreSQL-backed</p>
            <h2>Recent diagnostics</h2>
          </div>
          <Link to="/history">
            View all <ArrowUpRight size={15} />
          </Link>
        </div>
        <div className="run-list">
          {recent.isLoading && <div className="skeleton-row" />}
          {recent.data?.items.map((run) => (
            <Link className="run-row" to={`/runs/${run.id}`} key={run.id}>
              <div className="target-avatar">{run.normalizedHost[0]}</div>
              <div className="run-target">
                <strong>{run.normalizedHost}</strong>
                <span>DNS · {run.target}</span>
              </div>
              <span className="run-time">
                <Clock3 size={14} />
                {new Date(run.createdAt).toLocaleString()}
              </span>
              <StatusBadge status={run.status} />
              <ArrowUpRight size={17} />
            </Link>
          ))}
          {recent.data?.items.length === 0 && (
            <div className="empty-state">
              <RadioTowerIcon />
              <h3>No diagnostics yet</h3>
              <p>Your first DNS result will appear here.</p>
            </div>
          )}
        </div>
      </section>
    </main>
  );
}

function RadioTowerIcon() {
  return <div className="empty-icon">⌁</div>;
}
