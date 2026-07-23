import { CalendarClock, Check, LockKeyhole, X } from "lucide-react";
import type { CheckResult, TLSData } from "../api/types";
import { StatusBadge } from "./StatusBadge";

function Validation({ label, valid }: { label: string; valid: boolean }) {
  return (
    <span className={valid ? "validation-ok" : "validation-failed"}>
      {valid ? <Check size={13} /> : <X size={13} />} {label}
    </span>
  );
}

export function TLSResult({ result }: { result: CheckResult }) {
  const data = result.data as TLSData;
  return (
    <section className="result-card">
      <header className="result-header">
        <div className="result-icon">
          <LockKeyhole size={19} />
        </div>
        <div>
          <p className="card-kicker">Transport security</p>
          <h2>TLS certificate</h2>
        </div>
        <StatusBadge status={result.status} />
      </header>
      <div className="result-meta">
        <span>{data.tlsVersion || "Handshake failed"}</span>
        <span>{data.cipherSuite}</span>
      </div>
      {result.errorMessage && (
        <div className="error-alert">{result.errorMessage}</div>
      )}
      <div className="certificate-grid">
        <div>
          <span>Subject</span>
          <code>{data.subject || "Unavailable"}</code>
        </div>
        <div>
          <span>Issuer</span>
          <code>{data.issuer || "Unavailable"}</code>
        </div>
        <div>
          <span>Valid until</span>
          <strong>
            <CalendarClock size={14} />
            {data.validUntil
              ? new Date(data.validUntil).toLocaleDateString()
              : "—"}
          </strong>
        </div>
        <div>
          <span>Remaining</span>
          <strong>{data.daysRemaining} days</strong>
        </div>
      </div>
      <div className="validation-list">
        <Validation label="Hostname" valid={data.hostnameValid} />
        <Validation label="Trust chain" valid={data.chainValid} />
        <Validation label="Date" valid={!data.expired} />
      </div>
      {(data.sans ?? []).length > 0 && (
        <div className="sans-list">
          <span>SANs</span>
          {(data.sans ?? []).slice(0, 8).map((san) => (
            <code key={san}>{san}</code>
          ))}
        </div>
      )}
    </section>
  );
}
