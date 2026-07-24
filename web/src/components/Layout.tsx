import { useQuery } from "@tanstack/react-query";
import { Menu, Moon, Search, Sun, X } from "lucide-react";
import { useState } from "react";
import { Link, NavLink, Outlet } from "react-router-dom";
import { api } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { initialTheme, saveTheme, type Theme } from "../theme";
import { CommandPalette } from "./CommandPalette";

export function Layout() {
  const { account, logout, selectWorkspace } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [theme, setTheme] = useState<Theme>(initialTheme);
  const capabilities = useQuery({
    queryKey: ["capabilities"],
    queryFn: api.capabilities,
    retry: 1,
  });

  if (!account) return null;

  const toggleTheme = () => {
    const nextTheme = theme === "light" ? "dark" : "light";
    setTheme(nextTheme);
    saveTheme(nextTheme);
  };

  return (
    <div className="app-shell">
      <a className="skip-link" href="#main-content">
        Skip to main content
      </a>
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

        <nav
          id="main-navigation"
          className={menuOpen ? "is-open" : ""}
          aria-label="Main navigation"
          onClick={() => setMenuOpen(false)}
        >
          <NavLink to="/" end>
            Diagnose
          </NavLink>
          <NavLink to="/targets">Targets</NavLink>
          <NavLink to="/monitoring">Monitoring</NavLink>
          <NavLink to="/history">History</NavLink>
          <NavLink to="/settings">Docs</NavLink>
        </nav>

        <div className="account-tools">
          <button
            className="command-trigger"
            type="button"
            aria-label="Open command palette"
            onClick={() => setPaletteOpen(true)}
          >
            <Search aria-hidden="true" size={15} strokeWidth={1.8} />
            <span>Search</span>
            <kbd>Ctrl K</kbd>
          </button>
          <button
            className="icon-button"
            type="button"
            aria-label={`Switch to ${theme === "light" ? "dark" : "light"} theme`}
            title={`Switch to ${theme === "light" ? "dark" : "light"} theme`}
            onClick={toggleTheme}
          >
            {theme === "light" ? (
              <Moon aria-hidden="true" size={16} strokeWidth={1.7} />
            ) : (
              <Sun aria-hidden="true" size={16} strokeWidth={1.7} />
            )}
          </button>
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
          <button
            className="mobile-menu-button"
            type="button"
            aria-label={menuOpen ? "Close navigation" : "Open navigation"}
            aria-expanded={menuOpen}
            aria-controls="main-navigation"
            onClick={() => setMenuOpen((current) => !current)}
          >
            {menuOpen ? (
              <X aria-hidden="true" size={18} />
            ) : (
              <Menu aria-hidden="true" size={18} />
            )}
          </button>
        </div>
      </header>
      <div id="main-content" tabIndex={-1}>
        <Outlet />
      </div>
      <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
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
