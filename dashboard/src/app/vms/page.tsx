"use client";

import React, { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Header } from "@/components/header";
import { api } from "@/lib/api";
import type { VM, Execution } from "@/lib/types";
import { toast } from "sonner";
import {
  Server,
  Plus,
  Trash2,
  Terminal,
  ChevronDown,
  ChevronRight,
  Play,
} from "lucide-react";

export default function VMsPage() {
  const [vms, setVMs] = useState<VM[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedVM, setExpandedVM] = useState<string | null>(null);
  const [executions, setExecutions] = useState<Record<string, Execution[]>>({});

  // Create VM state
  const [createOpen, setCreateOpen] = useState(false);
  const [createName, setCreateName] = useState("");
  const [createVcpus, setCreateVcpus] = useState("2");
  const [createMemory, setCreateMemory] = useState("512");
  const [createTools, setCreateTools] = useState("");
  const [creating, setCreating] = useState(false);

  // Execute command state
  const [executeOpen, setExecuteOpen] = useState(false);
  const [executeVmId, setExecuteVmId] = useState("");
  const [executeCommand, setExecuteCommand] = useState("");
  const [executeArgs, setExecuteArgs] = useState("");
  const [executing, setExecuting] = useState(false);

  const fetchVMs = async () => {
    setLoading(true);
    try {
      const res = await api.listVMs();
      if (res.data) {
        setVMs(res.data.vms || []);
      } else if (res.error) {
        toast.error("Failed to fetch VMs: " + res.error);
      }
    } finally {
      setLoading(false);
    }
  };

  const fetchExecutions = async (vmId: string) => {
    const res = await api.getExecutions(vmId);
    if (res.data) {
      setExecutions((prev) => ({ ...prev, [vmId]: res.data!.executions || [] }));
    }
  };

  useEffect(() => {
    fetchVMs();
    const interval = setInterval(fetchVMs, 15000);
    return () => clearInterval(interval);
  }, []);

  const handleCreateVM = async () => {
    setCreating(true);
    try {
      const tools = createTools
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean);

      const res = await api.createVM({
        name: createName,
        vcpus: parseInt(createVcpus),
        memory_mb: parseInt(createMemory),
        additional_tools: tools.length > 0 ? tools : undefined,
      });

      if (res.data) {
        toast.success(`VM creation task submitted: ${res.data.task_id}`);
        setCreateOpen(false);
        setCreateName("");
        setCreateVcpus("2");
        setCreateMemory("512");
        setCreateTools("");
        setTimeout(fetchVMs, 2000);
      } else if (res.error) {
        toast.error("Failed to create VM: " + res.error);
      }
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteVM = async (vmId: string) => {
    const res = await api.deleteVM(vmId);
    if (res.data) {
      toast.success(`VM deletion task submitted: ${res.data.task_id}`);
      setTimeout(fetchVMs, 2000);
    } else if (res.error) {
      toast.error("Failed to delete VM: " + res.error);
    }
  };

  const handleExecuteCommand = async () => {
    setExecuting(true);
    try {
      const args = executeArgs
        .split(" ")
        .map((a) => a.trim())
        .filter(Boolean);

      const res = await api.executeCommand(executeVmId, {
        command: executeCommand,
        args: args.length > 0 ? args : undefined,
      });

      if (res.data) {
        toast.success(`Command execution task submitted: ${res.data.task_id}`);
        setExecuteOpen(false);
        setExecuteCommand("");
        setExecuteArgs("");
        setTimeout(() => fetchExecutions(executeVmId), 3000);
      } else if (res.error) {
        toast.error("Failed to execute command: " + res.error);
      }
    } finally {
      setExecuting(false);
    }
  };

  const toggleExpand = (vmId: string) => {
    if (expandedVM === vmId) {
      setExpandedVM(null);
    } else {
      setExpandedVM(vmId);
      if (!executions[vmId]) {
        fetchExecutions(vmId);
      }
    }
  };

  const openExecuteDialog = (vmId: string) => {
    setExecuteVmId(vmId);
    setExecuteOpen(true);
  };

  return (
    <div className="flex flex-col h-full">
      <Header onRefresh={fetchVMs} />
      <div className="flex-1 p-6 space-y-6">
        {/* Actions */}
        <div className="flex justify-between items-center">
          <div>
            <h2 className="text-lg font-semibold">
              {vms.length} Virtual Machine{vms.length !== 1 ? "s" : ""}
            </h2>
          </div>
          <Dialog open={createOpen} onOpenChange={setCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                Create VM
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create Virtual Machine</DialogTitle>
                <DialogDescription>
                  Create a new Firecracker microVM with the specified configuration.
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <label htmlFor="name" className="text-sm font-medium">
                    Name
                  </label>
                  <Input
                    id="name"
                    value={createName}
                    onChange={(e) => setCreateName(e.target.value)}
                    placeholder="my-vm"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="grid gap-2">
                    <label htmlFor="vcpus" className="text-sm font-medium">
                      vCPUs
                    </label>
                    <Input
                      id="vcpus"
                      type="number"
                      value={createVcpus}
                      onChange={(e) => setCreateVcpus(e.target.value)}
                      min="1"
                      max="8"
                    />
                  </div>
                  <div className="grid gap-2">
                    <label htmlFor="memory" className="text-sm font-medium">
                      Memory (MB)
                    </label>
                    <Input
                      id="memory"
                      type="number"
                      value={createMemory}
                      onChange={(e) => setCreateMemory(e.target.value)}
                      min="128"
                      max="8192"
                    />
                  </div>
                </div>
                <div className="grid gap-2">
                  <label htmlFor="tools" className="text-sm font-medium">
                    Additional Tools (comma-separated)
                  </label>
                  <Input
                    id="tools"
                    value={createTools}
                    onChange={(e) => setCreateTools(e.target.value)}
                    placeholder="go, python, rust"
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setCreateOpen(false)}>
                  Cancel
                </Button>
                <Button onClick={handleCreateVM} disabled={creating || !createName}>
                  {creating ? "Creating..." : "Create"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>

        {/* Execute Command Dialog */}
        <Dialog open={executeOpen} onOpenChange={setExecuteOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Execute Command</DialogTitle>
              <DialogDescription>
                Run a command inside the virtual machine.
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <label htmlFor="command" className="text-sm font-medium">
                  Command
                </label>
                <Input
                  id="command"
                  value={executeCommand}
                  onChange={(e) => setExecuteCommand(e.target.value)}
                  placeholder="ls"
                />
              </div>
              <div className="grid gap-2">
                <label htmlFor="args" className="text-sm font-medium">
                  Arguments (space-separated)
                </label>
                <Input
                  id="args"
                  value={executeArgs}
                  onChange={(e) => setExecuteArgs(e.target.value)}
                  placeholder="-la /home"
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setExecuteOpen(false)}>
                Cancel
              </Button>
              <Button
                onClick={handleExecuteCommand}
                disabled={executing || !executeCommand}
              >
                {executing ? "Executing..." : "Execute"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* VM List */}
        <Card>
          <CardContent className="p-0">
            {loading && vms.length === 0 ? (
              <div className="p-6 text-center text-muted-foreground">
                Loading virtual machines...
              </div>
            ) : vms.length === 0 ? (
              <div className="p-6 text-center text-muted-foreground">
                No virtual machines found. Create one to get started.
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-8"></TableHead>
                    <TableHead>Name</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Resources</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {vms.map((vm) => (
                    <React.Fragment key={vm.id}>
                      <TableRow>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
                            onClick={() => toggleExpand(vm.id)}
                          >
                            {expandedVM === vm.id ? (
                              <ChevronDown className="h-4 w-4" />
                            ) : (
                              <ChevronRight className="h-4 w-4" />
                            )}
                          </Button>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <Server className="h-4 w-4 text-muted-foreground" />
                            <span className="font-medium">{vm.name}</span>
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={
                              vm.status.toLowerCase() === "running"
                                ? "default"
                                : vm.status.toLowerCase() === "stopped"
                                ? "secondary"
                                : "outline"
                            }
                          >
                            {vm.status}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          {vm.vcpu_count} vCPU, {vm.memory_mb} MB
                        </TableCell>
                        <TableCell>
                          {new Date(vm.created_at).toLocaleString()}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-2">
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => openExecuteDialog(vm.id)}
                              disabled={vm.status.toLowerCase() !== "running"}
                            >
                              <Terminal className="h-4 w-4 mr-1" />
                              Execute
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleDeleteVM(vm.id)}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                      {expandedVM === vm.id && (
                        <TableRow>
                          <TableCell colSpan={6} className="bg-muted/50">
                            <div className="p-4 space-y-4">
                              <div className="grid grid-cols-2 gap-4 text-sm">
                                <div>
                                  <span className="text-muted-foreground">ID:</span>{" "}
                                  <code className="text-xs bg-muted px-1 rounded">
                                    {vm.id}
                                  </code>
                                </div>
                                <div>
                                  <span className="text-muted-foreground">
                                    Orchestrator:
                                  </span>{" "}
                                  {vm.orchestrator}
                                </div>
                                {vm.kernel_path && (
                                  <div>
                                    <span className="text-muted-foreground">
                                      Kernel:
                                    </span>{" "}
                                    {vm.kernel_path}
                                  </div>
                                )}
                                {vm.rootfs_path && (
                                  <div>
                                    <span className="text-muted-foreground">
                                      Rootfs:
                                    </span>{" "}
                                    {vm.rootfs_path}
                                  </div>
                                )}
                              </div>

                              <div>
                                <h4 className="font-medium mb-2">
                                  Recent Executions
                                </h4>
                                {!executions[vm.id] ? (
                                  <p className="text-sm text-muted-foreground">
                                    Loading executions...
                                  </p>
                                ) : executions[vm.id].length === 0 ? (
                                  <p className="text-sm text-muted-foreground">
                                    No executions found
                                  </p>
                                ) : (
                                  <div className="space-y-2">
                                    {executions[vm.id].slice(0, 5).map((exec) => (
                                      <div
                                        key={exec.id}
                                        className="border rounded-lg p-3 text-sm"
                                      >
                                        <div className="flex items-center justify-between mb-2">
                                          <code className="font-mono">
                                            {exec.command}{" "}
                                            {exec.args?.join(" ")}
                                          </code>
                                          <Badge
                                            variant={
                                              exec.exit_code === 0
                                                ? "default"
                                                : "destructive"
                                            }
                                          >
                                            Exit: {exec.exit_code ?? "pending"}
                                          </Badge>
                                        </div>
                                        {exec.stdout && (
                                          <pre className="bg-black text-green-400 p-2 rounded text-xs overflow-x-auto max-h-32">
                                            {exec.stdout}
                                          </pre>
                                        )}
                                        {exec.stderr && (
                                          <pre className="bg-black text-red-400 p-2 rounded text-xs overflow-x-auto max-h-32 mt-1">
                                            {exec.stderr}
                                          </pre>
                                        )}
                                      </div>
                                    ))}
                                  </div>
                                )}
                              </div>
                            </div>
                          </TableCell>
                        </TableRow>
                      )}
                    </React.Fragment>
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
