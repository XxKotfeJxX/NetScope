import type {
  APIError,
  AuditPage,
  Capabilities,
  CheckType,
  CreatedWorkspaceAPIKey,
  CreatedPublicReportLink,
  CurrentAccount,
  DiagnosticRun,
  MaintenanceWindow,
  MonitoredTarget,
  MonitoringCheckPage,
  MonitoringOverview,
  NotificationChannel,
  PublicReport,
  PublicReportLink,
  ReportComment,
  RunOptions,
  RunPage,
  TargetInput,
  TargetPage,
  Workspace,
  WorkspaceAPIKey,
  WorkspaceMember,
  WorkspaceRole,
} from "./types";

const API_BASE = import.meta.env.VITE_API_URL ?? "";

export class NetScopeAPIError extends Error {
  readonly code: string;
  readonly requestId?: string;

  constructor(payload: APIError) {
    super(payload.error.message);
    this.name = "NetScopeAPIError";
    this.code = payload.error.code;
    this.requestId = payload.error.requestId;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const workspaceID = window.localStorage.getItem("netscope.workspace");
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      Accept: "application/json",
      ...(workspaceID ? { "X-Workspace-ID": workspaceID } : {}),
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({
      error: {
        code: "network_error",
        message: "NetScope could not complete the request.",
      },
    }))) as APIError;
    throw new NetScopeAPIError(payload);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

export const api = {
  capabilities: () => request<Capabilities>("/api/v1/capabilities"),
  register: (payload: {
    email: string;
    password: string;
    displayName: string;
    workspaceName: string;
  }) =>
    request<CurrentAccount>("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  login: (payload: { email: string; password: string }) =>
    request<CurrentAccount>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  logout: () =>
    request<void>("/api/v1/auth/logout", {
      method: "POST",
    }),
  currentAccount: async () => {
    try {
      return await request<CurrentAccount>("/api/v1/me");
    } catch (error) {
      if (
        error instanceof NetScopeAPIError &&
        error.code === "workspace_not_found"
      ) {
        window.localStorage.removeItem("netscope.workspace");
        return request<CurrentAccount>("/api/v1/me");
      }
      throw error;
    }
  },
  createWorkspace: (name: string) =>
    request<Workspace>("/api/v1/workspaces", {
      method: "POST",
      body: JSON.stringify({ name }),
    }),
  listWorkspaceMembers: () =>
    request<WorkspaceMember[]>("/api/v1/workspace/members"),
  addWorkspaceMember: (email: string, role: WorkspaceRole) =>
    request<WorkspaceMember>("/api/v1/workspace/members", {
      method: "POST",
      body: JSON.stringify({ email, role }),
    }),
  updateWorkspaceMember: (userID: string, role: WorkspaceRole) =>
    request<WorkspaceMember>(`/api/v1/workspace/members/${userID}`, {
      method: "PATCH",
      body: JSON.stringify({ role }),
    }),
  removeWorkspaceMember: (userID: string) =>
    request<void>(`/api/v1/workspace/members/${userID}`, {
      method: "DELETE",
    }),
  workspaceAudit: () =>
    request<AuditPage>("/api/v1/workspace/audit?page=1&pageSize=20"),
  listWorkspaceAPIKeys: () =>
    request<WorkspaceAPIKey[]>("/api/v1/workspace/api-keys"),
  createWorkspaceAPIKey: (payload: {
    name: string;
    role: "operator" | "viewer";
    expiresAt: string;
  }) =>
    request<CreatedWorkspaceAPIKey>("/api/v1/workspace/api-keys", {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  revokeWorkspaceAPIKey: (keyID: string) =>
    request<void>(`/api/v1/workspace/api-keys/${keyID}`, {
      method: "DELETE",
    }),

  createRun: (payload: {
    target: string;
    checks: CheckType[];
    options: RunOptions;
  }) =>
    request<{ id: string; status: string; createdAt: string }>("/api/v1/runs", {
      method: "POST",
      body: JSON.stringify(payload),
    }),

  getRun: (id: string) => request<DiagnosticRun>(`/api/v1/runs/${id}`),
  runComments: (id: string) =>
    request<ReportComment[]>(`/api/v1/runs/${id}/comments`),
  createRunComment: (id: string, body: string) =>
    request<ReportComment>(`/api/v1/runs/${id}/comments`, {
      method: "POST",
      body: JSON.stringify({ body }),
    }),
  deleteRunComment: (runID: string, commentID: string) =>
    request<void>(`/api/v1/runs/${runID}/comments/${commentID}`, {
      method: "DELETE",
    }),
  publicReportLinks: (id: string) =>
    request<PublicReportLink[]>(`/api/v1/runs/${id}/public-links`),
  createPublicReportLink: (id: string, expiresAt?: string) =>
    request<CreatedPublicReportLink>(`/api/v1/runs/${id}/public-links`, {
      method: "POST",
      body: JSON.stringify(expiresAt ? { expiresAt } : {}),
    }),
  revokePublicReportLink: (runID: string, linkID: string) =>
    request<void>(`/api/v1/runs/${runID}/public-links/${linkID}`, {
      method: "DELETE",
    }),
  publicReport: (token: string) =>
    request<PublicReport>(
      `/api/v1/public/reports/${encodeURIComponent(token)}`,
    ),

  listRuns: (page = 1, pageSize = 20, status = "") => {
    const query = new URLSearchParams({
      page: String(page),
      pageSize: String(pageSize),
    });
    if (status) query.set("status", status);
    return request<RunPage>(`/api/v1/runs?${query}`);
  },

  cancelRun: (id: string) =>
    request<void>(`/api/v1/runs/${id}/cancel`, { method: "POST" }),

  eventURL: (id: string) => {
    const query = workspaceQuery();
    return `${API_BASE}/api/v1/runs/${id}/events${query ? `?${query}` : ""}`;
  },
  exportURL: (id: string, format: "json" | "csv" = "json") => {
    const query = new URLSearchParams({ format });
    const workspaceID = window.localStorage.getItem("netscope.workspace");
    if (workspaceID) query.set("workspaceId", workspaceID);
    return `${API_BASE}/api/v1/runs/${id}/export?${query}`;
  },

  listTargets: () => request<TargetPage>("/api/v1/targets?page=1&pageSize=100"),
  createTarget: (payload: TargetInput) =>
    request<MonitoredTarget>("/api/v1/targets", {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  getTarget: (id: string) => request<MonitoredTarget>(`/api/v1/targets/${id}`),
  updateTarget: (id: string, payload: TargetInput) =>
    request<MonitoredTarget>(`/api/v1/targets/${id}`, {
      method: "PUT",
      body: JSON.stringify(payload),
    }),
  deleteTarget: (id: string) =>
    request<void>(`/api/v1/targets/${id}`, { method: "DELETE" }),
  pauseTarget: (id: string) =>
    request<void>(`/api/v1/targets/${id}/pause`, { method: "POST" }),
  resumeTarget: (id: string) =>
    request<void>(`/api/v1/targets/${id}/resume`, { method: "POST" }),
  targetChecks: (id: string) =>
    request<MonitoringCheckPage>(
      `/api/v1/targets/${id}/checks?page=1&pageSize=100`,
    ),
  maintenanceWindows: (id: string) =>
    request<MaintenanceWindow[]>(`/api/v1/targets/${id}/maintenance`),
  createMaintenanceWindow: (
    id: string,
    payload: { startsAt: string; endsAt: string; reason: string },
  ) =>
    request<MaintenanceWindow>(`/api/v1/targets/${id}/maintenance`, {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  deleteMaintenanceWindow: (targetID: string, windowID: string) =>
    request<void>(`/api/v1/targets/${targetID}/maintenance/${windowID}`, {
      method: "DELETE",
    }),
  notificationChannels: (id: string) =>
    request<NotificationChannel[]>(`/api/v1/targets/${id}/notifications`),
  createNotificationChannel: (
    id: string,
    payload: { kind: "email" | "webhook"; destination: string },
  ) =>
    request<NotificationChannel>(`/api/v1/targets/${id}/notifications`, {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  deleteNotificationChannel: (targetID: string, channelID: string) =>
    request<void>(`/api/v1/targets/${targetID}/notifications/${channelID}`, {
      method: "DELETE",
    }),
  monitoringOverview: () =>
    request<MonitoringOverview>("/api/v1/monitoring?limit=100"),
};

function workspaceQuery() {
  const workspaceID = window.localStorage.getItem("netscope.workspace");
  if (!workspaceID) return "";
  return new URLSearchParams({ workspaceId: workspaceID }).toString();
}
