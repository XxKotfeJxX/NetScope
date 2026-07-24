import { type FormEvent, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import type {
  AuditEvent,
  WorkspaceAPIKey,
  WorkspaceMember,
  WorkspaceRole,
} from "../api/types";
import { useAuth } from "../auth/AuthContext";

type WorkspaceTab = "overview" | "members" | "api-keys" | "settings";

const roles: WorkspaceRole[] = ["owner", "admin", "operator", "viewer"];

export function WorkspacePage() {
  const { account, createWorkspace } = useAuth();
  const [tab, setTab] = useState<WorkspaceTab>("overview");
  const [name, setName] = useState("");
  const [creating, setCreating] = useState(false);
  const [members, setMembers] = useState<WorkspaceMember[]>([]);
  const [audit, setAudit] = useState<AuditEvent[]>([]);
  const [apiKeys, setAPIKeys] = useState<WorkspaceAPIKey[]>([]);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<WorkspaceRole>("viewer");
  const [busyMember, setBusyMember] = useState("");
  const [keyName, setKeyName] = useState("");
  const [keyRole, setKeyRole] = useState<"operator" | "viewer">("viewer");
  const [keyTTL, setKeyTTL] = useState("90");
  const [busyKey, setBusyKey] = useState("");
  const [revealedToken, setRevealedToken] = useState("");
  const [error, setError] = useState("");

  const canManage =
    account?.activeWorkspace.role === "owner" ||
    account?.activeWorkspace.role === "admin";

  useEffect(() => {
    if (!account || !canManage) return;
    let active = true;
    Promise.all([
      api.listWorkspaceMembers(),
      api.workspaceAudit(),
      api.listWorkspaceAPIKeys(),
    ])
      .then(([nextMembers, nextAudit, nextAPIKeys]) => {
        if (!active) return;
        setMembers(nextMembers);
        setAudit(nextAudit.items);
        setAPIKeys(nextAPIKeys);
        setError("");
      })
      .catch((reason: unknown) => {
        if (active)
          setError(
            reason instanceof Error ? reason.message : "Request failed.",
          );
      });
    return () => {
      active = false;
    };
  }, [account, canManage]);

  if (!account) return null;

  async function submitWorkspace(event: FormEvent) {
    event.preventDefault();
    if (!name.trim()) return;
    setCreating(true);
    setError("");
    try {
      await createWorkspace(name.trim());
      setName("");
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Request failed.");
    } finally {
      setCreating(false);
    }
  }

  async function submitMember(event: FormEvent) {
    event.preventDefault();
    if (!email.trim()) return;
    setBusyMember("new");
    setError("");
    try {
      const member = await api.addWorkspaceMember(email.trim(), role);
      setMembers((current) => [...current, member]);
      setEmail("");
      await refreshAudit();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Request failed.");
    } finally {
      setBusyMember("");
    }
  }

  async function changeRole(member: WorkspaceMember, next: WorkspaceRole) {
    setBusyMember(member.userId);
    setError("");
    try {
      const updated = await api.updateWorkspaceMember(member.userId, next);
      setMembers((current) =>
        current.map((item) =>
          item.userId === updated.userId ? updated : item,
        ),
      );
      await refreshAudit();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Request failed.");
    } finally {
      setBusyMember("");
    }
  }

  async function removeMember(member: WorkspaceMember) {
    setBusyMember(member.userId);
    setError("");
    try {
      await api.removeWorkspaceMember(member.userId);
      setMembers((current) =>
        current.filter((item) => item.userId !== member.userId),
      );
      await refreshAudit();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Request failed.");
    } finally {
      setBusyMember("");
    }
  }

  async function refreshAudit() {
    const page = await api.workspaceAudit();
    setAudit(page.items);
  }

  async function submitAPIKey(event: FormEvent) {
    event.preventDefault();
    if (!keyName.trim()) return;
    setBusyKey("new");
    setError("");
    try {
      const expiresAt = new Date(
        Date.now() + Number(keyTTL) * 24 * 60 * 60 * 1000,
      ).toISOString();
      const created = await api.createWorkspaceAPIKey({
        name: keyName.trim(),
        role: keyRole,
        expiresAt,
      });
      setAPIKeys((current) => [created, ...current]);
      setKeyName("");
      setRevealedToken(created.token);
      await refreshAudit();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Request failed.");
    } finally {
      setBusyKey("");
    }
  }

  async function revokeAPIKey(key: WorkspaceAPIKey) {
    setBusyKey(key.id);
    setError("");
    try {
      await api.revokeWorkspaceAPIKey(key.id);
      const revokedAt = new Date().toISOString();
      setAPIKeys((current) =>
        current.map((item) =>
          item.id === key.id ? { ...item, revokedAt } : item,
        ),
      );
      await refreshAudit();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Request failed.");
    } finally {
      setBusyKey("");
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
        <TabButton
          active={tab === "overview"}
          onClick={() => setTab("overview")}
        >
          Overview
        </TabButton>
        <Link to="/targets">Targets</Link>
        <TabButton active={tab === "members"} onClick={() => setTab("members")}>
          Members
        </TabButton>
        <TabButton
          active={tab === "api-keys"}
          onClick={() => setTab("api-keys")}
        >
          API Keys
        </TabButton>
        <TabButton
          active={tab === "settings"}
          onClick={() => setTab("settings")}
        >
          Settings
        </TabButton>
      </nav>

      {error && (
        <p className="workspace-error" role="alert">
          {error}
        </p>
      )}

      {tab === "overview" && (
        <>
          <section className="workspace-ledger">
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
            <form className="workspace-create" onSubmit={submitWorkspace}>
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
          <AuditJournal events={audit} visible={canManage} />
        </>
      )}

      {tab === "members" &&
        (canManage ? (
          <MembersPanel
            currentUserID={account.user.id}
            actorRole={account.activeWorkspace.role}
            members={members}
            email={email}
            role={role}
            busyMember={busyMember}
            onEmail={setEmail}
            onRole={setRole}
            onSubmit={submitMember}
            onChangeRole={changeRole}
            onRemove={removeMember}
          />
        ) : (
          <section className="workspace-next">
            <p className="section-label">MEMBERSHIP</p>
            <h2>Admin access required</h2>
            <p>
              Only workspace Owners and Admins can inspect or change the team.
            </p>
          </section>
        ))}

      {tab === "api-keys" &&
        (canManage ? (
          <APIKeysPanel
            keys={apiKeys}
            name={keyName}
            role={keyRole}
            ttl={keyTTL}
            busyKey={busyKey}
            revealedToken={revealedToken}
            onName={setKeyName}
            onRole={setKeyRole}
            onTTL={setKeyTTL}
            onSubmit={submitAPIKey}
            onRevoke={revokeAPIKey}
            onDismissToken={() => setRevealedToken("")}
          />
        ) : (
          <section className="workspace-next">
            <p className="section-label">PROGRAMMATIC ACCESS</p>
            <h2>Admin access required</h2>
            <p>Only workspace Owners and Admins can manage API keys.</p>
          </section>
        ))}

      {tab === "settings" && (
        <section className="workspace-next">
          <p className="section-label">WORKSPACE POLICY</p>
          <h2>Settings</h2>
          <p>
            Workspace identity, security policy, and destructive actions live
            here.
          </p>
        </section>
      )}
    </main>
  );
}

function APIKeysPanel({
  keys,
  name,
  role,
  ttl,
  busyKey,
  revealedToken,
  onName,
  onRole,
  onTTL,
  onSubmit,
  onRevoke,
  onDismissToken,
}: {
  keys: WorkspaceAPIKey[];
  name: string;
  role: "operator" | "viewer";
  ttl: string;
  busyKey: string;
  revealedToken: string;
  onName: (name: string) => void;
  onRole: (role: "operator" | "viewer") => void;
  onTTL: (ttl: string) => void;
  onSubmit: (event: FormEvent) => void;
  onRevoke: (key: WorkspaceAPIKey) => void;
  onDismissToken: () => void;
}) {
  const [referenceTime] = useState(Date.now);

  return (
    <section className="api-keys-panel">
      <div className="members-heading">
        <div>
          <p className="section-label">
            PROGRAMMATIC ACCESS / {keys.length} KEYS
          </p>
          <h2>Workspace API keys</h2>
        </div>
        <p>Secrets are shown once. Stored credentials are one-way hashes.</p>
      </div>
      {revealedToken && (
        <div className="key-reveal" role="status">
          <div>
            <strong>Copy this key now</strong>
            <span>It cannot be recovered after this notice is closed.</span>
          </div>
          <code>{revealedToken}</code>
          <button
            type="button"
            onClick={() => void navigator.clipboard.writeText(revealedToken)}
          >
            Copy
          </button>
          <button type="button" onClick={onDismissToken}>
            Done
          </button>
        </div>
      )}
      <form className="member-invite api-key-create" onSubmit={onSubmit}>
        <label>
          Key name
          <input
            value={name}
            onChange={(event) => onName(event.target.value)}
            placeholder="CI diagnostics"
            maxLength={100}
            required
          />
        </label>
        <label>
          Role
          <select
            value={role}
            onChange={(event) =>
              onRole(event.target.value as "operator" | "viewer")
            }
          >
            <option value="viewer">viewer</option>
            <option value="operator">operator</option>
          </select>
        </label>
        <label>
          Expires
          <select value={ttl} onChange={(event) => onTTL(event.target.value)}>
            <option value="30">30 days</option>
            <option value="90">90 days</option>
            <option value="365">365 days</option>
          </select>
        </label>
        <button type="submit" disabled={busyKey === "new"}>
          {busyKey === "new" ? "Creating…" : "Create key"}
        </button>
      </form>
      <div className="api-key-list">
        {keys.length === 0 ? (
          <p>No API keys issued for this workspace.</p>
        ) : (
          keys.map((key) => {
            const expired = new Date(key.expiresAt).getTime() <= referenceTime;
            const status = key.revokedAt
              ? "revoked"
              : expired
                ? "expired"
                : "active";
            return (
              <article key={key.id} className="api-key-row">
                <div>
                  <strong>{key.name}</strong>
                  <code>{key.prefix}…</code>
                </div>
                <span>{key.role}</span>
                <span className={`key-status is-${status}`}>{status}</span>
                <time dateTime={key.expiresAt}>
                  expires {new Date(key.expiresAt).toLocaleDateString()}
                </time>
                <button
                  type="button"
                  disabled={status !== "active" || busyKey === key.id}
                  onClick={() => onRevoke(key)}
                >
                  Revoke
                </button>
              </article>
            );
          })
        )}
      </div>
    </section>
  );
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: string;
}) {
  return (
    <button className={active ? "active" : ""} type="button" onClick={onClick}>
      {children}
    </button>
  );
}

function MembersPanel({
  currentUserID,
  actorRole,
  members,
  email,
  role,
  busyMember,
  onEmail,
  onRole,
  onSubmit,
  onChangeRole,
  onRemove,
}: {
  currentUserID: string;
  actorRole: WorkspaceRole;
  members: WorkspaceMember[];
  email: string;
  role: WorkspaceRole;
  busyMember: string;
  onEmail: (email: string) => void;
  onRole: (role: WorkspaceRole) => void;
  onSubmit: (event: FormEvent) => void;
  onChangeRole: (member: WorkspaceMember, role: WorkspaceRole) => void;
  onRemove: (member: WorkspaceMember) => void;
}) {
  const assignableRoles =
    actorRole === "owner" ? roles : (["operator", "viewer"] as WorkspaceRole[]);
  return (
    <section className="members-panel">
      <div className="members-heading">
        <div>
          <p className="section-label">TEAM / {members.length} ACCOUNTS</p>
          <h2>Workspace members</h2>
        </div>
        <p>Accounts must register before they can be added.</p>
      </div>
      <form className="member-invite" onSubmit={onSubmit}>
        <label>
          Account email
          <input
            type="email"
            value={email}
            onChange={(event) => onEmail(event.target.value)}
            placeholder="operator@example.com"
            required
          />
        </label>
        <label>
          Initial role
          <select
            value={role}
            onChange={(event) => onRole(event.target.value as WorkspaceRole)}
          >
            {assignableRoles.map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </select>
        </label>
        <button type="submit" disabled={busyMember === "new"}>
          {busyMember === "new" ? "Adding…" : "Add member"}
        </button>
      </form>
      <div className="member-list">
        {members.map((member) => {
          const privileged = member.role === "owner" || member.role === "admin";
          const canEdit =
            member.userId !== currentUserID &&
            (actorRole === "owner" || !privileged);
          return (
            <article key={member.userId} className="member-row">
              <span className="member-avatar" aria-hidden="true">
                {member.displayName.slice(0, 2).toUpperCase()}
              </span>
              <div>
                <strong>{member.displayName}</strong>
                <span>{member.email}</span>
              </div>
              <select
                aria-label={`Role for ${member.displayName}`}
                value={member.role}
                disabled={!canEdit || busyMember === member.userId}
                onChange={(event) =>
                  onChangeRole(member, event.target.value as WorkspaceRole)
                }
              >
                {(actorRole === "owner" ? roles : assignableRoles).map(
                  (value) => (
                    <option key={value} value={value}>
                      {value}
                    </option>
                  ),
                )}
              </select>
              <button
                className="member-remove"
                type="button"
                disabled={!canEdit || busyMember === member.userId}
                onClick={() => onRemove(member)}
              >
                Remove
              </button>
            </article>
          );
        })}
      </div>
    </section>
  );
}

function AuditJournal({
  events,
  visible,
}: {
  events: AuditEvent[];
  visible: boolean;
}) {
  if (!visible) return null;
  return (
    <section className="audit-journal">
      <div>
        <p className="section-label">AUDIT JOURNAL</p>
        <h2>Recent control-plane changes</h2>
      </div>
      {events.length === 0 ? (
        <p className="audit-empty">No membership changes recorded yet.</p>
      ) : (
        <ol>
          {events.map((event) => (
            <li key={event.id}>
              <span>{event.action.replaceAll(".", " / ")}</span>
              <time dateTime={event.createdAt}>
                {new Date(event.createdAt).toLocaleString()}
              </time>
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}
