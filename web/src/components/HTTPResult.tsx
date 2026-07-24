import type { CheckResult, HTTPData } from "../api/types";
import { ResultSection } from "./ResultSection";

export function HTTPResult({ result }: { result: CheckResult }) {
  const data = result.data as HTTPData;
  const phases = [
    ["DNS", data.timings?.dnsMs ?? 0],
    ["Connect", data.timings?.connectMs ?? 0],
    ["TLS", data.timings?.tlsMs ?? 0],
    ["TTFB", data.timings?.ttfbMs ?? 0],
  ] as const;
  const total = Math.max(data.timings?.totalMs ?? 0, 1);

  return (
    <ResultSection
      index="05"
      title="HTTP response"
      layer="Application layer"
      result={result}
    >
      <dl className="technical-table compact-table">
        <div className="technical-row">
          <dt>Status</dt>
          <dd>
            <code>
              {data.statusCode
                ? `${data.statusCode} · ${data.protocol ?? "HTTP"}`
                : "Unavailable"}
            </code>
          </dd>
        </div>
        <div className="technical-row">
          <dt>Final URL</dt>
          <dd>
            <code>{data.finalUrl ?? data.requestedUrl}</code>
          </dd>
        </div>
        <div className="technical-row">
          <dt>Content</dt>
          <dd>
            <code>{data.contentType || "unknown"}</code>
          </dd>
        </div>
        <div className="technical-row">
          <dt>Remote</dt>
          <dd>
            <code>{data.remoteAddress || data.resolvedIp || "—"}</code>
          </dd>
        </div>
        <div className="technical-row">
          <dt>Body SHA-256</dt>
          <dd>
            <code>{data.bodySha256 ?? "—"}</code>
          </dd>
        </div>
      </dl>

      <div className="timing-block">
        <div className="subsection-heading">
          <span>Request timing</span>
          <code>{data.timings?.totalMs ?? 0} ms total</code>
        </div>
        <div className="timing-strip" aria-label="HTTP request timing">
          {phases.map(([label, value]) => (
            <div
              className={`timing-segment timing-${label.toLowerCase()}`}
              key={label}
              style={{ flexGrow: Math.max(value, total * 0.04) }}
              title={`${label}: ${value} ms`}
            >
              <span>{label}</span>
              <code>{value} ms</code>
            </div>
          ))}
        </div>
      </div>
    </ResultSection>
  );
}
