"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";
import { useWorkspaceStream } from "@/lib/websocket";
import type { Workspace, PromptTask, Secret, PrepStep } from "@/lib/types";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PromptInput } from "@/components/prompt-input";
import { PromptList } from "@/components/prompt-result";
import { TerminalView, TextTerminal } from "@/components/terminal-view";
import { Textarea } from "@/components/ui/textarea";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  ArrowLeft,
  Bot,
  Cpu,
  RefreshCw,
  Send,
  Key,
  GitBranch,
  CheckCircle2,
  XCircle,
  Clock,
  Loader2,
  ChevronDown,
  ChevronRight,
  Trash2,
  FolderCode,
  Plus,
  Terminal,
} from "lucide-react";

function getStatusBadge(status: string) {
  switch (status.toLowerCase()) {
    case "ready":
      return <Badge className="bg-green-500">Ready</Badge>;
    case "creating":
      return <Badge className="bg-blue-500">Creating</Badge>;
    case "preparing":
      return <Badge className="bg-yellow-500">Preparing</Badge>;
    case "failed":
      return <Badge variant="destructive">Failed</Badge>;
    case "stopped":
      return <Badge variant="secondary">Stopped</Badge>;
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
}

function getPrepStepStatusIcon(status: string) {
  switch (status.toLowerCase()) {
    case "completed":
      return <CheckCircle2 className="h-4 w-4 text-green-500" />;
    case "failed":
      return <XCircle className="h-4 w-4 text-red-500" />;
    case "running":
      return <Loader2 className="h-4 w-4 text-blue-500 animate-spin" />;
    default:
      return <Clock className="h-4 w-4 text-muted-foreground" />;
  }
}

function getAIAssistantInfo(assistant: string) {
  switch (assistant.toLowerCase()) {
    case "claude-code":
      return {
        name: "Claude Code",
        description: "Anthropic's AI coding assistant",
        icon: <Bot className="h-5 w-5" />,
      };
    case "ampcode":
    case "amp":
      return {
        name: "Ampcode",
        description: "AI-powered development tool",
        icon: <Bot className="h-5 w-5" />,
      };
    default:
      return {
        name: assistant,
        description: "AI assistant",
        icon: <Bot className="h-5 w-5" />,
      };
  }
}

export default function WorkspaceDetailPage() {
  const params = useParams();
  const router = useRouter();
  const workspaceId = params.id as string;

  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [prompts, setPrompts] = useState<PromptTask[]>([]);
  const [secrets, setSecrets] = useState<Secret[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Secret addition state
  const [showAddSecret, setShowAddSecret] = useState(false);
  const [newSecretName, setNewSecretName] = useState("");
  const [newSecretValue, setNewSecretValue] = useState("");
  const [newSecretType, setNewSecretType] = useState("api_key");
  const [addingSecret, setAddingSecret] = useState(false);

  // WebSocket streaming state for real-time terminal output
  const [streamOutput, setStreamOutput] = useState("");
  const [streamExitCode, setStreamExitCode] = useState<number | null>(null);
  const [isStreaming, setIsStreaming] = useState(false);

  const fetchWorkspace = useCallback(async () => {
    const result = await api.getWorkspace(workspaceId);
    if (result.error) {
      setError(result.error);
    } else if (result.data) {
      setWorkspace(result.data);
      setError(null);
    }
  }, [workspaceId]);

  const fetchPrompts = useCallback(async () => {
    const result = await api.listPrompts(workspaceId);
    if (result.error) {
      console.error("Failed to fetch prompts:", result.error);
    } else if (result.data) {
      setPrompts(result.data.prompts || []);
    }
  }, [workspaceId]);

  const fetchSecrets = useCallback(async () => {
    const result = await api.listSecrets(workspaceId);
    if (result.error) {
      console.error("Failed to fetch secrets:", result.error);
    } else if (result.data) {
      setSecrets(result.data.secrets || []);
    }
  }, [workspaceId]);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    await Promise.all([fetchWorkspace(), fetchPrompts(), fetchSecrets()]);
    setLoading(false);
  }, [fetchWorkspace, fetchPrompts, fetchSecrets]);

  useEffect(() => {
    void fetchAll();
  }, [fetchAll]);

  // Auto-refresh prompts when workspace is ready
  // Poll faster (2s) when there are running/pending prompts, slower (5s) otherwise
  useEffect(() => {
    if (workspace?.status === "ready") {
      const hasActivePrompts = prompts.some(
        (p) => p.status === "running" || p.status === "pending"
      );
      const pollInterval = hasActivePrompts ? 2000 : 5000;

      const interval = setInterval(() => {
        fetchPrompts();
      }, pollInterval);
      return () => clearInterval(interval);
    }
  }, [workspace?.status, prompts, fetchPrompts]);

  // WebSocket streaming hook
  const {
    output: wsOutput,
    isConnected,
    isStreaming: wsIsStreaming,
    exitCode: wsExitCode,
    error: wsError,
    connect: wsConnect,
    disconnect: wsDisconnect,
    sendPrompt: wsSendPrompt,
    clearOutput: wsClearOutput,
  } = useWorkspaceStream({ workspaceId, autoConnect: false });

  // Connect WebSocket when workspace is ready
  useEffect(() => {
    if (workspace?.status === "ready") {
      wsConnect();
    }
    return () => {
      wsDisconnect();
    };
  }, [workspace?.status, wsConnect, wsDisconnect]);

  // Update streaming state from WebSocket
  useEffect(() => {
    if (wsOutput) {
      // Append to stream output (wsOutput is cumulative in the hook)
      setStreamOutput(wsOutput);
      setIsStreaming(wsIsStreaming);
    }
    if (wsExitCode !== null) {
      setStreamExitCode(wsExitCode);
      setIsStreaming(false);
      // Refresh prompts list after completion
      fetchPrompts();
    }
    if (wsError) {
      setError(wsError);
    }
  }, [wsOutput, wsIsStreaming, wsExitCode, wsError, fetchPrompts]);

  const handleSubmitPrompt = useCallback(async (
    prompt: string,
    systemPrompt?: string,
    workingDirectory?: string
  ) => {
    // Clear previous streaming output
    setStreamOutput("");
    setStreamExitCode(null);
    setIsStreaming(true);
    wsClearOutput();

    if (isConnected) {
      // Use WebSocket for real-time streaming
      // WebSocket handles execution and persists to database
      wsSendPrompt(prompt, systemPrompt, workingDirectory);
    } else {
      // Fallback to REST API only
      const result = await api.submitPrompt(workspaceId, {
        prompt,
        system_prompt: systemPrompt,
        working_directory: workingDirectory,
      });

      if (result.error) {
        setError(result.error);
        throw new Error(result.error);
      } else {
        setError(null);
        fetchPrompts();
      }
    }
  }, [workspaceId, fetchPrompts, isConnected, wsSendPrompt, wsClearOutput]);

  const handleAddSecret = async () => {
    if (!newSecretName.trim() || !newSecretValue.trim()) return;

    setAddingSecret(true);
    const result = await api.addSecret(workspaceId, {
      name: newSecretName,
      value: newSecretValue,
      type: newSecretType,
    });

    if (result.error) {
      setError(result.error);
    } else {
      setNewSecretName("");
      setNewSecretValue("");
      setNewSecretType("api_key");
      setShowAddSecret(false);
      fetchSecrets();
    }
    setAddingSecret(false);
  };

  const handleDeleteSecret = async (secretId: string) => {
    const result = await api.deleteSecret(workspaceId, secretId);
    if (result.error) {
      setError(result.error);
    } else {
      fetchSecrets();
    }
  };

  const handleDeleteWorkspace = async () => {
    const result = await api.deleteWorkspace(workspaceId);
    if (result.error) {
      setError(result.error);
    } else {
      router.push("/workspaces");
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!workspace) {
    return (
      <div className="p-8">
        <Card className="border-red-500">
          <CardContent className="pt-6">
            <p className="text-red-500">
              Workspace not found or failed to load: {error}
            </p>
            <Link href="/workspaces" className="mt-4 inline-block">
              <Button variant="outline">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Workspaces
              </Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    );
  }

  const aiInfo = getAIAssistantInfo(workspace.ai_assistant);
  const isReady = workspace.status.toLowerCase() === "ready";

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link href="/workspaces">
            <Button variant="ghost" size="sm">
              <ArrowLeft className="h-4 w-4" />
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold flex items-center gap-3">
              {workspace.name}
              {getStatusBadge(workspace.status)}
            </h1>
            {workspace.description && (
              <p className="text-muted-foreground mt-1">
                {workspace.description}
              </p>
            )}
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={fetchAll} disabled={loading}>
            <RefreshCw
              className={`h-4 w-4 mr-2 ${loading ? "animate-spin" : ""}`}
            />
            Refresh
          </Button>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="destructive">
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete Workspace</AlertDialogTitle>
                <AlertDialogDescription>
                  Are you sure you want to delete &quot;{workspace.name}&quot;?
                  This will also delete the associated VM and all data.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  onClick={handleDeleteWorkspace}
                  className="bg-red-500 hover:bg-red-600"
                >
                  Delete
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </div>

      {error && (
        <Card className="border-red-500">
          <CardContent className="pt-6">
            <p className="text-red-500">{error}</p>
          </CardContent>
        </Card>
      )}

      {/* Overview Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              {aiInfo.icon}
              AI Assistant
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{aiInfo.name}</p>
            <p className="text-xs text-muted-foreground">{aiInfo.description}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <FolderCode className="h-4 w-4" />
              Working Directory
            </CardTitle>
          </CardHeader>
          <CardContent>
            <code className="text-sm bg-muted px-2 py-1 rounded">
              {workspace.working_directory}
            </code>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Cpu className="h-4 w-4" />
              Resources
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              VM: {workspace.vm_name || workspace.vm_id || "Pending"}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Clock className="h-4 w-4" />
              Created
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm">
              {new Date(workspace.created_at).toLocaleString()}
            </p>
            {workspace.ready_at && (
              <p className="text-xs text-muted-foreground">
                Ready: {new Date(workspace.ready_at).toLocaleString()}
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Main Content Tabs */}
      <Tabs defaultValue="prompts" className="space-y-4">
        <TabsList>
          <TabsTrigger value="prompts">Prompts</TabsTrigger>
          <TabsTrigger value="terminal">
            <Terminal className="h-4 w-4 mr-1" />
            Terminal
          </TabsTrigger>
          <TabsTrigger value="preparation">Preparation Steps</TabsTrigger>
          <TabsTrigger value="secrets">Secrets ({secrets.length})</TabsTrigger>
        </TabsList>

        {/* Prompts Tab */}
        <TabsContent value="prompts" className="space-y-4">
          {/* Prompt Submission */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Send className="h-5 w-5" />
                Submit Prompt
              </CardTitle>
              <CardDescription>
                {isReady
                  ? `Send a prompt to ${aiInfo.name} running in your workspace`
                  : "Workspace is not ready yet. Please wait for preparation to complete."}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <PromptInput
                onSubmit={handleSubmitPrompt}
                disabled={!isReady}
                defaultWorkingDirectory={workspace.working_directory || "/workspace"}
                aiAssistantName={aiInfo.name}
              />
            </CardContent>
          </Card>

          {/* Prompt History */}
          <Card>
            <CardHeader>
              <CardTitle>Prompt History</CardTitle>
              <CardDescription>
                {prompts.length} prompt{prompts.length !== 1 ? "s" : ""} submitted
              </CardDescription>
            </CardHeader>
            <CardContent>
              <PromptList
                prompts={prompts}
                aiAssistantName={aiInfo.name}
              />
            </CardContent>
          </Card>
        </TabsContent>

        {/* Terminal Tab */}
        <TabsContent value="terminal" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Terminal className="h-5 w-5" />
                Terminal Output
                {isConnected && (
                  <Badge variant="outline" className="ml-2 text-green-500 border-green-500">
                    <span className="w-2 h-2 bg-green-500 rounded-full mr-1 animate-pulse" />
                    Live
                  </Badge>
                )}
              </CardTitle>
              <CardDescription>
                {isStreaming
                  ? "Streaming output in real-time..."
                  : prompts.length > 0
                  ? `Showing output from the most recent prompt execution`
                  : `No prompts executed yet. Submit a prompt to see terminal output.`}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {(() => {
                // Prefer streaming output when available
                if (streamOutput || isStreaming) {
                  return (
                    <div className="space-y-4">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          {isStreaming ? (
                            <Badge className="bg-blue-500">
                              <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                              Streaming
                            </Badge>
                          ) : streamExitCode !== null ? (
                            streamExitCode === 0 ? (
                              <Badge className="bg-green-500">Completed</Badge>
                            ) : (
                              <Badge variant="destructive">Failed</Badge>
                            )
                          ) : (
                            <Badge variant="secondary">Ready</Badge>
                          )}
                          {streamExitCode !== null && (
                            <Badge
                              variant={streamExitCode === 0 ? "outline" : "destructive"}
                              className="font-mono"
                            >
                              Exit: {streamExitCode}
                            </Badge>
                          )}
                        </div>
                        {isConnected && (
                          <span className="text-xs text-green-500">
                            WebSocket connected
                          </span>
                        )}
                      </div>

                      <TerminalView
                        output={streamOutput || (isStreaming ? "Waiting for output..." : "No output")}
                        height="500px"
                        title={`${aiInfo.name} Output (Live)`}
                        readOnly={true}
                      />
                    </div>
                  );
                }

                // Fall back to polled prompt output
                const runningPrompt = prompts.find(
                  (p) => p.status.toLowerCase() === "running"
                );
                const completedPrompt = prompts.find(
                  (p) =>
                    p.status.toLowerCase() === "completed" ||
                    p.status.toLowerCase() === "failed"
                );
                const activePrompt = runningPrompt || completedPrompt;

                if (!activePrompt) {
                  return (
                    <div className="text-center py-12 text-muted-foreground">
                      <Terminal className="h-16 w-16 mx-auto mb-4 opacity-30" />
                      <p className="text-lg">No output to display</p>
                      <p className="text-sm mt-2">
                        Submit a prompt from the Prompts tab to see terminal output here
                      </p>
                    </div>
                  );
                }

                // Combine stdout and stderr for display
                const output = [
                  activePrompt.stdout,
                  activePrompt.stderr ? `\n\x1b[31m${activePrompt.stderr}\x1b[0m` : "",
                  activePrompt.error ? `\n\x1b[31mError: ${activePrompt.error}\x1b[0m` : "",
                ]
                  .filter(Boolean)
                  .join("");

                const isRunning = activePrompt.status.toLowerCase() === "running";

                return (
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        {isRunning ? (
                          <Badge className="bg-blue-500">
                            <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                            Running
                          </Badge>
                        ) : activePrompt.status.toLowerCase() === "completed" ? (
                          <Badge className="bg-green-500">Completed</Badge>
                        ) : (
                          <Badge variant="destructive">Failed</Badge>
                        )}
                        <span className="text-sm text-muted-foreground">
                          {new Date(activePrompt.created_at).toLocaleString()}
                        </span>
                        {activePrompt.exit_code !== undefined &&
                          activePrompt.exit_code !== null && (
                            <Badge
                              variant={
                                activePrompt.exit_code === 0 ? "outline" : "destructive"
                              }
                              className="font-mono"
                            >
                              Exit: {activePrompt.exit_code}
                            </Badge>
                          )}
                      </div>
                      <div className="text-sm text-muted-foreground font-mono truncate max-w-md">
                        {activePrompt.prompt.substring(0, 50)}
                        {activePrompt.prompt.length > 50 ? "..." : ""}
                      </div>
                    </div>

                    <TerminalView
                      output={output || (isRunning ? "Waiting for output..." : "No output")}
                      height="500px"
                      title={`${aiInfo.name} Output`}
                      readOnly={true}
                    />
                  </div>
                );
              })()}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Preparation Steps Tab */}
        <TabsContent value="preparation">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <GitBranch className="h-5 w-5" />
                Preparation Steps
              </CardTitle>
              <CardDescription>
                Setup steps executed when the workspace was created
              </CardDescription>
            </CardHeader>
            <CardContent>
              {!workspace.prep_steps || workspace.prep_steps.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  <CheckCircle2 className="h-12 w-12 mx-auto mb-4 opacity-50" />
                  <p>No preparation steps configured</p>
                </div>
              ) : (
                <div className="space-y-4">
                  {workspace.prep_steps.map((step: PrepStep, index: number) => (
                    <Collapsible key={step.id}>
                      <div className="border rounded-lg p-4">
                        <CollapsibleTrigger asChild>
                          <div className="flex items-center justify-between cursor-pointer">
                            <div className="flex items-center gap-3">
                              <div className="flex items-center justify-center w-6 h-6 rounded-full bg-muted text-sm font-medium">
                                {index + 1}
                              </div>
                              {getPrepStepStatusIcon(step.status)}
                              <div>
                                <p className="font-medium capitalize">
                                  {step.step_type.replace("_", " ")}
                                </p>
                                <p className="text-xs text-muted-foreground">
                                  {step.step_type === "git_clone" &&
                                    (step.config as { url?: string })?.url}
                                  {step.step_type === "script" && "Custom script"}
                                  {step.step_type === "env_var" &&
                                    `Set ${(step.config as { name?: string })?.name}`}
                                </p>
                              </div>
                            </div>
                            <div className="flex items-center gap-2">
                              {step.exit_code !== undefined && (
                                <Badge
                                  variant={
                                    step.exit_code === 0 ? "outline" : "destructive"
                                  }
                                >
                                  Exit: {step.exit_code}
                                </Badge>
                              )}
                              <ChevronRight className="h-4 w-4" />
                            </div>
                          </div>
                        </CollapsibleTrigger>
                        <CollapsibleContent className="pt-4 space-y-2">
                          <div>
                            <Label className="text-xs">Configuration</Label>
                            <pre className="mt-1 p-3 bg-muted rounded text-sm overflow-x-auto">
                              {JSON.stringify(step.config, null, 2)}
                            </pre>
                          </div>
                          {step.stdout && (
                            <div>
                              <Label className="text-xs text-green-600">
                                Output
                              </Label>
                              <pre className="mt-1 p-3 bg-green-50 dark:bg-green-950 rounded text-sm whitespace-pre-wrap overflow-x-auto max-h-48">
                                {step.stdout}
                              </pre>
                            </div>
                          )}
                          {step.stderr && (
                            <div>
                              <Label className="text-xs text-red-600">
                                Errors
                              </Label>
                              <pre className="mt-1 p-3 bg-red-50 dark:bg-red-950 rounded text-sm whitespace-pre-wrap overflow-x-auto max-h-48">
                                {step.stderr}
                              </pre>
                            </div>
                          )}
                        </CollapsibleContent>
                      </div>
                    </Collapsible>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Secrets Tab */}
        <TabsContent value="secrets">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="flex items-center gap-2">
                    <Key className="h-5 w-5" />
                    Secrets
                  </CardTitle>
                  <CardDescription>
                    API keys and credentials available in the workspace
                  </CardDescription>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowAddSecret(!showAddSecret)}
                >
                  <Plus className="h-4 w-4 mr-2" />
                  Add Secret
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Add Secret Form */}
              {showAddSecret && (
                <div className="border rounded-lg p-4 space-y-4 bg-muted/50">
                  <h4 className="font-medium">Add New Secret</h4>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <Label htmlFor="secretName">Name</Label>
                      <Input
                        id="secretName"
                        placeholder="ANTHROPIC_API_KEY"
                        value={newSecretName}
                        onChange={(e) => setNewSecretName(e.target.value)}
                        className="mt-1"
                      />
                    </div>
                    <div>
                      <Label htmlFor="secretValue">Value</Label>
                      <Input
                        id="secretValue"
                        type="password"
                        placeholder="sk-..."
                        value={newSecretValue}
                        onChange={(e) => setNewSecretValue(e.target.value)}
                        className="mt-1"
                      />
                    </div>
                    <div>
                      <Label htmlFor="secretType">Type</Label>
                      <select
                        id="secretType"
                        value={newSecretType}
                        onChange={(e) => setNewSecretType(e.target.value)}
                        className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                      >
                        <option value="api_key">API Key</option>
                        <option value="token">Token</option>
                        <option value="ssh_key">SSH Key</option>
                        <option value="password">Password</option>
                        <option value="other">Other</option>
                      </select>
                    </div>
                  </div>
                  <div className="flex justify-end gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setShowAddSecret(false)}
                    >
                      Cancel
                    </Button>
                    <Button
                      size="sm"
                      onClick={handleAddSecret}
                      disabled={
                        addingSecret ||
                        !newSecretName.trim() ||
                        !newSecretValue.trim()
                      }
                    >
                      {addingSecret ? (
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      ) : (
                        <Plus className="h-4 w-4 mr-2" />
                      )}
                      Add Secret
                    </Button>
                  </div>
                </div>
              )}

              {/* Secrets List */}
              {secrets.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  <Key className="h-12 w-12 mx-auto mb-4 opacity-50" />
                  <p>No secrets configured</p>
                  <p className="text-sm mt-1">
                    Add API keys or credentials to use in your workspace
                  </p>
                </div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Type</TableHead>
                      <TableHead>Scope</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {secrets.map((secret) => (
                      <TableRow key={secret.id}>
                        <TableCell>
                          <code className="text-sm bg-muted px-2 py-1 rounded">
                            {secret.name}
                          </code>
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline">{secret.type}</Badge>
                        </TableCell>
                        <TableCell>
                          <Badge variant="secondary">{secret.scope}</Badge>
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {new Date(secret.created_at).toLocaleDateString()}
                        </TableCell>
                        <TableCell className="text-right">
                          <AlertDialog>
                            <AlertDialogTrigger asChild>
                              <Button variant="ghost" size="sm">
                                <Trash2 className="h-4 w-4 text-red-500" />
                              </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>Delete Secret</AlertDialogTitle>
                                <AlertDialogDescription>
                                  Are you sure you want to delete the secret
                                  &quot;{secret.name}&quot;? This cannot be undone.
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction
                                  onClick={() => handleDeleteSecret(secret.id)}
                                  className="bg-red-500 hover:bg-red-600"
                                >
                                  Delete
                                </AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
