import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, Server, ShieldCheck, Workflow } from "lucide-react";
import { api } from "../api/client";

export function SettingsPage() {
  const capabilities = useQuery({
    queryKey: ["capabilities"],
    queryFn: api.capabilities,
  });

  const runtime = capabilities.data?.runtime;
  const cards = [
    {
      icon: Server,
      label: "API version",
      value: capabilities.data?.version ?? "unavailable",
    },
    {
      icon: Workflow,
      label: "Run workers",
      value: runtime ? String(runtime.runWorkers) : "—",
    },
    {
      icon: CheckCircle2,
      label: "Probe concurrency",
      value: runtime ? String(runtime.probeConcurrency) : "—",
    },
    {
      icon: ShieldCheck,
      label: "Network policy",
      value: runtime?.networkPolicy ?? "—",
    },
  ];

  return (
    <main className="page settings-page">
      <div className="page-heading">
        <div>
          <p className="card-kicker">Read-only</p>
          <h1>Runtime settings</h1>
        </div>
      </div>
      <div className="settings-grid">
        {cards.map(({ icon: Icon, label, value }) => (
          <section className="settings-card" key={label}>
            <Icon size={20} />
            <span>{label}</span>
            <strong>{value}</strong>
          </section>
        ))}
      </div>
      <section className="capabilities-card">
        <div>
          <p className="card-kicker">Capabilities</p>
          <h2>Available checks</h2>
        </div>
        <div className="capability-list">
          {Object.entries(capabilities.data?.checks ?? {}).map(
            ([name, capability]) => (
              <span
                className={capability.available ? "available" : ""}
                key={name}
              >
                {name}
                <small>
                  {capability.available ? "available" : capability.reason}
                </small>
              </span>
            ),
          )}
        </div>
      </section>
    </main>
  );
}
