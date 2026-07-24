import { Moon, Sun } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { initialTheme, saveTheme, type Theme } from "../theme";

const repositoryURL = "https://github.com/XxKotfeJxX/NetScope";

export function DocsPage() {
  const { account } = useAuth();
  const [theme, setTheme] = useState<Theme>(initialTheme);
  const toggleTheme = () => {
    const next = theme === "light" ? "dark" : "light";
    setTheme(next);
    saveTheme(next);
  };

  return (
    <div className="docs-shell">
      <a className="skip-link" href="#docs-content">
        Skip to documentation
      </a>
      <header className="docs-topbar">
        <Link className="brand" to="/">
          NETSCOPE<span aria-hidden="true">/</span>
        </Link>
        <nav aria-label="Documentation navigation">
          <a href="#quickstart">Quickstart</a>
          <a href="#api">API</a>
          <a href="#operations">Operations</a>
        </nav>
        <div>
          <button
            className="icon-button"
            type="button"
            aria-label={`Switch to ${theme === "light" ? "dark" : "light"} theme`}
            onClick={toggleTheme}
          >
            {theme === "light" ? (
              <Moon aria-hidden="true" size={16} />
            ) : (
              <Sun aria-hidden="true" size={16} />
            )}
          </button>
          <Link className="docs-workspace-link" to="/">
            {account ? "Open workspace →" : "Sign in →"}
          </Link>
        </div>
      </header>

      <main className="docs-content" id="docs-content" tabIndex={-1}>
        <aside aria-label="On this page">
          <p className="section-label">Public manual / v1</p>
          <a href="#quickstart">Quickstart</a>
          <a href="#api">API contract</a>
          <a href="#authentication">Authentication</a>
          <a href="#errors">Errors and limits</a>
          <a href="#operations">Production operations</a>
        </aside>

        <article>
          <header className="docs-hero">
            <p className="section-label">Network field manual</p>
            <h1>NetScope documentation</h1>
            <p>
              Run bounded network diagnostics, monitor explicit targets, and
              publish read-only technical evidence from one stable API.
            </p>
          </header>

          <section id="quickstart">
            <p className="section-label">01 / Quickstart</p>
            <h2>Inspect a target</h2>
            <ol className="docs-steps">
              <li>
                <strong>Create an account</strong>
                <span>
                  Your first workspace is created with the owner role.
                </span>
              </li>
              <li>
                <strong>Enter one explicit target</strong>
                <span>
                  Use a hostname, URL, or single IP address—not a CIDR.
                </span>
              </li>
              <li>
                <strong>Choose checks and run</strong>
                <span>
                  DNS, TCP, TLS, HTTP, ping, and traceroute run within limits.
                </span>
              </li>
            </ol>
            <Link className="docs-action" to="/?target=example.com">
              Inspect example.com →
            </Link>
          </section>

          <section id="api">
            <p className="section-label">02 / API contract</p>
            <h2>Automate diagnostics</h2>
            <p>
              Versioned endpoints live under <code>/api/v1</code>. Use a
              workspace API key as a Bearer token and pass the workspace ID
              explicitly when the key or account can access more than one.
            </p>
            <pre className="docs-code">
              <code>{`curl -X POST https://netscope.example/api/v1/runs \\
  -H "Authorization: Bearer ns_key_..." \\
  -H "X-Workspace-ID: <workspace-id>" \\
  -H "Content-Type: application/json" \\
  -d '{
    "target": "example.com",
    "checks": ["dns", "tcp", "tls", "http"],
    "options": {
      "timeoutMs": 5000,
      "tcpPorts": [80, 443],
      "httpMethod": "GET",
      "followRedirects": true,
      "maxRedirects": 5,
      "ipVersion": "auto",
      "pingPackets": 4,
      "maxHops": 20
    }
  }'`}</code>
            </pre>
            <a
              className="docs-action"
              href={`${repositoryURL}/blob/dev/api/openapi.yaml`}
              target="_blank"
              rel="noreferrer"
            >
              Open the complete OpenAPI 3.1 contract ↗
            </a>
          </section>

          <section id="authentication">
            <p className="section-label">03 / Authentication</p>
            <h2>Sessions, keys, and roles</h2>
            <div className="docs-facts">
              <div>
                <h3>Browser sessions</h3>
                <p>
                  HttpOnly, SameSite cookies with strict origin checks for
                  unsafe requests in production.
                </p>
              </div>
              <div>
                <h3>API keys</h3>
                <p>
                  One-time secrets, hashed at rest, scoped to one workspace and
                  capped at operator permissions.
                </p>
              </div>
              <div>
                <h3>Workspace roles</h3>
                <p>
                  Owner, admin, operator, and viewer permissions are enforced
                  server-side.
                </p>
              </div>
            </div>
          </section>

          <section id="errors">
            <p className="section-label">04 / Errors and limits</p>
            <h2>Predictable failure responses</h2>
            <p>
              Errors use a stable envelope and never return internal failure
              details. Keep the request ID when reporting a problem.
            </p>
            <pre className="docs-code">
              <code>{`{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "Too many requests. Try again shortly.",
    "requestId": "..."
  }
}`}</code>
            </pre>
            <p>
              Rate-limited responses return HTTP 429 with{" "}
              <code>RateLimit-Limit</code>, <code>RateLimit-Remaining</code>,{" "}
              <code>RateLimit-Reset</code>, and <code>Retry-After</code>.
            </p>
          </section>

          <section id="operations">
            <p className="section-label">05 / Operations</p>
            <h2>Run NetScope in production</h2>
            <ul className="docs-checklist">
              <li>Terminate TLS at the trusted reverse proxy.</li>
              <li>
                Keep SSRF policy on public-only targets for internet
                deployments.
              </li>
              <li>
                Restrict metrics and database access to the operations network.
              </li>
              <li>Back up PostgreSQL and verify restores on a schedule.</li>
              <li>Alert on readiness, 5xx responses, panics, and restarts.</li>
            </ul>
            <a
              className="docs-action"
              href={`${repositoryURL}/tree/dev/docs`}
              target="_blank"
              rel="noreferrer"
            >
              Read deployment and security runbooks ↗
            </a>
          </section>
        </article>
      </main>
    </div>
  );
}
