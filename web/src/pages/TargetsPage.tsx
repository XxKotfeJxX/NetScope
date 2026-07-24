import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FormEvent, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { api } from "../api/client";
import type { CheckType, MonitoredTarget, TargetInput } from "../api/types";
import { StatusBadge } from "../components/StatusBadge";

const defaultChecks: CheckType[] = ["dns", "tcp", "tls", "http"];

function defaultInput(): TargetInput {
  return {
    name: "",
    address: "",
    tags: [],
    checks: defaultChecks,
    intervalSeconds: 300,
    failureThreshold: 3,
    options: {
      timeoutMs: 5000,
      tcpPorts: [80, 443],
      httpMethod: "GET",
      followRedirects: true,
      maxRedirects: 5,
      ipVersion: "auto",
      pingPackets: 4,
      maxHops: 20,
    },
  };
}

function groupTargets(targets: MonitoredTarget[]) {
  return targets.reduce<Record<string, MonitoredTarget[]>>((groups, target) => {
    const label = target.tags[0] || "Untagged";
    (groups[label] ??= []).push(target);
    return groups;
  }, {});
}

function relativeTime(value?: string) {
  if (!value) return "Not checked yet";
  const seconds = Math.max(
    Math.round((Date.now() - new Date(value).getTime()) / 1000),
    0,
  );
  if (seconds < 60) return `${seconds}s ago`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  return `${Math.floor(seconds / 3600)}h ago`;
}

export function TargetsPage() {
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const [createOpen, setCreateOpen] = useState(
    () => searchParams.get("new") === "1",
  );
  const targets = useQuery({
    queryKey: ["targets"],
    queryFn: api.listTargets,
    refetchInterval: 10_000,
  });
  const [input, setInput] = useState(defaultInput);
  const [tags, setTags] = useState("");
  const create = useMutation({
    mutationFn: api.createTarget,
    onSuccess: () => {
      setInput(defaultInput());
      setTags("");
      queryClient.invalidateQueries({ queryKey: ["targets"] });
    },
  });

  function submit(event: FormEvent) {
    event.preventDefault();
    create.mutate({
      ...input,
      tags: tags
        .split(",")
        .map((tag) => tag.trim())
        .filter(Boolean),
    });
  }

  const groups = groupTargets(targets.data?.items ?? []);

  return (
    <main className="page targets-page">
      <div className="page-heading">
        <div>
          <p className="section-label">Saved network atlas</p>
          <h1>Targets</h1>
          <p>Named endpoints with scheduled diagnostics and status history.</p>
        </div>
        <code>{targets.data?.totalItems ?? 0} saved targets</code>
      </div>

      <details
        className="target-create"
        open={createOpen || searchParams.get("new") === "1"}
        onToggle={(event) => {
          setCreateOpen(event.currentTarget.open);
          if (searchParams.has("new")) {
            const next = new URLSearchParams(searchParams);
            next.delete("new");
            setSearchParams(next, { replace: true });
          }
        }}
      >
        <summary>Save a target</summary>
        <form onSubmit={submit}>
          <label>
            <span className="option-label">Name</span>
            <input
              required
              maxLength={100}
              value={input.name}
              onChange={(event) =>
                setInput((current) => ({
                  ...current,
                  name: event.target.value,
                }))
              }
              placeholder="Production API"
            />
          </label>
          <label>
            <span className="option-label">Address</span>
            <input
              required
              value={input.address}
              onChange={(event) =>
                setInput((current) => ({
                  ...current,
                  address: event.target.value,
                }))
              }
              placeholder="api.example.com"
            />
          </label>
          <label>
            <span className="option-label">Tags</span>
            <input
              value={tags}
              onChange={(event) => setTags(event.target.value)}
              placeholder="production, api"
            />
          </label>
          <label>
            <span className="option-label">Interval</span>
            <select
              value={input.intervalSeconds}
              onChange={(event) =>
                setInput((current) => ({
                  ...current,
                  intervalSeconds: Number(event.target.value),
                }))
              }
            >
              <option value={60}>Every minute</option>
              <option value={300}>Every 5 minutes</option>
              <option value={900}>Every 15 minutes</option>
              <option value={3600}>Every hour</option>
            </select>
          </label>
          <button className="primary-button" disabled={create.isPending}>
            {create.isPending ? "Saving…" : "Save target"}
          </button>
        </form>
        {create.error && <p className="field-error">{create.error.message}</p>}
      </details>

      {targets.error && (
        <div className="error-alert">
          <strong>Targets unavailable</strong>
          <span>{targets.error.message}</span>
        </div>
      )}

      <div className="target-journal">
        {Object.entries(groups).map(([label, items]) => (
          <section className="target-group" key={label}>
            <h2>{label}</h2>
            <div>
              {items.map((target) => (
                <Link
                  className="target-row"
                  to={`/targets/${target.id}`}
                  key={target.id}
                >
                  <StatusBadge status={target.status} />
                  <div>
                    <strong>{target.name}</strong>
                    <code>{target.address}</code>
                  </div>
                  <span>{target.lastLatencyMs ?? "—"} ms</span>
                  <span>{relativeTime(target.lastCheckedAt)}</span>
                  <span className="row-action">Open target →</span>
                </Link>
              ))}
            </div>
          </section>
        ))}
        {!targets.isLoading && targets.data?.items.length === 0 && (
          <div className="empty-state">
            <div className="empty-copy">
              <p className="section-label">No saved targets</p>
              <h3>Save an endpoint to begin scheduled diagnostics</h3>
            </div>
          </div>
        )}
      </div>
    </main>
  );
}
