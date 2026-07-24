import type {
  CheckResult,
  CheckStatus,
  CheckType,
  DiagnosticRun,
  DNSData,
  HTTPData,
  TCPData,
  TLSData,
} from "../api/types";
import { StatusBadge } from "./StatusBadge";

const routeOrder: CheckType[] = ["dns", "tcp", "tls", "http"];

const labels: Record<CheckType, { title: string; action: string }> = {
  dns: { title: "DNS", action: "Resolve host" },
  tcp: { title: "TCP", action: "Open connection" },
  tls: { title: "TLS", action: "Verify certificate" },
  http: { title: "HTTP", action: "Read response" },
};

function resultSummary(result: CheckResult) {
  if (result.errorMessage) return result.errorMessage;
  if (result.type === "dns") {
    const data = result.data as DNSData;
    const addresses = [...(data.a ?? []), ...(data.aaaa ?? [])];
    return addresses.length
      ? `${addresses.length} address${addresses.length === 1 ? "" : "es"} · ${addresses[0]}`
      : result.summary || "Resolution complete";
  }
  if (result.type === "tcp") {
    const data = result.data as TCPData;
    const passed = (data.ports ?? [])
      .filter((port) => port.status === "passed")
      .map((port) => port.port);
    return passed.length ? `Port ${passed.join(", ")}` : result.summary || "";
  }
  if (result.type === "tls") {
    const data = result.data as TLSData;
    return data.tlsVersion
      ? `${data.tlsVersion} · valid for ${data.daysRemaining} days`
      : result.summary || "Handshake complete";
  }
  const data = result.data as HTTPData;
  return data.statusCode
    ? `${data.statusCode} · ${data.protocol ?? "HTTP"}`
    : result.summary || "";
}

function stageStatus(
  run: DiagnosticRun,
  type: CheckType,
  result: CheckResult | undefined,
  firstMissing: CheckType | undefined,
): CheckStatus | "skipped" {
  if (result) return result.status;
  if (run.status === "queued") return "pending";
  if (run.status === "running" && type === firstMissing) return "running";
  return run.status === "running" ? "pending" : "skipped";
}

export function DiagnosticRoute({ run }: { run: DiagnosticRun }) {
  const selected = routeOrder.filter((type) => run.checks.includes(type));
  const firstMissing = selected.find(
    (type) => !run.results.some((result) => result.type === type),
  );

  return (
    <section className="diagnostic-route" aria-labelledby="route-title">
      <header className="route-heading">
        <div>
          <p className="section-label">01 / Diagnostic route</p>
          <h2 id="route-title">Network path</h2>
        </div>
        <code>{run.normalizedHost}</code>
      </header>

      <ol className="route-list">
        <li className="route-step route-target">
          <span className="route-node route-node-passed" aria-hidden="true">
            ●
          </span>
          <div>
            <span>Target</span>
            <strong>Accepted</strong>
            <code>{run.target}</code>
          </div>
          <span className="route-duration">input</span>
        </li>
        {selected.map((type) => {
          const result = run.results.find((item) => item.type === type);
          const status = stageStatus(run, type, result, firstMissing);
          return (
            <li className={`route-step route-step-${status}`} key={type}>
              <span className={`route-node route-node-${status}`} aria-hidden>
                {status === "passed"
                  ? "●"
                  : status === "failed"
                    ? "×"
                    : status === "warning"
                      ? "△"
                      : status === "running"
                        ? "↻"
                        : "○"}
              </span>
              <div>
                <span>{labels[type].title}</span>
                <strong>
                  {result ? resultSummary(result) : labels[type].action}
                </strong>
                {result?.errorCode && <code>{result.errorCode}</code>}
              </div>
              <div className="route-end">
                <StatusBadge status={status} />
                <span className="route-duration">
                  {result ? `${result.durationMs} ms` : "—"}
                </span>
              </div>
            </li>
          );
        })}
      </ol>
    </section>
  );
}
