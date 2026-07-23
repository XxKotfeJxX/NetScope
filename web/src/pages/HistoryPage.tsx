import { useQuery } from "@tanstack/react-query";
import { ArrowUpRight, Clock3 } from "lucide-react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import { StatusBadge } from "../components/StatusBadge";

export function HistoryPage() {
  const runs = useQuery({
    queryKey: ["runs", 1, 20],
    queryFn: () => api.listRuns(1, 20),
  });

  return (
    <main className="page table-page">
      <div className="page-heading">
        <div>
          <p className="card-kicker">Saved results</p>
          <h1>Run history</h1>
        </div>
        <span>{runs.data?.totalItems ?? 0} total</span>
      </div>
      {runs.error && <div className="error-alert">{runs.error.message}</div>}
      <div className="history-table">
        <div className="history-head">
          <span>Target</span>
          <span>Checks</span>
          <span>Created</span>
          <span>Status</span>
          <span />
        </div>
        {runs.data?.items.map((run) => (
          <Link className="history-row" to={`/runs/${run.id}`} key={run.id}>
            <div>
              <strong>{run.normalizedHost}</strong>
              <small>{run.target}</small>
            </div>
            <span>DNS</span>
            <span>
              <Clock3 size={14} /> {new Date(run.createdAt).toLocaleString()}
            </span>
            <StatusBadge status={run.status} />
            <ArrowUpRight size={16} />
          </Link>
        ))}
      </div>
    </main>
  );
}
