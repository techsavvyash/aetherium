import type {
  VM,
  Task,
  Execution,
  Worker,
  QueueStats,
  CreateVMRequest,
  ExecuteCommandRequest,
  ApiResponse,
  Workspace,
  PromptTask,
  Secret,
  CreateWorkspaceRequest,
  CreateWorkspaceResponse,
  SubmitPromptRequest,
  SubmitPromptResponse,
  AddSecretRequest,
  AddSecretResponse,
  Environment,
  CreateEnvironmentRequest,
  UpdateEnvironmentRequest,
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

  // Workspaces
  listWorkspaces: () =>
    fetchApi<{ workspaces: Workspace[]; total: number }>("/api/v1/workspaces"),

  getWorkspace: (id: string) =>
    fetchApi<Workspace>(`/api/v1/workspaces/${id}`),

  createWorkspace: (req: CreateWorkspaceRequest) =>
    fetchApi<CreateWorkspaceResponse>("/api/v1/workspaces", {
      method: "POST",
      body: JSON.stringify(req),
    }),

  deleteWorkspace: (id: string) =>
    fetchApi<{ id: string; type: string; status: string }>(`/api/v1/workspaces/${id}`, {
      method: "DELETE",
    }),

  // Prompts
  submitPrompt: (workspaceId: string, req: SubmitPromptRequest) =>
    fetchApi<SubmitPromptResponse>(`/api/v1/workspaces/${workspaceId}/prompts`, {
      method: "POST",
      body: JSON.stringify(req),
    }),

  listPrompts: (workspaceId: string) =>
    fetchApi<{ prompts: PromptTask[]; total: number }>(`/api/v1/workspaces/${workspaceId}/prompts`),

  getPrompt: (workspaceId: string, promptId: string) =>
    fetchApi<PromptTask>(`/api/v1/workspaces/${workspaceId}/prompts/${promptId}`),

  // Secrets
  addSecret: (workspaceId: string, req: AddSecretRequest) =>
    fetchApi<AddSecretResponse>(`/api/v1/workspaces/${workspaceId}/secrets`, {
      method: "POST",
      body: JSON.stringify(req),
    }),

  listSecrets: (workspaceId: string) =>
    fetchApi<{ secrets: Secret[]; total: number }>(`/api/v1/workspaces/${workspaceId}/secrets`),

  deleteSecret: (workspaceId: string, secretId: string) =>
    fetchApi<{ status: string }>(`/api/v1/workspaces/${workspaceId}/secrets/${secretId}`, {
      method: "DELETE",
    }),

  // Environments
  listEnvironments: () =>
    fetchApi<{ environments: Environment[]; total: number }>("/api/v1/environments"),

  getEnvironment: (id: string) =>
    fetchApi<Environment>(`/api/v1/environments/${id}`),

  createEnvironment: (req: CreateEnvironmentRequest) =>
    fetchApi<Environment>("/api/v1/environments", {
      method: "POST",
      body: JSON.stringify(req),
    }),

  updateEnvironment: (id: string, req: UpdateEnvironmentRequest) =>
    fetchApi<Environment>(`/api/v1/environments/${id}`, {
      method: "PUT",
      body: JSON.stringify(req),
    }),

  deleteEnvironment: (id: string) =>
    fetchApi<{ status: string }>(`/api/v1/environments/${id}`, {
      method: "DELETE",
    }),

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
