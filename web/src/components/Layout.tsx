import { Activity, Clock3, Code2, Settings2 } from "lucide-react";
import { Link, NavLink, Outlet } from "react-router-dom";

export function Layout() {
  return (
    <div className="app-shell">
      <header className="topbar">
        <Link className="brand" to="/">
          <span className="brand-mark">
            <Activity size={20} />
          </span>
          NetScope
          <span className="version">dev</span>
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
      <Outlet />
    </div>
  );
}
