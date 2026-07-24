import { useQuery } from "@tanstack/react-query";
import { api } from "../api/client";

export function SettingsPage() {
  const capabilities = useQuery({
    queryKey: ["capabilities"],
    queryFn: api.capabilities,
  });

  const runtime = capabilities.data?.runtime;

  return (
    <main className="page runtime-page">
      <div className="page-heading">
        <div>
          <p className="section-label">Instrument profile</p>
          <h1>Runtime configuration</h1>
          <p>Read-only operating limits reported by the active NetScope API.</p>
        </div>
        <span className="runtime-connection">
          <i className={capabilities.isSuccess ? "is-online" : "is-checking"} />
          {capabilities.isSuccess ? "Connected" : "Unavailable"}
        </span>
      </div>

      <section className="runtime-section">
        <div className="runtime-section-heading">
          <p className="section-label">01 / Process</p>
          <h2>Execution profile</h2>
        </div>
        <dl className="runtime-table">
          <div>
            <dt>API version</dt>
            <dd>{capabilities.data?.version ?? "unavailable"}</dd>
          </div>
          <div>
            <dt>Run workers</dt>
            <dd>{runtime?.runWorkers ?? "—"}</dd>
          </div>
          <div>
            <dt>Probe concurrency</dt>
            <dd>{runtime?.probeConcurrency ?? "—"}</dd>
          </div>
          <div>
            <dt>Default timeout</dt>
            <dd>{runtime ? `${runtime.defaultTimeoutMs / 1000}s` : "—"}</dd>
          </div>
          <div>
            <dt>Maximum timeout</dt>
            <dd>{runtime ? `${runtime.maxTimeoutMs / 1000}s` : "—"}</dd>
          </div>
          <div>
            <dt>Network policy</dt>
            <dd>{runtime?.networkPolicy ?? "—"}</dd>
          </div>
        </dl>
      </section>

      <section className="runtime-section">
        <div className="runtime-section-heading">
          <p className="section-label">02 / Capabilities</p>
          <h2>Available diagnostics</h2>
        </div>
        <div className="capability-list">
          {Object.entries(capabilities.data?.checks ?? {}).map(
            ([name, capability]) => (
              <div key={name}>
                <span className={capability.available ? "available" : ""}>
                  {capability.available ? "●" : "○"}
                </span>
                <strong>{name}</strong>
                <small>
                  {capability.available
                    ? "Available"
                    : capability.reason || "Unavailable"}
                </small>
              </div>
            ),
          )}
        </div>
      </section>
    </main>
  );
}
