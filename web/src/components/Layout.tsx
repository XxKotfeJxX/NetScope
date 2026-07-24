import { useQuery } from "@tanstack/react-query";
import { Link, NavLink, Outlet } from "react-router-dom";
import { api } from "../api/client";

export function Layout() {
  const capabilities = useQuery({
    queryKey: ["capabilities"],
    queryFn: api.capabilities,
    retry: 1,
  });

  return (
    <div className="app-shell">
      <header className="topbar">
        <Link className="brand" to="/">
          NETSCOPE<span aria-hidden="true">/</span>
        </Link>
        <nav aria-label="Main navigation">
          <NavLink to="/" end>
            Diagnose
          </NavLink>
          <NavLink to="/targets">Targets</NavLink>
          <NavLink to="/monitoring">Monitoring</NavLink>
          <NavLink to="/history">History</NavLink>
          <NavLink to="/settings">Runtime</NavLink>
          <a
            href="https://github.com/XxKotfeJxX/NetScope/blob/dev/api/openapi.yaml"
            target="_blank"
            rel="noreferrer"
          >
            API reference ↗
          </a>
        </nav>
        <Link className="api-state" to="/settings">
          <span
            className={capabilities.isSuccess ? "is-online" : "is-checking"}
          />
          {capabilities.isSuccess
            ? `API online · ${capabilities.data.version}`
            : capabilities.isError
              ? "API offline"
              : "Checking API"}
        </Link>
      </header>
      <Outlet />
    </div>
  );
}
