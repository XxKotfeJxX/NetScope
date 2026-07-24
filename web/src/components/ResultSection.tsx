import type { ReactNode } from "react";
import type { CheckResult } from "../api/types";
import { StatusBadge } from "./StatusBadge";

export function ResultSection({
  index,
  title,
  layer,
  result,
  children,
}: {
  index: string;
  title: string;
  layer: string;
  result: CheckResult;
  children: ReactNode;
}) {
  return (
    <section className="result-section">
      <details open={result.status === "failed"}>
        <summary>
          <span className="result-index">{index}</span>
          <span className="result-title">
            <small>{layer}</small>
            <strong>{title}</strong>
            <span>{result.summary || "Diagnostic result"}</span>
          </span>
          <StatusBadge status={result.status} />
          <code className="result-duration">{result.durationMs} ms</code>
          <span className="disclosure-label">Details</span>
        </summary>
        <div className="result-content">
          {result.errorMessage && (
            <div className="error-alert">
              <strong>{title} failed</strong>
              <span>{result.errorMessage}</span>
            </div>
          )}
          {children}
          <details className="raw-data">
            <summary>View raw data</summary>
            <pre>{JSON.stringify(result.data, null, 2)}</pre>
          </details>
        </div>
      </details>
    </section>
  );
}
