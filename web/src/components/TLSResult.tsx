import type { CheckResult, TLSData } from "../api/types";
import { ResultSection } from "./ResultSection";

function Validation({ label, valid }: { label: string; valid: boolean }) {
  return (
    <span className={valid ? "validation-ok" : "validation-failed"}>
      <span aria-hidden="true">{valid ? "●" : "×"}</span>
      {label}
    </span>
  );
}

export function TLSResult({ result }: { result: CheckResult }) {
  const data = result.data as TLSData;
  return (
    <ResultSection
      index="06"
      title="TLS certificate"
      layer="Transport security"
      result={result}
    >
      <dl className="technical-table compact-table">
        <div className="technical-row">
          <dt>Protocol</dt>
          <dd>
            <code>{data.tlsVersion || "Handshake failed"}</code>
          </dd>
        </div>
        <div className="technical-row">
          <dt>Cipher suite</dt>
          <dd>
            <code>{data.cipherSuite || "Unavailable"}</code>
          </dd>
        </div>
        <div className="technical-row">
          <dt>Subject</dt>
          <dd>
            <code>{data.subject || "Unavailable"}</code>
          </dd>
        </div>
        <div className="technical-row">
          <dt>Issuer</dt>
          <dd>
            <code>{data.issuer || "Unavailable"}</code>
          </dd>
        </div>
        <div className="technical-row">
          <dt>Valid until</dt>
          <dd>
            <code>
              {data.validUntil
                ? new Date(data.validUntil).toLocaleDateString()
                : "—"}
              {data.validUntil ? ` · ${data.daysRemaining} days remaining` : ""}
            </code>
          </dd>
        </div>
      </dl>
      <div className="validation-list">
        <Validation label="Hostname verified" valid={data.hostnameValid} />
        <Validation label="Trust chain valid" valid={data.chainValid} />
        <Validation label="Certificate in date" valid={!data.expired} />
      </div>
      {(data.sans ?? []).length > 0 && (
        <div className="sans-list">
          <span>Subject alternative names</span>
          <div>
            {(data.sans ?? []).slice(0, 8).map((san) => (
              <code key={san}>{san}</code>
            ))}
          </div>
        </div>
      )}
    </ResultSection>
  );
}
