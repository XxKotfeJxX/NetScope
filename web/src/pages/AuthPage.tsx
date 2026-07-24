import { type FormEvent, useState } from "react";
import { NetScopeAPIError } from "../api/client";
import { useAuth } from "../auth/AuthContext";

export function AuthPage() {
  const { login, register, sessionExpired } = useAuth();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [workspaceName, setWorkspaceName] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      if (mode === "login") {
        await login(email, password);
      } else {
        await register({ email, password, displayName, workspaceName });
      }
    } catch (cause) {
      setError(
        cause instanceof NetScopeAPIError
          ? cause.message
          : "NetScope could not complete authentication.",
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="auth-shell">
      <section className="auth-intro">
        <span className="brand auth-brand">
          NETSCOPE<span aria-hidden="true">/</span>
        </span>
        <p className="section-label">COLLABORATIVE NETWORK OPERATIONS</p>
        <h1>Inspect the route. Share the evidence.</h1>
        <p>
          Focused diagnostics, scheduled monitoring, and a common technical
          record for your workspace.
        </p>
        <div className="auth-route" aria-hidden="true">
          <span>● Identity</span>
          <i />
          <span>● Workspace</span>
          <i />
          <span>● Network evidence</span>
        </div>
      </section>

      <section className="auth-panel" aria-labelledby="auth-title">
        <div className="auth-mode" role="tablist" aria-label="Authentication">
          <button
            type="button"
            className={mode === "login" ? "active" : ""}
            onClick={() => {
              setMode("login");
              setError("");
            }}
          >
            Sign in
          </button>
          <button
            type="button"
            className={mode === "register" ? "active" : ""}
            onClick={() => {
              setMode("register");
              setError("");
            }}
          >
            Create account
          </button>
        </div>
        <p className="section-label">SECURE SESSION</p>
        <h2 id="auth-title">
          {mode === "login" ? "Return to your workspace" : "Create a workspace"}
        </h2>
        {sessionExpired && mode === "login" && (
          <div className="auth-notice" role="alert">
            <strong>Session expired</strong>
            <span>
              The server no longer accepts this session. Sign in again to
              continue.
            </span>
          </div>
        )}
        <form onSubmit={submit}>
          {mode === "register" && (
            <>
              <label>
                Your name
                <input
                  autoComplete="name"
                  value={displayName}
                  onChange={(event) => setDisplayName(event.target.value)}
                  required
                  maxLength={80}
                />
              </label>
              <label>
                Workspace
                <input
                  value={workspaceName}
                  onChange={(event) => setWorkspaceName(event.target.value)}
                  placeholder="Acme Production"
                  maxLength={100}
                />
              </label>
            </>
          )}
          <label>
            Email
            <input
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
            />
          </label>
          <label>
            Password
            <input
              type="password"
              autoComplete={
                mode === "login" ? "current-password" : "new-password"
              }
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              minLength={10}
              maxLength={128}
              required
            />
          </label>
          {error && <p className="form-error">{error}</p>}
          <button
            className="primary-button"
            type="submit"
            disabled={submitting}
          >
            {submitting
              ? "Authenticating…"
              : mode === "login"
                ? "Open workspace"
                : "Create account"}
          </button>
        </form>
        <p className="auth-security">
          Passwords use Argon2id. Session tokens are revocable and stored only
          in an HttpOnly cookie.
        </p>
      </section>
    </main>
  );
}
