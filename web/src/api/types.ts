export type CheckType = "dns" | "tcp" | "http" | "tls";
export type RunStatus =
  | "queued"
  | "running"
  | "completed"
  | "partial"
  | "failed"
  | "cancelled"
  | "interrupted";
export type CheckStatus =
  "pending" | "running" | "passed" | "warning" | "failed" | "cancelled";

export interface RunOptions {
  timeoutMs: number;
  tcpPorts?: number[];
  followRedirects: boolean;
  maxRedirects: number;
  ipVersion: string;
}

export interface CheckResult {
  id: string;
  runId: string;
  type: CheckType;
  status: CheckStatus;
  durationMs: number;
  summary?: string;
  data: unknown;
  errorCode?: string;
  errorMessage?: string;
  startedAt: string;
  completedAt: string;
}

export interface DiagnosticRun {
  id: string;
  target: string;
  normalizedHost: string;
  normalizedUrl?: string;
  status: RunStatus;
  checks: CheckType[];
  options: RunOptions;
  results: CheckResult[];
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  cancelledAt?: string;
}

export interface RunPage {
  items: DiagnosticRun[];
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
}

export interface Capability {
  available: boolean;
  reason?: string;
}

export interface Capabilities {
  version: string;
  checks: Record<string, Capability>;
  runtime: {
    defaultTimeoutMs: number;
    maxTimeoutMs: number;
    runWorkers: number;
    probeConcurrency: number;
    networkPolicy: string;
  };
}

export interface DNSData {
  a: string[];
  aaaa: string[];
  cname?: string;
  mx: Array<{ host: string; preference: number }>;
  ns: string[];
  txt: string[];
  ptr: string[];
  errors?: Record<string, { code: string; message: string }>;
}

export interface APIError {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
    requestId?: string;
  };
}
