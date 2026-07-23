import type {
  APIError,
  Capabilities,
  CheckType,
  DiagnosticRun,
  RunOptions,
  RunPage,
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
  exportURL: (id: string) => `${API_BASE}/api/v1/runs/${id}/export?format=json`,
};
