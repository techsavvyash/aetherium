"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { CreateEnvironmentRequest, MCPServerConfig } from "@/lib/types";
import { MCP_PRESETS } from "@/lib/types";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import {
  Layers,
  ArrowLeft,
  ArrowRight,
  Check,
  Cpu,
  HardDrive,
  GitBranch,
  Wrench,
  Server,
  Settings,
  Plus,
  X,
  Terminal,
  Globe,
} from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";

type Step = 1 | 2 | 3 | 4 | 5;

const AVAILABLE_TOOLS = [
  { name: "git", description: "Version control" },
  { name: "nodejs@20", description: "Node.js runtime" },
  { name: "bun", description: "Fast JavaScript runtime" },
  { name: "claude-code", description: "Claude Code CLI" },
  { name: "go", description: "Go programming language" },
  { name: "python", description: "Python runtime" },
  { name: "rust", description: "Rust programming language" },
  { name: "docker", description: "Container runtime" },
];

export default function CreateEnvironmentPage() {
  const router = useRouter();
  const [step, setStep] = useState<Step>(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Form state
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [vcpus, setVcpus] = useState(2);
  const [memoryMb, setMemoryMb] = useState(2048);
  const [gitRepoUrl, setGitRepoUrl] = useState("");
  const [gitBranch, setGitBranch] = useState("main");
  const [workingDirectory, setWorkingDirectory] = useState("/workspace");
  const [tools, setTools] = useState<string[]>(["git", "nodejs@20", "claude-code"]);
  const [mcpServers, setMcpServers] = useState<MCPServerConfig[]>([]);
  const [envVars, setEnvVars] = useState<Record<string, string>>({});
  const [idleTimeoutSeconds, setIdleTimeoutSeconds] = useState(1800);

  // Custom MCP server form
  const [customMcpName, setCustomMcpName] = useState("");
  const [customMcpType, setCustomMcpType] = useState<"stdio" | "http">("stdio");
  const [customMcpCommand, setCustomMcpCommand] = useState("");
  const [customMcpArgs, setCustomMcpArgs] = useState("");
  const [customMcpUrl, setCustomMcpUrl] = useState("");

  // Env var form
  const [newEnvKey, setNewEnvKey] = useState("");
  const [newEnvValue, setNewEnvValue] = useState("");

  const toggleTool = (tool: string) => {
    setTools((prev) =>
      prev.includes(tool) ? prev.filter((t) => t !== tool) : [...prev, tool]
    );
  };

  const addMcpPreset = (presetName: string) => {
    const preset = MCP_PRESETS[presetName];
    if (preset && !mcpServers.some((s) => s.name === preset.name)) {
      setMcpServers((prev) => [...prev, preset]);
    }
  };

  const addCustomMcpServer = () => {
    if (!customMcpName) return;

    const newServer: MCPServerConfig = {
      name: customMcpName,
      type: customMcpType,
      ...(customMcpType === "stdio"
        ? { command: customMcpCommand, args: customMcpArgs.split(" ").filter(Boolean) }
        : { url: customMcpUrl }),
    };

    setMcpServers((prev) => [...prev, newServer]);
    setCustomMcpName("");
    setCustomMcpCommand("");
    setCustomMcpArgs("");
    setCustomMcpUrl("");
  };

  const removeMcpServer = (name: string) => {
    setMcpServers((prev) => prev.filter((s) => s.name !== name));
  };

  const addEnvVar = () => {
    if (newEnvKey && newEnvValue) {
      setEnvVars((prev) => ({ ...prev, [newEnvKey]: newEnvValue }));
      setNewEnvKey("");
      setNewEnvValue("");
    }
  };

  const removeEnvVar = (key: string) => {
    setEnvVars((prev) => {
      const next = { ...prev };
      delete next[key];
      return next;
    });
  };

  const handleSubmit = async () => {
    setLoading(true);
    setError(null);

    const req: CreateEnvironmentRequest = {
      name,
      description: description || undefined,
      vcpus,
      memory_mb: memoryMb,
      git_repo_url: gitRepoUrl || undefined,
      git_branch: gitBranch,
      working_directory: workingDirectory,
      tools,
      mcp_servers: mcpServers.length > 0 ? mcpServers : undefined,
      env_vars: Object.keys(envVars).length > 0 ? envVars : undefined,
      idle_timeout_seconds: idleTimeoutSeconds,
    };

    const result = await api.createEnvironment(req);

    if (result.error) {
      setError(result.error);
      setLoading(false);
    } else {
      router.push("/environments");
    }
  };

  const canProceed = () => {
    switch (step) {
      case 1:
        return name.trim().length > 0;
      case 2:
        return vcpus > 0 && memoryMb > 0;
      case 3:
        return true;
      case 4:
        return true;
      case 5:
        return true;
      default:
        return false;
    }
  };

  return (
    <div className="p-8 max-w-4xl mx-auto">
      <div className="flex items-center gap-4 mb-8">
        <Button variant="ghost" onClick={() => router.back()}>
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back
        </Button>
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Layers className="h-8 w-8" />
            Create Environment
          </h1>
          <p className="text-muted-foreground mt-1">
            Define a reusable VM template with tools and MCP servers
          </p>
        </div>
      </div>

      {/* Progress Steps */}
      <div className="flex items-center justify-between mb-8">
        {[1, 2, 3, 4, 5].map((s) => (
          <div
            key={s}
            className={`flex items-center ${s < 5 ? "flex-1" : ""}`}
          >
            <div
              className={`w-10 h-10 rounded-full flex items-center justify-center ${
                step >= s
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground"
              }`}
            >
              {step > s ? <Check className="h-5 w-5" /> : s}
            </div>
            {s < 5 && (
              <div
                className={`flex-1 h-1 mx-2 ${
                  step > s ? "bg-primary" : "bg-muted"
                }`}
              />
            )}
          </div>
        ))}
      </div>

      {error && (
        <Card className="mb-4 border-red-500">
          <CardContent className="pt-6">
            <p className="text-red-500">{error}</p>
          </CardContent>
        </Card>
      )}

      {/* Step 1: Basic Info */}
      {step === 1 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings className="h-5 w-5" />
              Basic Information
            </CardTitle>
            <CardDescription>
              Name and describe your environment template
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., python-dev, nodejs-fullstack"
              />
            </div>
            <div>
              <Label htmlFor="description">Description (optional)</Label>
              <Textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Describe what this environment is for..."
              />
            </div>
          </CardContent>
        </Card>
      )}

      {/* Step 2: Resources */}
      {step === 2 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Cpu className="h-5 w-5" />
              VM Resources
            </CardTitle>
            <CardDescription>
              Configure CPU, memory, and timeout settings
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="vcpus" className="flex items-center gap-2">
                  <Cpu className="h-4 w-4" />
                  vCPUs
                </Label>
                <Input
                  id="vcpus"
                  type="number"
                  min={1}
                  max={16}
                  value={vcpus}
                  onChange={(e) => setVcpus(parseInt(e.target.value) || 1)}
                />
              </div>
              <div>
                <Label htmlFor="memory" className="flex items-center gap-2">
                  <HardDrive className="h-4 w-4" />
                  Memory (MB)
                </Label>
                <Input
                  id="memory"
                  type="number"
                  min={256}
                  max={32768}
                  step={256}
                  value={memoryMb}
                  onChange={(e) => setMemoryMb(parseInt(e.target.value) || 256)}
                />
              </div>
            </div>
            <div>
              <Label htmlFor="timeout">Idle Timeout (seconds)</Label>
              <Input
                id="timeout"
                type="number"
                min={60}
                max={86400}
                value={idleTimeoutSeconds}
                onChange={(e) =>
                  setIdleTimeoutSeconds(parseInt(e.target.value) || 1800)
                }
              />
              <p className="text-sm text-muted-foreground mt-1">
                VM will be destroyed after {Math.floor(idleTimeoutSeconds / 60)} minutes of inactivity
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Step 3: Repository */}
      {step === 3 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <GitBranch className="h-5 w-5" />
              Repository (Optional)
            </CardTitle>
            <CardDescription>
              Clone a repository when the workspace starts
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <Label htmlFor="gitRepoUrl">Git Repository URL</Label>
              <Input
                id="gitRepoUrl"
                value={gitRepoUrl}
                onChange={(e) => setGitRepoUrl(e.target.value)}
                placeholder="https://github.com/user/repo.git"
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="gitBranch">Branch</Label>
                <Input
                  id="gitBranch"
                  value={gitBranch}
                  onChange={(e) => setGitBranch(e.target.value)}
                  placeholder="main"
                />
              </div>
              <div>
                <Label htmlFor="workingDirectory">Working Directory</Label>
                <Input
                  id="workingDirectory"
                  value={workingDirectory}
                  onChange={(e) => setWorkingDirectory(e.target.value)}
                  placeholder="/workspace"
                />
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Step 4: Tools */}
      {step === 4 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Wrench className="h-5 w-5" />
              Tools
            </CardTitle>
            <CardDescription>
              Select tools to install in the VM
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              {AVAILABLE_TOOLS.map((tool) => (
                <div
                  key={tool.name}
                  className="flex items-center space-x-2 p-3 rounded border hover:bg-muted cursor-pointer"
                  onClick={() => toggleTool(tool.name)}
                >
                  <Checkbox
                    checked={tools.includes(tool.name)}
                    onCheckedChange={() => toggleTool(tool.name)}
                  />
                  <div>
                    <p className="font-medium">{tool.name}</p>
                    <p className="text-sm text-muted-foreground">
                      {tool.description}
                    </p>
                  </div>
                </div>
              ))}
            </div>
            <div className="flex flex-wrap gap-2 mt-4">
              <span className="text-sm text-muted-foreground">Selected:</span>
              {tools.map((tool) => (
                <Badge key={tool} variant="secondary">
                  {tool}
                  <button
                    onClick={() => toggleTool(tool)}
                    className="ml-1 hover:text-red-500"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </Badge>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Step 5: MCP Servers & Env Vars */}
      {step === 5 && (
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5" />
                MCP Servers (Optional)
              </CardTitle>
              <CardDescription>
                Configure Model Context Protocol servers for Claude Code
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Presets */}
              <div>
                <Label>Quick Add Presets</Label>
                <div className="flex flex-wrap gap-2 mt-2">
                  {Object.keys(MCP_PRESETS).map((preset) => (
                    <Button
                      key={preset}
                      variant="outline"
                      size="sm"
                      onClick={() => addMcpPreset(preset)}
                      disabled={mcpServers.some((s) => s.name === preset)}
                    >
                      <Plus className="h-3 w-3 mr-1" />
                      {preset}
                    </Button>
                  ))}
                </div>
              </div>

              {/* Custom MCP Server */}
              <div className="border rounded p-4 space-y-3">
                <Label>Add Custom MCP Server</Label>
                <div className="grid grid-cols-2 gap-2">
                  <Input
                    placeholder="Server name"
                    value={customMcpName}
                    onChange={(e) => setCustomMcpName(e.target.value)}
                  />
                  <div className="flex gap-2">
                    <Button
                      variant={customMcpType === "stdio" ? "default" : "outline"}
                      size="sm"
                      onClick={() => setCustomMcpType("stdio")}
                    >
                      <Terminal className="h-3 w-3 mr-1" />
                      Stdio
                    </Button>
                    <Button
                      variant={customMcpType === "http" ? "default" : "outline"}
                      size="sm"
                      onClick={() => setCustomMcpType("http")}
                    >
                      <Globe className="h-3 w-3 mr-1" />
                      HTTP
                    </Button>
                  </div>
                </div>
                {customMcpType === "stdio" ? (
                  <div className="grid grid-cols-2 gap-2">
                    <Input
                      placeholder="Command (e.g., npx)"
                      value={customMcpCommand}
                      onChange={(e) => setCustomMcpCommand(e.target.value)}
                    />
                    <Input
                      placeholder="Args (space-separated)"
                      value={customMcpArgs}
                      onChange={(e) => setCustomMcpArgs(e.target.value)}
                    />
                  </div>
                ) : (
                  <Input
                    placeholder="URL (e.g., http://localhost:3001/mcp)"
                    value={customMcpUrl}
                    onChange={(e) => setCustomMcpUrl(e.target.value)}
                  />
                )}
                <Button
                  size="sm"
                  onClick={addCustomMcpServer}
                  disabled={!customMcpName}
                >
                  <Plus className="h-3 w-3 mr-1" />
                  Add Server
                </Button>
              </div>

              {/* Current MCP Servers */}
              {mcpServers.length > 0 && (
                <div>
                  <Label>Configured Servers</Label>
                  <div className="space-y-2 mt-2">
                    {mcpServers.map((server) => (
                      <div
                        key={server.name}
                        className="flex items-center justify-between p-2 border rounded"
                      >
                        <div>
                          <span className="font-medium">{server.name}</span>
                          <Badge variant="outline" className="ml-2">
                            {server.type}
                          </Badge>
                          <p className="text-sm text-muted-foreground">
                            {server.type === "stdio"
                              ? `${server.command} ${server.args?.join(" ")}`
                              : server.url}
                          </p>
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => removeMcpServer(server.name)}
                        >
                          <X className="h-4 w-4 text-red-500" />
                        </Button>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Environment Variables */}
          <Card>
            <CardHeader>
              <CardTitle>Environment Variables (Optional)</CardTitle>
              <CardDescription>
                Set environment variables for the workspace
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex gap-2">
                <Input
                  placeholder="KEY"
                  value={newEnvKey}
                  onChange={(e) => setNewEnvKey(e.target.value.toUpperCase())}
                  className="w-1/3"
                />
                <Input
                  placeholder="value"
                  value={newEnvValue}
                  onChange={(e) => setNewEnvValue(e.target.value)}
                  className="flex-1"
                />
                <Button onClick={addEnvVar} disabled={!newEnvKey || !newEnvValue}>
                  <Plus className="h-4 w-4" />
                </Button>
              </div>
              {Object.entries(envVars).length > 0 && (
                <div className="space-y-2">
                  {Object.entries(envVars).map(([key, value]) => (
                    <div
                      key={key}
                      className="flex items-center justify-between p-2 border rounded"
                    >
                      <code className="text-sm">
                        {key}=<span className="text-muted-foreground">{value}</span>
                      </code>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => removeEnvVar(key)}
                      >
                        <X className="h-4 w-4 text-red-500" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}

      {/* Navigation Buttons */}
      <div className="flex justify-between mt-6">
        <Button
          variant="outline"
          onClick={() => setStep((s) => (s > 1 ? ((s - 1) as Step) : s))}
          disabled={step === 1}
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Previous
        </Button>
        {step < 5 ? (
          <Button
            onClick={() => setStep((s) => (s < 5 ? ((s + 1) as Step) : s))}
            disabled={!canProceed()}
          >
            Next
            <ArrowRight className="h-4 w-4 ml-2" />
          </Button>
        ) : (
          <Button onClick={handleSubmit} disabled={loading || !canProceed()}>
            {loading ? "Creating..." : "Create Environment"}
            <Check className="h-4 w-4 ml-2" />
          </Button>
        )}
      </div>
    </div>
  );
}
