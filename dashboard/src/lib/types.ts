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
