import type { CheckResult, DNSData } from "../api/types";
import { ResultSection } from "./ResultSection";

function Records({ label, records }: { label: string; records: string[] }) {
  return (
    <div className="technical-row">
      <dt>{label}</dt>
      <dd>
        {records.length ? (
          records.map((record) => <code key={record}>{record}</code>)
        ) : (
          <span className="empty-value">None</span>
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
    <ResultSection
      index="02"
      title="DNS resolution"
      layer="Domain name system"
      result={result}
    >
      <dl className="technical-table">
        <Records label="A records" records={data.a ?? []} />
        <Records label="AAAA records" records={data.aaaa ?? []} />
        <Records
          label="Canonical name"
          records={data.cname ? [data.cname] : []}
        />
        <Records label="Mail exchange" records={mx} />
        <Records label="Name servers" records={data.ns ?? []} />
        <Records label="Text records" records={data.txt ?? []} />
        <Records label="Reverse lookup" records={data.ptr ?? []} />
      </dl>
    </ResultSection>
  );
}
