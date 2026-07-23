import { Clock3, Globe2 } from "lucide-react";
import type { CheckResult, HTTPData } from "../api/types";
import { StatusBadge } from "./StatusBadge";

export function HTTPResult({ result }: { result: CheckResult }) {
  const data = result.data as HTTPData;
  const phases = [
    ["DNS", data.timings?.dnsMs ?? 0],
    ["Connect", data.timings?.connectMs ?? 0],
    ["TLS", data.timings?.tlsMs ?? 0],
    ["TTFB", data.timings?.ttfbMs ?? 0],
    ["Total", data.timings?.totalMs ?? 0],
  ] as const;
  const maximum = Math.max(...phases.map(([, value]) => value), 1);

  return (
    <section className="result-card">
      <header className="result-header">
        <div className="result-icon">
          <Globe2 size={19} />
        </div>
        <div>
          <p className="card-kicker">Application layer</p>
          <h2>HTTP response</h2>
        </div>
        <StatusBadge status={result.status} />
      </header>
      <div className="result-meta">
        <span>
          <Clock3 size={14} /> {result.durationMs} ms
        </span>
        <span>
          {data.statusCode
            ? `${data.statusCode} · ${data.protocol}`
            : result.summary}
        </span>
      </div>
      {result.errorMessage && (
        <div className="error-alert">{result.errorMessage}</div>
      )}
      <div className="http-summary-grid">
        <div>
          <span>Final URL</span>
          <code>{data.finalUrl ?? data.requestedUrl}</code>
        </div>
        <div>
          <span>Content</span>
          <code>{data.contentType || "unknown"}</code>
        </div>
        <div>
          <span>Remote</span>
          <code>{data.remoteAddress || data.resolvedIp || "—"}</code>
        </div>
        <div>
          <span>Body SHA-256</span>
          <code>{data.bodySha256?.slice(0, 20) ?? "—"}…</code>
        </div>
      </div>
      <div className="timing-chart">
        {phases.map(([label, value]) => (
          <div className="timing-row" key={label}>
            <span>{label}</span>
            <div>
              <i
                style={{ width: `${Math.max((value / maximum) * 100, 2)}%` }}
              />
            </div>
            <code>{value} ms</code>
          </div>
        ))}
      </div>
    </section>
  );
}
