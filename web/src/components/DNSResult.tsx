import { Braces, Clock3 } from "lucide-react";
import type { CheckResult, DNSData } from "../api/types";
import { StatusBadge } from "./StatusBadge";

function Records({ label, records }: { label: string; records: string[] }) {
  return (
    <div className="record-group">
      <dt>{label}</dt>
      <dd>
        {records.length ? (
          records.map((record) => <code key={record}>{record}</code>)
        ) : (
          <span className="empty-value">No records</span>
        )}
      </dd>
    </div>
  );
}

export function DNSResult({ result }: { result: CheckResult }) {
  const data = result.data as DNSData;
  const mx = (data.mx ?? []).map(
    (record) => `${record.preference} ${record.host}`,
  );

  return (
    <section className="result-card">
      <header className="result-header">
        <div className="result-icon">
          <Braces size={19} />
        </div>
        <div>
          <p className="card-kicker">Domain name system</p>
          <h2>DNS records</h2>
        </div>
        <StatusBadge status={result.status} />
      </header>
      <div className="result-meta">
        <span>
          <Clock3 size={14} /> {result.durationMs} ms
        </span>
        <span>{result.summary}</span>
      </div>
      {result.errorMessage && (
        <div className="error-alert">{result.errorMessage}</div>
      )}
      <dl className="records-grid">
        <Records label="A" records={data.a ?? []} />
        <Records label="AAAA" records={data.aaaa ?? []} />
        <Records label="CNAME" records={data.cname ? [data.cname] : []} />
        <Records label="MX" records={mx} />
        <Records label="NS" records={data.ns ?? []} />
        <Records label="TXT" records={data.txt ?? []} />
        <Records label="PTR" records={data.ptr ?? []} />
      </dl>
    </section>
  );
}
