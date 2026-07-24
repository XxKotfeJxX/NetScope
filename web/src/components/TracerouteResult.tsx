import type { CheckResult, TracerouteData } from "../api/types";
import { StatusBadge } from "./StatusBadge";
import { ResultSection } from "./ResultSection";

export function TracerouteResult({ result }: { result: CheckResult }) {
  const data = result.data as TracerouteData;

  return (
    <ResultSection
      index="04"
      title="Route trace"
      layer="Network path"
      result={result}
    >
      <div className="trace-summary">
        <span>{data.hops?.length ?? 0} hops recorded</span>
        <strong>
          {data.reached ? `Reached ${data.address}` : "Hop limit reached"}
        </strong>
      </div>
      <div className="data-grid-header trace-grid">
        <span>Hop</span>
        <span>Status</span>
        <span>Router address</span>
        <span>RTT</span>
      </div>
      <div className="data-grid trace-hops">
        {(data.hops ?? []).map((hop) => (
          <div className="data-grid-row trace-grid" key={hop.number}>
            <code>{String(hop.number).padStart(2, "0")}</code>
            <StatusBadge
              status={
                hop.status === "timeout"
                  ? "failed"
                  : hop.destination
                    ? "passed"
                    : "running"
              }
            />
            <code>{hop.address ?? "* * *"}</code>
            <strong>
              {hop.rttMs === undefined ? "—" : `${hop.rttMs.toFixed(2)} ms`}
            </strong>
          </div>
        ))}
      </div>
    </ResultSection>
  );
}
