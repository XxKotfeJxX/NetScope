import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FormEvent, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { api } from "../api/client";
import type { MonitoringCheck } from "../api/types";
import { StatusBadge } from "../components/StatusBadge";

function timestamp(value?: string) {
  return value ? new Date(value).toLocaleString() : "—";
}

function LatencySparkline({ checks }: { checks: MonitoringCheck[] }) {
  const points = checks
    .filter((check) => check.latencyMs !== undefined)
    .slice(0, 30)
    .reverse();
  if (points.length < 2)
    return <span className="empty-value">More samples needed</span>;
  const maximum = Math.max(...points.map((point) => point.latencyMs ?? 0), 1);
  const path = points
    .map((point, index) => {
      const x = (index / (points.length - 1)) * 100;
      const y = 38 - ((point.latencyMs ?? 0) / maximum) * 34;
      return `${index === 0 ? "M" : "L"} ${x} ${y}`;
    })
    .join(" ");
  return (
    <svg
      className="latency-sparkline"
      viewBox="0 0 100 42"
      preserveAspectRatio="none"
      role="img"
      aria-label="Recent latency history"
    >
      <path d={path} />
    </svg>
  );
}

export function TargetDetailPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [maintenance, setMaintenance] = useState({
    startsAt: "",
    endsAt: "",
    reason: "",
  });
  const [channel, setChannel] = useState<{
    kind: "email" | "webhook";
    destination: string;
  }>({ kind: "webhook", destination: "" });

  const target = useQuery({
    queryKey: ["target", id],
    queryFn: () => api.getTarget(id),
    enabled: Boolean(id),
    refetchInterval: 10_000,
  });
  const checks = useQuery({
    queryKey: ["target-checks", id],
    queryFn: () => api.targetChecks(id),
    enabled: Boolean(id),
    refetchInterval: 10_000,
  });
  const windows = useQuery({
    queryKey: ["maintenance", id],
    queryFn: () => api.maintenanceWindows(id),
    enabled: Boolean(id),
  });
  const channels = useQuery({
    queryKey: ["notification-channels", id],
    queryFn: () => api.notificationChannels(id),
    enabled: Boolean(id),
  });

  const refreshTarget = () => {
    queryClient.invalidateQueries({ queryKey: ["target", id] });
    queryClient.invalidateQueries({ queryKey: ["targets"] });
  };
  const toggle = useMutation({
    mutationFn: (enabled: boolean) =>
      enabled ? api.resumeTarget(id) : api.pauseTarget(id),
    onSuccess: refreshTarget,
  });
  const remove = useMutation({
    mutationFn: () => api.deleteTarget(id),
    onSuccess: () => navigate("/targets"),
  });
  const addMaintenance = useMutation({
    mutationFn: () =>
      api.createMaintenanceWindow(id, {
        startsAt: new Date(maintenance.startsAt).toISOString(),
        endsAt: new Date(maintenance.endsAt).toISOString(),
        reason: maintenance.reason,
      }),
    onSuccess: () => {
      setMaintenance({ startsAt: "", endsAt: "", reason: "" });
      queryClient.invalidateQueries({ queryKey: ["maintenance", id] });
    },
  });
  const addChannel = useMutation({
    mutationFn: () => api.createNotificationChannel(id, channel),
    onSuccess: () => {
      setChannel((current) => ({ ...current, destination: "" }));
      queryClient.invalidateQueries({
        queryKey: ["notification-channels", id],
      });
    },
  });

  const orderedChecks = useMemo(
    () => checks.data?.items ?? [],
    [checks.data?.items],
  );

  if (target.isLoading) {
    return (
      <main className="page centered-state">
        <span className="button-spinner dark" />
        <p>Loading monitored target…</p>
      </main>
    );
  }
  if (!target.data || target.error) {
    return (
      <main className="page centered-state">
        <div className="error-alert">
          <strong>Target unavailable</strong>
          <span>{target.error?.message ?? "Target not found."}</span>
        </div>
        <Link to="/targets">Return to targets</Link>
      </main>
    );
  }

  const data = target.data;

  return (
    <main className="page target-detail-page">
      <Link className="back-link" to="/targets">
        ← Back to targets
      </Link>
      <div className="target-detail-head">
        <div>
          <p className="section-label">Target / {data.id.slice(0, 8)}</p>
          <h1>{data.name}</h1>
          <code>{data.address}</code>
        </div>
        <div className="target-head-state">
          <StatusBadge status={data.status} />
          <span>Last checked {timestamp(data.lastCheckedAt)}</span>
        </div>
      </div>

      <div className="target-summary-line">
        <span>{data.lastLatencyMs ?? "—"} ms latest latency</span>
        <span>{data.consecutiveFailures} consecutive failures</span>
        <span>
          TLS{" "}
          {data.tlsExpiresAt ? timestamp(data.tlsExpiresAt) : "not observed"}
        </span>
        <button onClick={() => toggle.mutate(!data.enabled)}>
          {data.enabled ? "Pause monitoring" : "Resume monitoring"}
        </button>
      </div>

      <section className="target-history">
        <div className="section-heading">
          <div>
            <p className="section-label">01 / Status history</p>
            <h2>Timeline</h2>
          </div>
          <LatencySparkline checks={orderedChecks} />
        </div>
        <div className="monitoring-timeline">
          {orderedChecks.map((check) => (
            <div className="timeline-entry" key={check.id}>
              <time>{timestamp(check.checkedAt ?? check.createdAt)}</time>
              <StatusBadge status={check.status} />
              <strong>
                {check.latencyMs === undefined ? "—" : `${check.latencyMs} ms`}
              </strong>
              <span>
                {check.errorMessage || "Scheduled diagnostic completed"}
              </span>
              {check.runId ? (
                <Link to={`/runs/${check.runId}`}>Report →</Link>
              ) : (
                <i />
              )}
            </div>
          ))}
          {!checks.isLoading && orderedChecks.length === 0 && (
            <div className="empty-state">
              <div className="empty-copy">
                <p className="section-label">Awaiting first check</p>
                <h3>The scheduler will record status and latency here</h3>
              </div>
            </div>
          )}
        </div>
      </section>

      <div className="target-config-grid">
        <section>
          <p className="section-label">02 / Configuration</p>
          <dl className="technical-table">
            <div className="technical-row">
              <dt>Interval</dt>
              <dd>{data.intervalSeconds / 60} minutes</dd>
            </div>
            <div className="technical-row">
              <dt>Checks</dt>
              <dd>
                {data.checks.map((check) => check.toUpperCase()).join(" · ")}
              </dd>
            </div>
            <div className="technical-row">
              <dt>Failure threshold</dt>
              <dd>{data.failureThreshold}</dd>
            </div>
            <div className="technical-row">
              <dt>Tags</dt>
              <dd>{data.tags.join(" · ") || "None"}</dd>
            </div>
          </dl>
        </section>

        <section>
          <p className="section-label">03 / Maintenance</p>
          <form
            className="compact-config-form"
            onSubmit={(event: FormEvent) => {
              event.preventDefault();
              addMaintenance.mutate();
            }}
          >
            <input
              required
              type="datetime-local"
              value={maintenance.startsAt}
              onChange={(event) =>
                setMaintenance((current) => ({
                  ...current,
                  startsAt: event.target.value,
                }))
              }
              aria-label="Maintenance starts"
            />
            <input
              required
              type="datetime-local"
              value={maintenance.endsAt}
              onChange={(event) =>
                setMaintenance((current) => ({
                  ...current,
                  endsAt: event.target.value,
                }))
              }
              aria-label="Maintenance ends"
            />
            <input
              value={maintenance.reason}
              onChange={(event) =>
                setMaintenance((current) => ({
                  ...current,
                  reason: event.target.value,
                }))
              }
              placeholder="Deployment"
              aria-label="Maintenance reason"
            />
            <button>Add window</button>
          </form>
          <div className="config-list">
            {windows.data?.map((window) => (
              <div key={window.id}>
                <span>{window.reason || "Maintenance"}</span>
                <code>{timestamp(window.startsAt)}</code>
                <button
                  onClick={() =>
                    api.deleteMaintenanceWindow(id, window.id).then(() =>
                      queryClient.invalidateQueries({
                        queryKey: ["maintenance", id],
                      }),
                    )
                  }
                >
                  Remove
                </button>
              </div>
            ))}
          </div>
        </section>

        <section>
          <p className="section-label">04 / Notifications</p>
          <form
            className="compact-config-form channel-form"
            onSubmit={(event: FormEvent) => {
              event.preventDefault();
              addChannel.mutate();
            }}
          >
            <select
              value={channel.kind}
              onChange={(event) =>
                setChannel((current) => ({
                  ...current,
                  kind: event.target.value as "email" | "webhook",
                }))
              }
            >
              <option value="webhook">Webhook</option>
              <option value="email">Email</option>
            </select>
            <input
              required
              value={channel.destination}
              onChange={(event) =>
                setChannel((current) => ({
                  ...current,
                  destination: event.target.value,
                }))
              }
              placeholder={
                channel.kind === "email"
                  ? "ops@example.com"
                  : "https://hooks.example.com/…"
              }
            />
            <button>Add channel</button>
          </form>
          <div className="config-list">
            {channels.data?.map((item) => (
              <div key={item.id}>
                <span>{item.kind}</span>
                <code>{item.destination}</code>
                <button
                  onClick={() =>
                    api.deleteNotificationChannel(id, item.id).then(() =>
                      queryClient.invalidateQueries({
                        queryKey: ["notification-channels", id],
                      }),
                    )
                  }
                >
                  Remove
                </button>
              </div>
            ))}
          </div>
        </section>
      </div>

      <div className="target-danger-zone">
        <span>Deleting a target also removes its monitoring history.</span>
        <button
          disabled={remove.isPending}
          onClick={() => {
            if (window.confirm(`Delete ${data.name}?`)) remove.mutate();
          }}
        >
          Delete target
        </button>
      </div>
    </main>
  );
}
