import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { CheckResult, CheckType, DiagnosticRun } from "../api/types";
import { RunComparison } from "./CompareRunsPage";

function result(type: CheckType, durationMs: number): CheckResult {
  return {
    id: `${type}-result`,
    runId: "run",
    type,
    status: "passed",
    durationMs,
    summary: `${type} complete`,
    data: {},
    startedAt: "2026-07-24T10:00:00Z",
    completedAt: "2026-07-24T10:00:01Z",
  };
}

function run(
  id: string,
  createdAt: string,
  results: CheckResult[],
): DiagnosticRun {
  return {
    id,
    target: "example.com",
    normalizedHost: "example.com",
    status: "completed",
    checks: results.map((item) => item.type),
    options: {
      timeoutMs: 5000,
      tcpPorts: [80, 443],
      httpMethod: "GET",
      followRedirects: true,
      maxRedirects: 5,
      ipVersion: "auto",
      pingPackets: 4,
      maxHops: 20,
    },
    results,
    createdAt,
    completedAt: createdAt,
  };
}

describe("RunComparison", () => {
  afterEach(cleanup);

  it("shows timing deltas and checks missing from one run", () => {
    const previous = run("previous", "2026-07-24T10:00:00Z", [
      result("dns", 12),
    ]);
    const current = run("current", "2026-07-24T11:00:00Z", [
      result("dns", 30),
      result("ping", 20),
    ]);

    render(<RunComparison left={previous} right={current} />);

    expect(screen.getByText("+18 ms")).toHaveClass("comparison-regression");
    expect(screen.getByText("PING")).toBeInTheDocument();
    expect(screen.getByText("Not requested")).toBeInTheDocument();
  });
});
