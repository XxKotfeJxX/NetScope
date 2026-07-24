import { useQuery } from "@tanstack/react-query";
import { Link, NavLink, Outlet } from "react-router-dom";
import { api } from "../api/client";
import { useAuth } from "../auth/AuthContext";

export function Layout() {
  const { account, logout, selectWorkspace } = useAuth();
  const capabilities = useQuery({
    queryKey: ["capabilities"],
    queryFn: api.capabilities,
    retry: 1,
  });

  if (!account) return null;

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="workspace-identity">
          <Link className="brand" to="/">
            NETSCOPE<span aria-hidden="true">/</span>
          </Link>
          <span aria-hidden="true">/</span>
          <select
            aria-label="Active workspace"
            value={account.activeWorkspace.id}
            onChange={(event) => void selectWorkspace(event.target.value)}
          >
            {account.workspaces.map((workspace) => (
              <option key={workspace.id} value={workspace.id}>
                {workspace.name}
              </option>
            ))}
          </select>
        </div>

        <nav aria-label="Main navigation">
          <NavLink to="/" end>
            Diagnose
          </NavLink>
          <NavLink to="/targets">Targets</NavLink>
          <NavLink to="/monitoring">Monitoring</NavLink>
          <NavLink to="/history">History</NavLink>
          <NavLink to="/settings">Docs</NavLink>
        </nav>

        <div className="account-tools">
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
          <Link className="workspace-link" to="/workspace">
            {account.activeWorkspace.role}
          </Link>
          <button
            className="account-mark"
            type="button"
            title={`Sign out ${account.user.email}`}
            aria-label={`Sign out ${account.user.displayName}`}
            onClick={() => void logout()}
          >
            {initials(account.user.displayName)}
          </button>
        </div>
      </header>
      <Outlet />
    </div>
  );
}

function initials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join("");
}
