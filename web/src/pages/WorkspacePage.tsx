import { type FormEvent, useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

export function WorkspacePage() {
  const { account, createWorkspace } = useAuth();
  const [name, setName] = useState("");
  const [creating, setCreating] = useState(false);

  if (!account) return null;

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!name.trim()) return;
    setCreating(true);
    try {
      await createWorkspace(name.trim());
      setName("");
    } finally {
      setCreating(false);
    }
  }

  return (
    <main className="page workspace-page">
      <header className="page-heading">
        <div>
          <p className="section-label">WORKSPACE / CONTROL PLANE</p>
          <h1>{account.activeWorkspace.name}</h1>
          <p>
            Shared network evidence for{" "}
            <span className="mono-value">{account.user.email}</span>.
          </p>
        </div>
        <span className="workspace-role">{account.activeWorkspace.role}</span>
      </header>

      <nav className="context-tabs" aria-label="Workspace sections">
        <a className="active" href="#overview">
          Overview
        </a>
        <Link to="/targets">Targets</Link>
        <a href="#members">Members</a>
        <a href="#api-keys">API Keys</a>
        <a href="#settings">Settings</a>
      </nav>

      <section id="overview" className="workspace-ledger">
        <div>
          <p className="section-label">ACTIVE CONTEXT</p>
          <h2>{account.activeWorkspace.name}</h2>
          <dl>
            <div>
              <dt>Workspace ID</dt>
              <dd>{account.activeWorkspace.id}</dd>
            </div>
            <div>
              <dt>Your role</dt>
              <dd>{account.activeWorkspace.role}</dd>
            </div>
            <div>
              <dt>Available workspaces</dt>
              <dd>{account.workspaces.length}</dd>
            </div>
          </dl>
        </div>
        <form className="workspace-create" onSubmit={submit}>
          <p className="section-label">NEW WORKSPACE</p>
          <label>
            Name
            <input
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="Platform Operations"
              maxLength={100}
              required
            />
          </label>
          <button type="submit" disabled={creating}>
            {creating ? "Creating…" : "Create workspace"}
          </button>
        </form>
      </section>

      <section id="members" className="workspace-next">
        <p className="section-label">COLLABORATION LAYER</p>
        <p>
          Member roles, API keys, shared report links, comments, and the audit
          journal are enabled in the next workspace revisions.
        </p>
      </section>
    </main>
  );
}
