export interface VM {
  id: string;
  name: string;
  orchestrator: string;
  status: string;
  kernel_path?: string;
  rootfs_path?: string;
  socket_path?: string;
  vcpu_count: number;
  memory_mb: number;
  created_at: string;
  started_at?: string;
  stopped_at?: string;
  metadata?: Record<string, unknown>;
}

export interface Task {
  id: string;
  type: string;
  status: string;
  priority: number;
  payload: Record<string, unknown>;
  result?: Record<string, unknown>;
  error?: string;
  vm_id?: string;
  worker_id?: string;
  max_retries: number;
  retry_count: number;
  created_at: string;
  scheduled_at: string;
  started_at?: string;
  completed_at?: string;
  metadata?: Record<string, unknown>;
}

export interface Execution {
  id: string;
  job_id?: string;
  vm_id: string;
  command: string;
  args?: string[];
  env?: Record<string, string>;
  exit_code?: number;
  stdout?: string;
  stderr?: string;
  error?: string;
  started_at: string;
  completed_at?: string;
  duration_ms?: number;
  metadata?: Record<string, unknown>;
}

export interface Worker {
  id: string;
  hostname: string;
  address: string;
  zone: string;
  status: string;
  labels: Record<string, string>;
  capabilities: string[];
  cpu_cores: number;
  memory_mb: number;
  disk_gb: number;
  used_cpu_cores: number;
  used_memory_mb: number;
  used_disk_gb: number;
  cpu_usage_percent: number;
  memory_usage_percent: number;
  vm_count: number;
  max_vms: number;
  started_at: string;
  last_seen: string;
  uptime: string;
  is_healthy: boolean;
}

export interface QueueStats {
  pending: number;
  active: number;
  scheduled: number;
  retry: number;
  archived: number;
  completed: number;
  failed: number;
}

export interface CreateVMRequest {
  name: string;
  vcpus: number;
  memory_mb: number;
  additional_tools?: string[];
  tool_versions?: Record<string, string>;
}

export interface ExecuteCommandRequest {
  command: string;
  args?: string[];
  env?: Record<string, string>;
  timeout?: string;
}

export interface ApiResponse<T> {
  data?: T;
  error?: string;
  task_id?: string;
  status?: string;
}

// Workspace types

export interface Workspace {
  id: string;
  name: string;
  description?: string;
  vm_id?: string;
  vm_name?: string;
  status: string;
  ai_assistant: string;
  ai_assistant_config?: Record<string, unknown>;
  working_directory: string;
  created_at: string;
  ready_at?: string;
  stopped_at?: string;
  metadata?: Record<string, unknown>;
  prep_steps?: PrepStep[];
  secrets?: Secret[];
}

export interface PrepStep {
  id: string;
  workspace_id: string;
  step_type: 'git_clone' | 'script' | 'env_var';
  step_order: number;
  config: Record<string, unknown>;
  status: string;
  exit_code?: number;
  stdout?: string;
  stderr?: string;
  started_at?: string;
  completed_at?: string;
}

export interface Secret {
  id: string;
  name: string;
  type: string;
  description?: string;
  scope: string;
  created_at: string;
  updated_at: string;
}

export interface PromptTask {
  id: string;
  workspace_id: string;
  prompt: string;
  system_prompt?: string;
  working_directory?: string;
  environment?: Record<string, unknown>;
  priority: number;
  status: string;
  exit_code?: number;
  stdout?: string;
  stderr?: string;
  error?: string;
  created_at: string;
  scheduled_at: string;
  started_at?: string;
  completed_at?: string;
  duration_ms?: number;
  metadata?: Record<string, unknown>;
}

export interface SessionMessage {
  id: string;
  session_id: string;
  message_type: string;
  content: string;
  exit_code?: number;
  created_at: string;
}

// Workspace request types

export interface CreateWorkspaceRequest {
  name: string;
  description?: string;
  vcpus: number;
  memory_mb: number;
  ai_assistant: 'claude-code' | 'ampcode';
  ai_assistant_config?: Record<string, unknown>;
  working_directory?: string;
  secrets?: SecretRequest[];
  prep_steps?: PrepStepRequest[];
  additional_tools?: string[];
  tool_versions?: Record<string, string>;
}

export interface SecretRequest {
  name: string;
  value: string;
  type?: string;
  description?: string;
}

export interface PrepStepRequest {
  type: 'git_clone' | 'script' | 'env_var';
  order: number;
  config: GitCloneConfig | ScriptConfig | EnvVarConfig;
}

export interface GitCloneConfig {
  url: string;
  branch?: string;
  target_dir?: string;
  depth?: number;
}

export interface ScriptConfig {
  script: string;
  working_directory?: string;
}

export interface EnvVarConfig {
  name: string;
  value: string;
}

export interface AddSecretRequest {
  name: string;
  value: string;
  type?: string;
  description?: string;
  scope?: string;
}

export interface SubmitPromptRequest {
  prompt: string;
  system_prompt?: string;
  working_directory?: string;
  environment?: Record<string, unknown>;
  priority?: number;
}

// Workspace response types

export interface CreateWorkspaceResponse {
  task_id: string;
  workspace_id: string;
  status: string;
}

export interface SubmitPromptResponse {
  prompt_id: string;
  workspace_id: string;
  status: string;
}

export interface AddSecretResponse {
  secret_id: string;
  name: string;
  status: string;
}

// Environment types

export interface MCPServerConfig {
  name: string;
  type: 'stdio' | 'http';
  command?: string;
  args?: string[];
  url?: string;
  headers?: Record<string, string>;
  env?: Record<string, string>;
}

export interface Environment {
  id: string;
  name: string;
  description?: string;
  vcpus: number;
  memory_mb: number;
  git_repo_url?: string;
  git_branch: string;
  working_directory: string;
  tools: string[];
  env_vars?: Record<string, string>;
  mcp_servers?: MCPServerConfig[];
  idle_timeout_seconds: number;
  created_at: string;
  updated_at: string;
}

export interface CreateEnvironmentRequest {
  name: string;
  description?: string;
  vcpus?: number;
  memory_mb?: number;
  git_repo_url?: string;
  git_branch?: string;
  working_directory?: string;
  tools?: string[];
  env_vars?: Record<string, string>;
  mcp_servers?: MCPServerConfig[];
  idle_timeout_seconds?: number;
}

export interface UpdateEnvironmentRequest {
  name?: string;
  description?: string;
  vcpus?: number;
  memory_mb?: number;
  git_repo_url?: string;
  git_branch?: string;
  working_directory?: string;
  tools?: string[];
  env_vars?: Record<string, string>;
  mcp_servers?: MCPServerConfig[];
  idle_timeout_seconds?: number;
}

// MCP Server Presets for easy configuration
export const MCP_PRESETS: Record<string, MCPServerConfig> = {
  playwright: {
    name: 'playwright',
    type: 'stdio',
    command: 'npx',
    args: ['@playwright/mcp@latest'],
  },
  filesystem: {
    name: 'filesystem',
    type: 'stdio',
    command: 'npx',
    args: ['-y', '@anthropic/mcp-filesystem', '/workspace'],
  },
  git: {
    name: 'git',
    type: 'stdio',
    command: 'npx',
    args: ['-y', '@anthropic/mcp-git'],
  },
};
