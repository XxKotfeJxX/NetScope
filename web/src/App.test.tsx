import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "./App";
import { AuthProvider } from "./auth/AuthProvider";

function renderApp() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <MemoryRouter>
          <App />
        </MemoryRouter>
      </AuthProvider>
    </QueryClientProvider>,
  );
}

describe("App", () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: string | URL | Request) => {
        const path = String(input);
        const data = path.includes("/api/v1/me")
          ? {
              user: {
                id: "user",
                email: "owner@example.com",
                displayName: "Acme Owner",
                createdAt: "2026-07-24T10:00:00Z",
                updatedAt: "2026-07-24T10:00:00Z",
              },
              workspaces: [
                {
                  id: "workspace",
                  name: "Acme Production",
                  slug: "acme-production",
                  role: "owner",
                  createdBy: "user",
                  createdAt: "2026-07-24T10:00:00Z",
                  updatedAt: "2026-07-24T10:00:00Z",
                },
              ],
              activeWorkspace: {
                id: "workspace",
                name: "Acme Production",
                slug: "acme-production",
                role: "owner",
                createdBy: "user",
                createdAt: "2026-07-24T10:00:00Z",
                updatedAt: "2026-07-24T10:00:00Z",
              },
              sessionExpiresAt: "2026-08-24T10:00:00Z",
            }
          : path.includes("capabilities")
            ? {
                version: "test",
                checks: { dns: { available: true } },
                runtime: {
                  defaultTimeoutMs: 5000,
                  maxTimeoutMs: 30000,
                  runWorkers: 4,
                  probeConcurrency: 8,
                  networkPolicy: "local",
                },
              }
            : {
                items: [],
                page: 1,
                pageSize: 5,
                totalItems: 0,
                totalPages: 0,
              };
        return {
          ok: true,
          status: 200,
          json: async () => data,
        } as Response;
      }),
    );
  });

  it("renders the diagnostic workspace", async () => {
    renderApp();

    expect(
      await screen.findByRole("heading", {
        name: /run a network diagnostic/i,
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("navigation", { name: /main navigation/i }),
    ).toBeInTheDocument();
  });

  it("validates an empty target", async () => {
    renderApp();

    fireEvent.click(
      await screen.findByRole("button", { name: /run diagnostic/i }),
    );

    expect(
      await screen.findByText(/enter a hostname, url, or ip address/i),
    ).toBeInTheDocument();
  });

  it("shows the workspace sign-in when no session exists", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        return {
          ok: false,
          status: 401,
          json: async () => ({
            error: {
              code: "authentication_required",
              message: "Sign in to continue.",
            },
          }),
        } as Response;
      }),
    );

    renderApp();

    expect(
      await screen.findByRole("heading", { name: /return to your workspace/i }),
    ).toBeInTheDocument();
  });
});
