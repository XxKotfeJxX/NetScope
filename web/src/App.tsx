import { Activity, Clock3, Code2, Settings2 } from "lucide-react";
import { Link, NavLink, Route, Routes } from "react-router-dom";

function Dashboard() {
  return (
    <main className="page">
      <section className="hero">
        <div className="eyebrow">
          <span className="pulse" />
          Diagnostic workspace
        </div>
        <h1>
          Understand a target
          <span> from DNS to TLS.</span>
        </h1>
        <p>
          NetScope runs focused network checks concurrently and presents every
          result as it arrives.
        </p>
      </section>

      <section className="bootstrap-card">
        <div>
          <p className="card-kicker">Bootstrap ready</p>
          <h2>The diagnostic pipeline is the next vertical slice.</h2>
          <p>
            API health, configuration, PostgreSQL connectivity, and the React
            shell are in place.
          </p>
        </div>
        <Link className="button" to="/settings">
          View runtime
        </Link>
      </section>
    </main>
  );
}

function Placeholder({ title, copy }: { title: string; copy: string }) {
  return (
    <main className="page compact">
      <p className="card-kicker">NetScope</p>
      <h1>{title}</h1>
      <p className="muted">{copy}</p>
    </main>
  );
}

export function App() {
  return (
    <div className="app-shell">
      <header className="topbar">
        <Link className="brand" to="/">
          <span className="brand-mark">
            <Activity size={20} />
          </span>
          NetScope
          <span className="version">v0.0.1</span>
        </Link>
        <nav aria-label="Main navigation">
          <NavLink to="/" end>
            <Activity size={16} /> Diagnose
          </NavLink>
          <NavLink to="/history">
            <Clock3 size={16} /> History
          </NavLink>
          <NavLink to="/settings">
            <Settings2 size={16} /> Settings
          </NavLink>
        </nav>
        <a
          className="icon-link"
          href="https://github.com/XxKotfeJxX/netscope"
          aria-label="NetScope on GitHub"
        >
          <Code2 size={19} />
        </a>
      </header>

      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route
          path="/history"
          element={
            <Placeholder
              title="Run history"
              copy="Saved diagnostics will appear here after the first vertical slice."
            />
          }
        />
        <Route
          path="/settings"
          element={
            <Placeholder
              title="Runtime settings"
              copy="API capabilities and concurrency limits will be exposed here."
            />
          }
        />
      </Routes>
    </div>
  );
}
