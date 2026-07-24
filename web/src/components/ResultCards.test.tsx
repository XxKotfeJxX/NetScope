import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { CheckResult } from "../api/types";
import { DNSResult } from "./DNSResult";
import { HTTPResult } from "./HTTPResult";
import { PingResult } from "./PingResult";
import { TCPResult } from "./TCPResult";
import { TLSResult } from "./TLSResult";
import { TracerouteResult } from "./TracerouteResult";

const base: Omit<CheckResult, "type" | "data"> = {
  id: "result",
  runId: "run",
  status: "passed",
  durationMs: 42,
  summary: "completed",
  startedAt: "2026-07-23T12:00:00Z",
  completedAt: "2026-07-23T12:00:00Z",
};

describe("result cards", () => {
  afterEach(cleanup);

  it("renders typed probe data", () => {
    render(
      <>
        <DNSResult
          result={{
            ...base,
            type: "dns",
            data: {
              a: ["192.0.2.1"],
              aaaa: [],
              mx: [],
              ns: [],
              txt: [],
              ptr: [],
            },
          }}
        />
        <TCPResult
          result={{
            ...base,
            type: "tcp",
            data: {
              ports: [
                {
                  port: 443,
                  status: "passed",
                  resolvedIp: "192.0.2.1",
                  connectTimeMs: 12,
                },
              ],
            },
          }}
        />
        <HTTPResult
          result={{
            ...base,
            type: "http",
            data: {
              requestedUrl: "https://example.com",
              finalUrl: "https://example.com",
              method: "GET",
              statusCode: 200,
              protocol: "HTTP/2.0",
              redirectChain: [],
              timings: {
                dnsMs: 2,
                connectMs: 4,
                tlsMs: 8,
                ttfbMs: 16,
                totalMs: 20,
              },
              bodyTruncated: false,
            },
          }}
        />
        <TLSResult
          result={{
            ...base,
            type: "tls",
            data: {
              serverName: "example.com",
              sans: ["example.com"],
              daysRemaining: 90,
              hostnameValid: true,
              chainValid: true,
              selfSigned: false,
              expired: false,
              warnings: [],
            },
          }}
        />
        <PingResult
          result={{
            ...base,
            type: "ping",
            data: {
              address: "192.0.2.1",
              packetsSent: 4,
              packetsReceived: 4,
              packetLossPercent: 0,
              minRttMs: 10,
              averageRttMs: 12,
              maxRttMs: 14,
              samples: [
                {
                  sequence: 1,
                  status: "received",
                  address: "192.0.2.1",
                  rttMs: 12,
                },
              ],
            },
          }}
        />
        <TracerouteResult
          result={{
            ...base,
            type: "traceroute",
            data: {
              address: "192.0.2.1",
              reached: true,
              maxHops: 20,
              hops: [
                {
                  number: 1,
                  status: "replied",
                  address: "192.0.2.1",
                  rttMs: 12,
                  destination: true,
                },
              ],
            },
          }}
        />
      </>,
    );

    expect(screen.getAllByText("192.0.2.1").length).toBeGreaterThanOrEqual(4);
    expect(screen.getByText(":443")).toBeInTheDocument();
    expect(screen.getByText(/200 · HTTP\/2.0/)).toBeInTheDocument();
    expect(screen.getByText(/Trust chain/)).toBeInTheDocument();
    expect(screen.getByText(/0%/)).toBeInTheDocument();
    expect(screen.getByText(/Reached 192.0.2.1/)).toBeInTheDocument();
  });
});
