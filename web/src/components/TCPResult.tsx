import type { CheckResult, TCPData } from "../api/types";
import { StatusBadge } from "./StatusBadge";
import { ResultSection } from "./ResultSection";

export function TCPResult({ result }: { result: CheckResult }) {
  const data = result.data as TCPData;
  return (
    <ResultSection
      index="05"
      title="TCP connections"
      layer="Transport layer"
      result={result}
    >
      <div className="data-grid-header port-grid">
        <span>Port</span>
        <span>Status</span>
        <span>Resolved address</span>
        <span>Connect</span>
      </div>
      <div className="data-grid">
        {(data.ports ?? []).map((port) => (
          <div className="data-grid-row port-grid" key={port.port}>
            <code>:{port.port}</code>
            <StatusBadge
              status={port.status === "passed" ? "passed" : "failed"}
            />
            <code>{port.resolvedIp ?? port.errorMessage ?? "Unavailable"}</code>
            <strong>{port.connectTimeMs} ms</strong>
          </div>
        ))}
      </div>
    </ResultSection>
  );
}
