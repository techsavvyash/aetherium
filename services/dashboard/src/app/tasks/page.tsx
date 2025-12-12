"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Header } from "@/components/header";
import { api } from "@/lib/api";
import type { Task } from "@/lib/types";
import { ListTodo, ChevronDown, ChevronRight } from "lucide-react";

const taskTypeLabels: Record<string, string> = {
  "vm:create": "Create VM",
  "vm:execute": "Execute Command",
  "vm:delete": "Delete VM",
};

const statusColors: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  pending: "secondary",
  active: "default",
  completed: "default",
  failed: "destructive",
  retry: "outline",
};

export default function TasksPage() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedTask, setExpandedTask] = useState<string | null>(null);
  const [filterStatus, setFilterStatus] = useState<string>("all");
  const [filterType, setFilterType] = useState<string>("all");

  const fetchTasks = async () => {
    setLoading(true);
    try {
      const res = await api.listTasks();
      if (res.data) {
        setTasks(res.data.tasks || []);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTasks();
    const interval = setInterval(fetchTasks, 10000);
    return () => clearInterval(interval);
  }, []);

  const filteredTasks = tasks.filter((task) => {
    if (filterStatus !== "all" && task.status !== filterStatus) return false;
    if (filterType !== "all" && task.type !== filterType) return false;
    return true;
  });

  const taskStats = {
    pending: tasks.filter((t) => t.status === "pending").length,
    active: tasks.filter((t) => t.status === "active").length,
    completed: tasks.filter((t) => t.status === "completed").length,
    failed: tasks.filter((t) => t.status === "failed").length,
  };

  return (
    <div className="flex flex-col h-full">
      <Header onRefresh={fetchTasks} />
      <div className="flex-1 p-6 space-y-6">
        {/* Stats */}
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold">{taskStats.pending}</div>
              <p className="text-xs text-muted-foreground">Pending</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold">{taskStats.active}</div>
              <p className="text-xs text-muted-foreground">Active</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold text-green-600">
                {taskStats.completed}
              </div>
              <p className="text-xs text-muted-foreground">Completed</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold text-red-600">
                {taskStats.failed}
              </div>
              <p className="text-xs text-muted-foreground">Failed</p>
            </CardContent>
          </Card>
        </div>

        {/* Filters */}
        <div className="flex gap-4">
          <div className="w-48">
            <Select value={filterStatus} onValueChange={setFilterStatus}>
              <SelectTrigger>
                <SelectValue placeholder="Filter by status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Statuses</SelectItem>
                <SelectItem value="pending">Pending</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="completed">Completed</SelectItem>
                <SelectItem value="failed">Failed</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="w-48">
            <Select value={filterType} onValueChange={setFilterType}>
              <SelectTrigger>
                <SelectValue placeholder="Filter by type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Types</SelectItem>
                <SelectItem value="vm:create">Create VM</SelectItem>
                <SelectItem value="vm:execute">Execute Command</SelectItem>
                <SelectItem value="vm:delete">Delete VM</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        {/* Tasks Table */}
        <Card>
          <CardHeader>
            <CardTitle>
              Tasks ({filteredTasks.length} of {tasks.length})
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {loading && tasks.length === 0 ? (
              <div className="p-6 text-center text-muted-foreground">
                Loading tasks...
              </div>
            ) : filteredTasks.length === 0 ? (
              <div className="p-6 text-center text-muted-foreground">
                No tasks found matching the filters.
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-8"></TableHead>
                    <TableHead>Task ID</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Priority</TableHead>
                    <TableHead>Retries</TableHead>
                    <TableHead>Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredTasks.map((task) => (
                    <>
                      <TableRow key={task.id}>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
                            onClick={() =>
                              setExpandedTask(
                                expandedTask === task.id ? null : task.id
                              )
                            }
                          >
                            {expandedTask === task.id ? (
                              <ChevronDown className="h-4 w-4" />
                            ) : (
                              <ChevronRight className="h-4 w-4" />
                            )}
                          </Button>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <ListTodo className="h-4 w-4 text-muted-foreground" />
                            <code className="text-xs bg-muted px-1 rounded">
                              {task.id.slice(0, 8)}...
                            </code>
                          </div>
                        </TableCell>
                        <TableCell>
                          {taskTypeLabels[task.type] || task.type}
                        </TableCell>
                        <TableCell>
                          <Badge variant={statusColors[task.status] || "outline"}>
                            {task.status}
                          </Badge>
                        </TableCell>
                        <TableCell>{task.priority}</TableCell>
                        <TableCell>
                          {task.retry_count}/{task.max_retries}
                        </TableCell>
                        <TableCell>
                          {new Date(task.created_at).toLocaleString()}
                        </TableCell>
                      </TableRow>
                      {expandedTask === task.id && (
                        <TableRow>
                          <TableCell colSpan={7} className="bg-muted/50">
                            <div className="p-4 space-y-4">
                              <div className="grid grid-cols-2 gap-4 text-sm">
                                <div>
                                  <span className="text-muted-foreground">
                                    Full ID:
                                  </span>{" "}
                                  <code className="text-xs bg-muted px-1 rounded">
                                    {task.id}
                                  </code>
                                </div>
                                {task.vm_id && (
                                  <div>
                                    <span className="text-muted-foreground">
                                      VM ID:
                                    </span>{" "}
                                    <code className="text-xs bg-muted px-1 rounded">
                                      {task.vm_id}
                                    </code>
                                  </div>
                                )}
                                {task.worker_id && (
                                  <div>
                                    <span className="text-muted-foreground">
                                      Worker:
                                    </span>{" "}
                                    {task.worker_id}
                                  </div>
                                )}
                                {task.started_at && (
                                  <div>
                                    <span className="text-muted-foreground">
                                      Started:
                                    </span>{" "}
                                    {new Date(task.started_at).toLocaleString()}
                                  </div>
                                )}
                                {task.completed_at && (
                                  <div>
                                    <span className="text-muted-foreground">
                                      Completed:
                                    </span>{" "}
                                    {new Date(task.completed_at).toLocaleString()}
                                  </div>
                                )}
                              </div>

                              <div>
                                <h4 className="font-medium mb-2">Payload</h4>
                                <pre className="bg-black text-green-400 p-3 rounded text-xs overflow-x-auto">
                                  {JSON.stringify(task.payload, null, 2)}
                                </pre>
                              </div>

                              {task.result && (
                                <div>
                                  <h4 className="font-medium mb-2">Result</h4>
                                  <pre className="bg-black text-blue-400 p-3 rounded text-xs overflow-x-auto">
                                    {JSON.stringify(task.result, null, 2)}
                                  </pre>
                                </div>
                              )}

                              {task.error && (
                                <div>
                                  <h4 className="font-medium mb-2 text-red-600">
                                    Error
                                  </h4>
                                  <pre className="bg-black text-red-400 p-3 rounded text-xs overflow-x-auto">
                                    {task.error}
                                  </pre>
                                </div>
                              )}
                            </div>
                          </TableCell>
                        </TableRow>
                      )}
                    </>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
