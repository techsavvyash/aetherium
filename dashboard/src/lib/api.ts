import type {
  VM,
  Task,
  Execution,
  Worker,
  QueueStats,
  CreateVMRequest,
  ExecuteCommandRequest,
  ApiResponse,
} from "./types";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function fetchApi<T>(
  endpoint: string,
  options?: RequestInit
): Promise<ApiResponse<T>> {
  try {
    const response = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...options?.headers,
      },
    });

    const data = await response.json();

    if (!response.ok) {
      return { error: data.error || `HTTP ${response.status}` };
    }

    return { data };
  } catch (error) {
    return { error: error instanceof Error ? error.message : "Unknown error" };
  }
}

export const api = {
  // VMs
  listVMs: () => fetchApi<{ vms: VM[]; total: number }>("/api/v1/vms"),

  getVM: (id: string) => fetchApi<VM>(`/api/v1/vms/${id}`),

  createVM: (req: CreateVMRequest) =>
    fetchApi<{ task_id: string; status: string }>("/api/v1/vms", {
      method: "POST",
      body: JSON.stringify(req),
    }),

  deleteVM: (id: string) =>
    fetchApi<{ task_id: string; status: string }>(`/api/v1/vms/${id}`, {
      method: "DELETE",
    }),

  executeCommand: (vmId: string, req: ExecuteCommandRequest) =>
    fetchApi<{ task_id: string; vm_id: string; status: string }>(
      `/api/v1/vms/${vmId}/execute`,
      {
        method: "POST",
        body: JSON.stringify(req),
      }
    ),

  getExecutions: (vmId: string) =>
    fetchApi<{ executions: Execution[] }>(`/api/v1/vms/${vmId}/executions`),

  // Tasks
  listTasks: () => fetchApi<{ tasks: Task[]; total: number }>("/api/v1/tasks"),

  getTask: (id: string) => fetchApi<Task>(`/api/v1/tasks/${id}`),

  // Workers
  listWorkers: () =>
    fetchApi<{ workers: Worker[] }>("/api/v1/workers"),

  getWorker: (id: string) => fetchApi<Worker>(`/api/v1/workers/${id}`),

  // Queue Stats
  getQueueStats: () => fetchApi<QueueStats>("/api/v1/queue/stats"),

  // Health
  health: () => fetchApi<{ status: string }>("/api/v1/health"),

  // Custom request
  customRequest: async (
    method: string,
    path: string,
    body?: string
  ): Promise<{ status: number; data: unknown; headers: Record<string, string> }> => {
    try {
      const response = await fetch(`${API_BASE}${path}`, {
        method,
        headers: {
          "Content-Type": "application/json",
        },
        body: body || undefined,
      });

      const responseHeaders: Record<string, string> = {};
      response.headers.forEach((value, key) => {
        responseHeaders[key] = value;
      });

      let data: unknown;
      const contentType = response.headers.get("content-type");
      if (contentType?.includes("application/json")) {
        data = await response.json();
      } else {
        data = await response.text();
      }

      return { status: response.status, data, headers: responseHeaders };
    } catch (error) {
      throw error;
    }
  },
};
