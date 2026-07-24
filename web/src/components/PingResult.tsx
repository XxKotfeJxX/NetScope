import type { CheckResult, PingData } from "../api/types";
import { StatusBadge } from "./StatusBadge";
import { ResultSection } from "./ResultSection";

function milliseconds(value: number | undefined) {
  return value === undefined ? "—" : `${value.toFixed(2)} ms`;
}

export function PingResult({ result }: { result: CheckResult }) {
  const data = result.data as PingData;

  return (
    <ResultSection
      index="03"
      title="ICMP echo"
      layer="Network reachability"
      result={result}
    >
      <div className="metric-strip">
        <div>
          <span>Packet loss</span>
          <strong>{data.packetLossPercent ?? 0}%</strong>
        </div>
        <div>
          <span>Average RTT</span>
          <strong>{milliseconds(data.averageRttMs)}</strong>
        </div>
        <div>
          <span>Range</span>
          <strong>
            {milliseconds(data.minRttMs)} — {milliseconds(data.maxRttMs)}
          </strong>
        </div>
      </div>

      <div className="data-grid-header ping-grid">
        <span>Packet</span>
        <span>Status</span>
        <span>Reply address</span>
        <span>RTT</span>
      </div>
      <div className="data-grid">
        {(data.samples ?? []).map((sample) => (
          <div className="data-grid-row ping-grid" key={sample.sequence}>
            <code>#{sample.sequence}</code>
            <StatusBadge
              status={sample.status === "received" ? "passed" : "failed"}
            />
            <code>{sample.address ?? data.address ?? "—"}</code>
            <strong>{milliseconds(sample.rttMs)}</strong>
          </div>
        ))}
      </div>
    </ResultSection>
  );
}
