export type CheckType = "dns" | "tcp" | "http" | "tls" | "ping" | "traceroute";
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
  ipVersion: "auto" | "ipv4" | "ipv6";
  pingPackets: number;
  maxHops: number;
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
  workspaceId: string;
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

export interface PingData {
  address: string;
  packetsSent: number;
  packetsReceived: number;
  packetLossPercent: number;
  minRttMs: number;
  averageRttMs: number;
  maxRttMs: number;
  samples: Array<{
    sequence: number;
    status: "received" | "timeout";
    address?: string;
    rttMs?: number;
  }>;
}

export interface TracerouteData {
  address: string;
  reached: boolean;
  maxHops: number;
  hops: Array<{
    number: number;
    status: "replied" | "timeout";
    address?: string;
    rttMs?: number;
    destination: boolean;
  }>;
}

export type TargetStatus =
  | "pending"
  | "operational"
  | "warning"
  | "unavailable"
  | "paused"
  | "maintenance";

export interface TargetInput {
  name: string;
  address: string;
  tags: string[];
  checks: CheckType[];
  options: RunOptions;
  intervalSeconds: number;
  failureThreshold: number;
}

export interface MonitoredTarget extends TargetInput {
  id: string;
  workspaceId: string;
  enabled: boolean;
  consecutiveFailures: number;
  status: TargetStatus;
  lastCheckedAt?: string;
  lastLatencyMs?: number;
  tlsExpiresAt?: string;
  nextCheckAt: string;
  createdAt: string;
  updatedAt: string;
}

export interface TargetPage {
  items: MonitoredTarget[];
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
}

export interface MonitoringCheck {
  id: string;
  targetId: string;
  runId?: string;
  status: TargetStatus;
  latencyMs?: number;
  tlsExpiresAt?: string;
  errorMessage?: string;
  checkedAt?: string;
  createdAt: string;
}

export interface MonitoringCheckPage {
  items: MonitoringCheck[];
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
}

export interface MaintenanceWindow {
  id: string;
  targetId: string;
  startsAt: string;
  endsAt: string;
  reason: string;
  createdAt: string;
}

export interface NotificationChannel {
  id: string;
  targetId: string;
  kind: "email" | "webhook";
  destination: string;
  enabled: boolean;
  createdAt: string;
}

export interface MonitoringJournalEntry extends MonitoringCheck {
  targetName: string;
  targetAddress: string;
}

export interface MonitoringOverview {
  activeTargets: number;
  warningTargets: number;
  unavailableTargets: number;
  recentChecks: MonitoringJournalEntry[];
}

export type WorkspaceRole = "owner" | "admin" | "operator" | "viewer";

export interface User {
  id: string;
  email: string;
  displayName: string;
  createdAt: string;
  updatedAt: string;
}

export interface Workspace {
  id: string;
  name: string;
  slug: string;
  role: WorkspaceRole;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface CurrentAccount {
  user: User;
  workspaces: Workspace[];
  activeWorkspace: Workspace;
  sessionExpiresAt: string;
}

export interface WorkspaceMember {
  userId: string;
  email: string;
  displayName: string;
  role: WorkspaceRole;
  joinedAt: string;
}

export interface AuditEvent {
  id: string;
  workspaceId: string;
  actorUserId: string;
  action: string;
  resourceType: string;
  resourceId?: string;
  metadata: Record<string, unknown>;
  createdAt: string;
}

export interface AuditPage {
  items: AuditEvent[];
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
}

export interface WorkspaceAPIKey {
  id: string;
  workspaceId: string;
  name: string;
  prefix: string;
  role: "operator" | "viewer";
  createdBy: string;
  expiresAt: string;
  lastUsedAt?: string;
  revokedAt?: string;
  createdAt: string;
}

export interface CreatedWorkspaceAPIKey extends WorkspaceAPIKey {
  token: string;
}

export interface ReportComment {
  id: string;
  workspaceId: string;
  runId: string;
  authorId: string;
  authorName: string;
  authorEmail: string;
  body: string;
  createdAt: string;
  updatedAt: string;
}

export interface PublicReportLink {
  id: string;
  workspaceId: string;
  runId: string;
  tokenPrefix: string;
  createdBy: string;
  expiresAt: string;
  revokedAt?: string;
  lastViewedAt?: string;
  createdAt: string;
}

export interface CreatedPublicReportLink extends PublicReportLink {
  token: string;
}

export interface PublicReport {
  workspaceName: string;
  publishedAt: string;
  expiresAt: string;
  run: DiagnosticRun;
}

export interface APIError {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
    requestId?: string;
  };
}
