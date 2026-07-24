import type {
  APIError,
  Capabilities,
  CheckType,
  DiagnosticRun,
  MaintenanceWindow,
  MonitoredTarget,
  MonitoringCheckPage,
  MonitoringOverview,
  NotificationChannel,
  RunOptions,
  RunPage,
  TargetInput,
  TargetPage,
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
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      Accept: "application/json",
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

  eventURL: (id: string) => `${API_BASE}/api/v1/runs/${id}/events`,
  exportURL: (id: string, format: "json" | "csv" = "json") =>
    `${API_BASE}/api/v1/runs/${id}/export?format=${format}`,

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
