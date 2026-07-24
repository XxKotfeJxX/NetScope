import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MonitoringPage } from "./MonitoringPage";
import { TargetsPage } from "./TargetsPage";

function renderPage(page: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>{page}</MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("monitoring pages", () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("renders saved targets as an operational journal", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => ({
        ok: true,
        status: 200,
        json: async () => ({
          items: [
            {
              id: "target-1",
              name: "Production API",
              address: "api.example.com",
              tags: ["production"],
              checks: ["dns", "http"],
              options: {},
              intervalSeconds: 300,
              failureThreshold: 3,
              enabled: true,
              consecutiveFailures: 0,
              status: "operational",
              lastLatencyMs: 124,
              nextCheckAt: "2026-07-24T12:05:00Z",
              createdAt: "2026-07-24T12:00:00Z",
              updatedAt: "2026-07-24T12:00:00Z",
            },
          ],
          page: 1,
          pageSize: 100,
          totalItems: 1,
          totalPages: 1,
        }),
      })),
    );
    renderPage(<TargetsPage />);

    expect(await screen.findByText("Production API")).toBeInTheDocument();
    expect(screen.getByText("124 ms")).toBeInTheDocument();
    expect(screen.getByLabelText("Status: Operational")).toBeInTheDocument();
  });

  it("renders the live operational summary and journal", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => ({
        ok: true,
        status: 200,
        json: async () => ({
          activeTargets: 12,
          warningTargets: 1,
          unavailableTargets: 0,
          recentChecks: [],
        }),
      })),
    );
    renderPage(<MonitoringPage />);

    expect(await screen.findByText("12 active targets")).toBeInTheDocument();
    expect(screen.getByText("1 warning")).toBeInTheDocument();
    expect(screen.getByText("0 unavailable")).toBeInTheDocument();
  });
});
