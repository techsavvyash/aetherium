"use client";

import React, { useEffect, useState, useRef } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Header } from "@/components/header";
import { api } from "@/lib/api";
import type { VM, Execution } from "@/lib/types";
import { toast } from "sonner";
import {
  Terminal,
  Send,
  Trash2,
  RefreshCw,
  Server,
  Clock,
  CheckCircle,
  XCircle,
  Loader2,
  Plus,
} from "lucide-react";

interface CommandHistory {
  id: string;
  command: string;
  args: string;
  status: "pending" | "running" | "success" | "error";
  taskId?: string;
  result?: Execution;
  submittedAt: Date;
}

export default function ConsolePage() {
  const [vms, setVMs] = useState<VM[]>([]);
  const [selectedVmId, setSelectedVmId] = useState<string>("");
  const [command, setCommand] = useState("");
  const [args, setArgs] = useState("");
  const [executing, setExecuting] = useState(false);
  const [commandHistory, setCommandHistory] = useState<CommandHistory[]>([]);
  const [executions, setExecutions] = useState<Execution[]>([]);
  const [loading, setLoading] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);

  // Create VM state
  const [createOpen, setCreateOpen] = useState(false);
  const [createName, setCreateName] = useState("");
  const [createVcpus, setCreateVcpus] = useState("2");
  const [createMemory, setCreateMemory] = useState("512");
  const [createTools, setCreateTools] = useState("");
  const [creating, setCreating] = useState(false);

  const fetchVMs = async () => {
    try {
      const res = await api.listVMs();
      if (res.data) {
        const vmList = res.data.vms || [];
        setVMs(vmList);
        // Auto-select first running VM if none selected
        if (!selectedVmId && vmList.length > 0) {
          const runningVm = vmList.find(
            (vm) => vm.status.toLowerCase() === "running"
          );
          if (runningVm) {
            setSelectedVmId(runningVm.id);
          }
        }
      }
    } finally {
      setLoading(false);
    }
  };

  const fetchExecutions = async () => {
    if (!selectedVmId) return;
    try {
      const res = await api.getExecutions(selectedVmId);
      if (res.data) {
        setExecutions(res.data.executions || []);
      }
    } catch (error) {
      // Silently fail - executions endpoint might not exist
    }
  };

  useEffect(() => {
    fetchVMs();
  }, []);

  useEffect(() => {
    if (selectedVmId) {
      fetchExecutions();
      const interval = setInterval(fetchExecutions, 3000);
      return () => clearInterval(interval);
    }
  }, [selectedVmId]);

  // Scroll to bottom when new commands are added
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [commandHistory]);

  const handleExecute = async () => {
    if (!selectedVmId || !command.trim()) return;

    setExecuting(true);
    const historyId = crypto.randomUUID();
    const argsArray = args
      .split(" ")
      .map((a) => a.trim())
      .filter(Boolean);

    // Add to local history immediately
    const newEntry: CommandHistory = {
      id: historyId,
      command: command.trim(),
      args: args,
      status: "pending",
      submittedAt: new Date(),
    };
    setCommandHistory((prev) => [...prev, newEntry]);

    try {
      const res = await api.executeCommand(selectedVmId, {
        command: command.trim(),
        args: argsArray.length > 0 ? argsArray : undefined,
      });

      if (res.data?.task_id) {
        setCommandHistory((prev) =>
          prev.map((h) =>
            h.id === historyId
              ? { ...h, status: "running", taskId: res.data!.task_id }
              : h
          )
        );
        toast.success(`Command submitted: ${res.data.task_id}`);

        // Poll for result
        pollForResult(historyId, selectedVmId);
      } else if (res.error) {
        setCommandHistory((prev) =>
          prev.map((h) => (h.id === historyId ? { ...h, status: "error" } : h))
        );
        toast.error("Failed to execute: " + res.error);
      }
    } catch (error) {
      setCommandHistory((prev) =>
        prev.map((h) => (h.id === historyId ? { ...h, status: "error" } : h))
      );
      toast.error("Failed to execute command");
    } finally {
      setExecuting(false);
      setCommand("");
      setArgs("");
    }
  };

  const pollForResult = async (historyId: string, vmId: string) => {
    // Poll for execution results
    let attempts = 0;
    const maxAttempts = 30;

    const poll = async () => {
      attempts++;
      try {
        const res = await api.getExecutions(vmId);
        if (res.data?.executions) {
          setExecutions(res.data.executions);

          // Find the matching execution by checking recent ones
          const historyEntry = commandHistory.find((h) => h.id === historyId);
          if (historyEntry) {
            const matchingExec = res.data.executions.find(
              (e) =>
                e.command === historyEntry.command &&
                new Date(e.started_at) >= historyEntry.submittedAt
            );

            if (matchingExec && matchingExec.completed_at) {
              setCommandHistory((prev) =>
                prev.map((h) =>
                  h.id === historyId
                    ? {
                        ...h,
                        status: matchingExec.exit_code === 0 ? "success" : "error",
                        result: matchingExec,
                      }
                    : h
                )
              );
              return; // Stop polling
            }
          }
        }
      } catch (error) {
        // Continue polling
      }

      if (attempts < maxAttempts) {
        setTimeout(poll, 2000);
      } else {
        // Timeout - mark as unknown
        setCommandHistory((prev) =>
          prev.map((h) =>
            h.id === historyId && h.status === "running"
              ? { ...h, status: "success" }
              : h
          )
        );
      }
    };

    setTimeout(poll, 2000);
  };

  const clearHistory = () => {
    setCommandHistory([]);
  };

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
        toast.success(`VM creation started: ${res.data.task_id}`);
        setCreateOpen(false);
        setCreateName("");
        setCreateVcpus("2");
        setCreateMemory("512");
        setCreateTools("");
        // Poll for the new VM to appear
        const pollForNewVM = () => {
          setTimeout(async () => {
            await fetchVMs();
            // Check if new VM appeared
            const newVMs = await api.listVMs();
            const newVm = newVMs.data?.vms?.find(
              (vm) => vm.name === createName && vm.status.toLowerCase() === "running"
            );
            if (newVm) {
              setSelectedVmId(newVm.id);
              toast.success(`VM "${createName}" is now ready!`);
            } else {
              pollForNewVM(); // Keep polling
            }
          }, 5000);
        };
        pollForNewVM();
      } else if (res.error) {
        toast.error("Failed to create VM: " + res.error);
      }
    } finally {
      setCreating(false);
    }
  };

  const selectedVm = vms.find((vm) => vm.id === selectedVmId);
  const runningVMs = vms.filter((vm) => vm.status.toLowerCase() === "running");

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleExecute();
    }
  };

  return (
    <div className="flex flex-col h-full">
      <Header onRefresh={fetchVMs} />
      <div className="flex-1 p-6 flex flex-col gap-6 overflow-hidden">
        {/* VM Selector */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-lg flex items-center gap-2">
              <Terminal className="h-5 w-5" />
              VM Console
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-4">
              <div className="flex-1">
                <Select value={selectedVmId} onValueChange={setSelectedVmId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select a VM..." />
                  </SelectTrigger>
                  <SelectContent>
                    {runningVMs.length === 0 ? (
                      <SelectItem value="none" disabled>
                        No running VMs available
                      </SelectItem>
                    ) : (
                      runningVMs.map((vm) => (
                        <SelectItem key={vm.id} value={vm.id}>
                          <div className="flex items-center gap-2">
                            <Server className="h-4 w-4" />
                            <span>{vm.name}</span>
                            <Badge variant="outline" className="ml-2">
                              {vm.vcpu_count} vCPU, {vm.memory_mb} MB
                            </Badge>
                          </div>
                        </SelectItem>
                      ))
                    )}
                  </SelectContent>
                </Select>
              </div>
              <Dialog open={createOpen} onOpenChange={setCreateOpen}>
                <DialogTrigger asChild>
                  <Button variant="outline">
                    <Plus className="h-4 w-4 mr-2" />
                    New VM
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Create Virtual Machine</DialogTitle>
                    <DialogDescription>
                      Create a new Firecracker microVM to execute commands.
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
                      {creating ? (
                        <>
                          <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                          Creating...
                        </>
                      ) : (
                        "Create"
                      )}
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
              {selectedVm && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Badge
                    variant={
                      selectedVm.status.toLowerCase() === "running"
                        ? "default"
                        : "secondary"
                    }
                  >
                    {selectedVm.status}
                  </Badge>
                  <span>ID: {selectedVm.id.slice(0, 8)}...</span>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Terminal Area */}
        <Card className="flex-1 flex flex-col overflow-hidden">
          <CardHeader className="pb-3 flex-shrink-0">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm">Command Output</CardTitle>
              <div className="flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={fetchExecutions}
                  disabled={!selectedVmId}
                >
                  <RefreshCw className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={clearHistory}
                  disabled={commandHistory.length === 0}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent className="flex-1 p-0 overflow-hidden">
            <ScrollArea className="h-full" ref={scrollRef}>
              <div className="bg-black text-green-400 font-mono text-sm p-4 min-h-full">
                {commandHistory.length === 0 && executions.length === 0 ? (
                  <div className="text-gray-500">
                    {selectedVmId
                      ? "No commands executed yet. Enter a command below."
                      : "Select a VM to start executing commands."}
                  </div>
                ) : (
                  <div className="space-y-4">
                    {/* Show execution history from API */}
                    {executions.length > 0 && commandHistory.length === 0 && (
                      <>
                        <div className="text-gray-500 mb-4">
                          --- Previous Executions ---
                        </div>
                        {executions.slice(-10).map((exec) => (
                          <div key={exec.id} className="border-b border-gray-800 pb-4">
                            <div className="flex items-center gap-2 mb-2">
                              <span className="text-blue-400">$</span>
                              <span className="text-white">
                                {exec.command} {exec.args?.join(" ")}
                              </span>
                              <Badge
                                variant={
                                  exec.exit_code === 0 ? "default" : "destructive"
                                }
                                className="ml-auto text-xs"
                              >
                                exit: {exec.exit_code ?? "?"}
                              </Badge>
                            </div>
                            {exec.stdout && (
                              <pre className="text-green-400 whitespace-pre-wrap ml-4">
                                {exec.stdout}
                              </pre>
                            )}
                            {exec.stderr && (
                              <pre className="text-red-400 whitespace-pre-wrap ml-4">
                                {exec.stderr}
                              </pre>
                            )}
                            <div className="text-gray-600 text-xs mt-1 ml-4">
                              {exec.duration_ms && `${exec.duration_ms}ms`}
                              {exec.completed_at &&
                                ` - ${new Date(exec.completed_at).toLocaleString()}`}
                            </div>
                          </div>
                        ))}
                      </>
                    )}

                    {/* Show current session commands */}
                    {commandHistory.map((entry) => (
                      <div key={entry.id} className="border-b border-gray-800 pb-4">
                        <div className="flex items-center gap-2 mb-2">
                          <span className="text-blue-400">$</span>
                          <span className="text-white">
                            {entry.command} {entry.args}
                          </span>
                          <div className="ml-auto flex items-center gap-2">
                            {entry.status === "pending" && (
                              <Loader2 className="h-4 w-4 animate-spin text-yellow-400" />
                            )}
                            {entry.status === "running" && (
                              <Loader2 className="h-4 w-4 animate-spin text-blue-400" />
                            )}
                            {entry.status === "success" && (
                              <CheckCircle className="h-4 w-4 text-green-400" />
                            )}
                            {entry.status === "error" && (
                              <XCircle className="h-4 w-4 text-red-400" />
                            )}
                            <span className="text-xs text-gray-600">
                              {entry.submittedAt.toLocaleTimeString()}
                            </span>
                          </div>
                        </div>
                        {entry.result && (
                          <>
                            {entry.result.stdout && (
                              <pre className="text-green-400 whitespace-pre-wrap ml-4">
                                {entry.result.stdout}
                              </pre>
                            )}
                            {entry.result.stderr && (
                              <pre className="text-red-400 whitespace-pre-wrap ml-4">
                                {entry.result.stderr}
                              </pre>
                            )}
                            <div className="text-gray-600 text-xs mt-1 ml-4">
                              Exit code: {entry.result.exit_code}
                              {entry.result.duration_ms &&
                                ` | Duration: ${entry.result.duration_ms}ms`}
                            </div>
                          </>
                        )}
                        {entry.status === "running" && (
                          <div className="text-yellow-400 ml-4 animate-pulse">
                            Executing...
                          </div>
                        )}
                        {entry.status === "pending" && (
                          <div className="text-gray-500 ml-4">Submitting...</div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </ScrollArea>
          </CardContent>
        </Card>

        {/* Command Input */}
        <Card className="flex-shrink-0">
          <CardContent className="p-4">
            <div className="flex gap-2">
              <div className="flex-1 flex gap-2">
                <div className="flex-1">
                  <Input
                    value={command}
                    onChange={(e) => setCommand(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder="Command (e.g., ls, cat, node)"
                    className="font-mono"
                    disabled={!selectedVmId || executing}
                  />
                </div>
                <div className="flex-1">
                  <Input
                    value={args}
                    onChange={(e) => setArgs(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder="Arguments (e.g., -la /home)"
                    className="font-mono"
                    disabled={!selectedVmId || executing}
                  />
                </div>
              </div>
              <Button
                onClick={handleExecute}
                disabled={!selectedVmId || !command.trim() || executing}
              >
                {executing ? (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                ) : (
                  <Send className="h-4 w-4 mr-2" />
                )}
                Execute
              </Button>
            </div>
            <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
              <span>Press Enter to execute</span>
              <span>|</span>
              <span>Commands run via vsock agent inside the VM</span>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
