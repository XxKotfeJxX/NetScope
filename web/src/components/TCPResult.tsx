import { Clock3, Network } from "lucide-react";
import type { CheckResult, TCPData } from "../api/types";
import { StatusBadge } from "./StatusBadge";

export function TCPResult({ result }: { result: CheckResult }) {
  const data = result.data as TCPData;
  return (
    <section className="result-card">
      <header className="result-header">
        <div className="result-icon">
          <Network size={19} />
        </div>
        <div>
          <p className="card-kicker">Transport layer</p>
          <h2>TCP connections</h2>
        </div>
        <StatusBadge status={result.status} />
      </header>
      <div className="result-meta">
        <span>
          <Clock3 size={14} /> {result.durationMs} ms
        </span>
        <span>{result.summary}</span>
      </div>
      <div className="port-list">
        {(data.ports ?? []).map((port) => (
          <div className="port-row" key={port.port}>
            <code>:{port.port}</code>
            <StatusBadge
              status={port.status === "passed" ? "passed" : "failed"}
            />
            <span>{port.resolvedIp ?? port.errorMessage}</span>
            <strong>{port.connectTimeMs} ms</strong>
          </div>
        ))}
      </div>
    </section>
  );
}
