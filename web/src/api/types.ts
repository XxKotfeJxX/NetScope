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
  httpMethod: string;
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

export interface TCPData {
  ports: Array<{
    port: number;
    status: string;
    resolvedIp?: string;
    connectTimeMs: number;
    errorCode?: string;
    errorMessage?: string;
  }>;
}

export interface HTTPData {
  requestedUrl: string;
  finalUrl?: string;
  method: string;
  statusCode?: number;
  protocol?: string;
  contentType?: string;
  contentLength?: number;
  redirectChain: Array<{ url: string; statusCode: number }>;
  resolvedIp?: string;
  remoteAddress?: string;
  timings: {
    dnsMs: number;
    connectMs: number;
    tlsMs: number;
    ttfbMs: number;
    totalMs: number;
  };
  bodySha256?: string;
  bodyPreview?: string;
  bodyTruncated: boolean;
}

export interface TLSData {
  tlsVersion?: string;
  cipherSuite?: string;
  serverName: string;
  resolvedIp?: string;
  subject?: string;
  issuer?: string;
  serialNumber?: string;
  sans: string[];
  validFrom?: string;
  validUntil?: string;
  daysRemaining: number;
  hostnameValid: boolean;
  chainValid: boolean;
  selfSigned: boolean;
  expired: boolean;
  warnings: string[];
}

export interface APIError {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
    requestId?: string;
  };
}
