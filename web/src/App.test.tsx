import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "./App";

function renderApp() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <App />
      </MemoryRouter>
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
        const data = path.includes("capabilities")
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

  it("renders the diagnostic workspace", () => {
    renderApp();

    expect(
      screen.getByRole("heading", { name: /run a network diagnostic/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("navigation", { name: /main navigation/i }),
    ).toBeInTheDocument();
  });

  it("validates an empty target", async () => {
    renderApp();

    fireEvent.click(screen.getByRole("button", { name: /run diagnostic/i }));

    expect(
      await screen.findByText(/enter a hostname, url, or ip address/i),
    ).toBeInTheDocument();
  });
});
