"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type {
  CreateWorkspaceRequest,
  SecretRequest,
  PrepStepRequest,
} from "@/lib/types";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sparkles,
  ArrowLeft,
  ArrowRight,
  Check,
  Plus,
  Trash2,
  Loader2,
  Bot,
  GitBranch,
  Key,
  FileCode,
} from "lucide-react";
import Link from "next/link";

const steps = [
  { id: 1, name: "Basic Info", description: "Name and resources" },
  { id: 2, name: "AI Assistant", description: "Select your AI" },
  { id: 3, name: "Secrets", description: "API keys and credentials" },
  { id: 4, name: "Preparation", description: "Git repos and scripts" },
  { id: 5, name: "Review", description: "Confirm and create" },
];

export default function CreateWorkspacePage() {
  const router = useRouter();
  const [currentStep, setCurrentStep] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Form state
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [vcpus, setVcpus] = useState(2);
  const [memoryMb, setMemoryMb] = useState(2048);
  const [aiAssistant, setAiAssistant] = useState<"claude-code" | "ampcode">("claude-code");
  const [workingDirectory, setWorkingDirectory] = useState("/workspace");
  const [secrets, setSecrets] = useState<SecretRequest[]>([]);
  const [prepSteps, setPrepSteps] = useState<PrepStepRequest[]>([]);

  // Temporary state for adding new items
  const [newSecretName, setNewSecretName] = useState("");
  const [newSecretValue, setNewSecretValue] = useState("");
  const [newSecretType, setNewSecretType] = useState("api_key");

  const [newGitUrl, setNewGitUrl] = useState("");
  const [newGitBranch, setNewGitBranch] = useState("");
  const [newScript, setNewScript] = useState("");

  const addSecret = () => {
    if (newSecretName && newSecretValue) {
      setSecrets([
        ...secrets,
        {
          name: newSecretName,
          value: newSecretValue,
          type: newSecretType,
        },
      ]);
      setNewSecretName("");
      setNewSecretValue("");
      setNewSecretType("api_key");
    }
  };

  const removeSecret = (index: number) => {
    setSecrets(secrets.filter((_, i) => i !== index));
  };

  const addGitClone = () => {
    if (newGitUrl) {
      setPrepSteps([
        ...prepSteps,
        {
          type: "git_clone",
          order: prepSteps.length + 1,
          config: {
            url: newGitUrl,
            branch: newGitBranch || undefined,
            target_dir: workingDirectory,
          },
        },
      ]);
      setNewGitUrl("");
      setNewGitBranch("");
    }
  };

  const addScript = () => {
    if (newScript) {
      setPrepSteps([
        ...prepSteps,
        {
          type: "script",
          order: prepSteps.length + 1,
          config: {
            script: newScript,
            working_directory: workingDirectory,
          },
        },
      ]);
      setNewScript("");
    }
  };

  const removePrepStep = (index: number) => {
    setPrepSteps(prepSteps.filter((_, i) => i !== index));
  };

  const handleSubmit = async () => {
    setLoading(true);
    setError(null);

    const request: CreateWorkspaceRequest = {
      name,
      description: description || undefined,
      vcpus,
      memory_mb: memoryMb,
      ai_assistant: aiAssistant,
      working_directory: workingDirectory,
      secrets: secrets.length > 0 ? secrets : undefined,
      prep_steps: prepSteps.length > 0 ? prepSteps : undefined,
    };

    const result = await api.createWorkspace(request);

    if (result.error) {
      setError(result.error);
      setLoading(false);
    } else {
      router.push(`/workspaces/${result.data?.workspace_id}`);
    }
  };

  const canProceed = () => {
    switch (currentStep) {
      case 1:
        return name.trim().length > 0;
      default:
        return true;
    }
  };

  return (
    <div className="p-8 max-w-4xl mx-auto">
      <div className="flex items-center gap-4 mb-8">
        <Link href="/workspaces">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
        </Link>
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Sparkles className="h-8 w-8" />
            Create Workspace
          </h1>
          <p className="text-muted-foreground">
            Set up a new AI workspace environment
          </p>
        </div>
      </div>

      {/* Progress Steps */}
      <div className="mb-8">
        <div className="flex items-center justify-between">
          {steps.map((step, index) => (
            <div
              key={step.id}
              className={`flex items-center ${
                index < steps.length - 1 ? "flex-1" : ""
              }`}
            >
              <div
                className={`flex items-center justify-center w-10 h-10 rounded-full border-2 ${
                  currentStep > step.id
                    ? "bg-primary border-primary text-primary-foreground"
                    : currentStep === step.id
                    ? "border-primary text-primary"
                    : "border-muted-foreground text-muted-foreground"
                }`}
              >
                {currentStep > step.id ? (
                  <Check className="h-5 w-5" />
                ) : (
                  step.id
                )}
              </div>
              {index < steps.length - 1 && (
                <div
                  className={`flex-1 h-1 mx-2 ${
                    currentStep > step.id ? "bg-primary" : "bg-muted"
                  }`}
                />
              )}
            </div>
          ))}
        </div>
        <div className="flex justify-between mt-2">
          {steps.map((step) => (
            <div
              key={step.id}
              className={`text-xs text-center ${
                currentStep >= step.id
                  ? "text-foreground"
                  : "text-muted-foreground"
              }`}
              style={{ width: "120px" }}
            >
              <div className="font-medium">{step.name}</div>
              <div className="text-muted-foreground">{step.description}</div>
            </div>
          ))}
        </div>
      </div>

      {error && (
        <Card className="mb-4 border-red-500">
          <CardContent className="pt-6">
            <p className="text-red-500">{error}</p>
          </CardContent>
        </Card>
      )}

      {/* Step Content */}
      <Card className="mb-8">
        <CardHeader>
          <CardTitle>{steps[currentStep - 1].name}</CardTitle>
          <CardDescription>{steps[currentStep - 1].description}</CardDescription>
        </CardHeader>
        <CardContent>
          {currentStep === 1 && (
            <div className="space-y-4">
              <div>
                <Label htmlFor="name">Workspace Name *</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="my-ai-workspace"
                />
              </div>
              <div>
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Optional description of this workspace"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="vcpus">vCPUs</Label>
                  <Select
                    value={vcpus.toString()}
                    onValueChange={(v) => setVcpus(parseInt(v))}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="1">1 vCPU</SelectItem>
                      <SelectItem value="2">2 vCPUs</SelectItem>
                      <SelectItem value="4">4 vCPUs</SelectItem>
                      <SelectItem value="8">8 vCPUs</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label htmlFor="memory">Memory</Label>
                  <Select
                    value={memoryMb.toString()}
                    onValueChange={(v) => setMemoryMb(parseInt(v))}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="1024">1 GB</SelectItem>
                      <SelectItem value="2048">2 GB</SelectItem>
                      <SelectItem value="4096">4 GB</SelectItem>
                      <SelectItem value="8192">8 GB</SelectItem>
                      <SelectItem value="16384">16 GB</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div>
                <Label htmlFor="workingDir">Working Directory</Label>
                <Input
                  id="workingDir"
                  value={workingDirectory}
                  onChange={(e) => setWorkingDirectory(e.target.value)}
                  placeholder="/workspace"
                />
              </div>
            </div>
          )}

          {currentStep === 2 && (
            <div className="space-y-6">
              <p className="text-muted-foreground">
                Select which AI coding assistant you want to use in this workspace.
              </p>
              <div className="grid grid-cols-2 gap-4">
                <Card
                  className={`cursor-pointer transition-all ${
                    aiAssistant === "claude-code"
                      ? "border-primary ring-2 ring-primary"
                      : "hover:border-muted-foreground"
                  }`}
                  onClick={() => setAiAssistant("claude-code")}
                >
                  <CardContent className="pt-6">
                    <div className="flex items-center gap-4">
                      <div className="p-3 bg-primary/10 rounded-lg">
                        <Bot className="h-8 w-8 text-primary" />
                      </div>
                      <div>
                        <h3 className="font-semibold">Claude Code</h3>
                        <p className="text-sm text-muted-foreground">
                          Anthropic&apos;s official coding assistant CLI
                        </p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
                <Card
                  className={`cursor-pointer transition-all ${
                    aiAssistant === "ampcode"
                      ? "border-primary ring-2 ring-primary"
                      : "hover:border-muted-foreground"
                  }`}
                  onClick={() => setAiAssistant("ampcode")}
                >
                  <CardContent className="pt-6">
                    <div className="flex items-center gap-4">
                      <div className="p-3 bg-primary/10 rounded-lg">
                        <Bot className="h-8 w-8 text-primary" />
                      </div>
                      <div>
                        <h3 className="font-semibold">Ampcode</h3>
                        <p className="text-sm text-muted-foreground">
                          Alternative AI coding assistant
                        </p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </div>
          )}

          {currentStep === 3 && (
            <div className="space-y-6">
              <p className="text-muted-foreground">
                Add API keys and credentials. These will be encrypted and made
                available as environment variables in the workspace.
              </p>

              {secrets.length > 0 && (
                <div className="space-y-2">
                  {secrets.map((secret, index) => (
                    <div
                      key={index}
                      className="flex items-center justify-between p-3 bg-muted rounded-lg"
                    >
                      <div className="flex items-center gap-3">
                        <Key className="h-4 w-4 text-muted-foreground" />
                        <span className="font-mono">{secret.name}</span>
                        <span className="text-sm text-muted-foreground">
                          ({secret.type})
                        </span>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => removeSecret(index)}
                      >
                        <Trash2 className="h-4 w-4 text-red-500" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}

              <div className="grid grid-cols-3 gap-4">
                <div>
                  <Label>Name</Label>
                  <Input
                    value={newSecretName}
                    onChange={(e) => setNewSecretName(e.target.value)}
                    placeholder="ANTHROPIC_API_KEY"
                  />
                </div>
                <div>
                  <Label>Value</Label>
                  <Input
                    type="password"
                    value={newSecretValue}
                    onChange={(e) => setNewSecretValue(e.target.value)}
                    placeholder="sk-..."
                  />
                </div>
                <div>
                  <Label>Type</Label>
                  <Select value={newSecretType} onValueChange={setNewSecretType}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="api_key">API Key</SelectItem>
                      <SelectItem value="token">Token</SelectItem>
                      <SelectItem value="ssh_key">SSH Key</SelectItem>
                      <SelectItem value="password">Password</SelectItem>
                      <SelectItem value="other">Other</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <Button onClick={addSecret} disabled={!newSecretName || !newSecretValue}>
                <Plus className="h-4 w-4 mr-2" />
                Add Secret
              </Button>
            </div>
          )}

          {currentStep === 4 && (
            <div className="space-y-6">
              <p className="text-muted-foreground">
                Configure preparation steps that run before your workspace is ready.
              </p>

              {prepSteps.length > 0 && (
                <div className="space-y-2">
                  {prepSteps.map((step, index) => (
                    <div
                      key={index}
                      className="flex items-center justify-between p-3 bg-muted rounded-lg"
                    >
                      <div className="flex items-center gap-3">
                        {step.type === "git_clone" ? (
                          <GitBranch className="h-4 w-4 text-muted-foreground" />
                        ) : (
                          <FileCode className="h-4 w-4 text-muted-foreground" />
                        )}
                        <span>
                          {step.type === "git_clone"
                            ? `Clone: ${(step.config as { url: string }).url}`
                            : `Script: ${(step.config as { script: string }).script.substring(0, 50)}...`}
                        </span>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => removePrepStep(index)}
                      >
                        <Trash2 className="h-4 w-4 text-red-500" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}

              <div className="space-y-4">
                <div>
                  <h4 className="font-medium mb-2 flex items-center gap-2">
                    <GitBranch className="h-4 w-4" />
                    Git Clone
                  </h4>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <Label>Repository URL</Label>
                      <Input
                        value={newGitUrl}
                        onChange={(e) => setNewGitUrl(e.target.value)}
                        placeholder="https://github.com/user/repo"
                      />
                    </div>
                    <div>
                      <Label>Branch (optional)</Label>
                      <Input
                        value={newGitBranch}
                        onChange={(e) => setNewGitBranch(e.target.value)}
                        placeholder="main"
                      />
                    </div>
                  </div>
                  <Button onClick={addGitClone} disabled={!newGitUrl} className="mt-2">
                    <Plus className="h-4 w-4 mr-2" />
                    Add Git Clone
                  </Button>
                </div>

                <div>
                  <h4 className="font-medium mb-2 flex items-center gap-2">
                    <FileCode className="h-4 w-4" />
                    Custom Script
                  </h4>
                  <Textarea
                    value={newScript}
                    onChange={(e) => setNewScript(e.target.value)}
                    placeholder="#!/bin/bash&#10;npm install"
                    rows={4}
                  />
                  <Button onClick={addScript} disabled={!newScript} className="mt-2">
                    <Plus className="h-4 w-4 mr-2" />
                    Add Script
                  </Button>
                </div>
              </div>
            </div>
          )}

          {currentStep === 5 && (
            <div className="space-y-6">
              <p className="text-muted-foreground">
                Review your workspace configuration before creating.
              </p>

              <div className="space-y-4">
                <div className="p-4 bg-muted rounded-lg">
                  <h4 className="font-medium mb-2">Basic Information</h4>
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <div>Name:</div>
                    <div className="font-mono">{name}</div>
                    <div>vCPUs:</div>
                    <div>{vcpus}</div>
                    <div>Memory:</div>
                    <div>{memoryMb / 1024} GB</div>
                    <div>Working Directory:</div>
                    <div className="font-mono">{workingDirectory}</div>
                  </div>
                </div>

                <div className="p-4 bg-muted rounded-lg">
                  <h4 className="font-medium mb-2">AI Assistant</h4>
                  <div className="flex items-center gap-2">
                    <Bot className="h-4 w-4" />
                    {aiAssistant === "claude-code" ? "Claude Code" : "Ampcode"}
                  </div>
                </div>

                {secrets.length > 0 && (
                  <div className="p-4 bg-muted rounded-lg">
                    <h4 className="font-medium mb-2">Secrets ({secrets.length})</h4>
                    <div className="space-y-1">
                      {secrets.map((s, i) => (
                        <div key={i} className="text-sm flex items-center gap-2">
                          <Key className="h-3 w-3" />
                          {s.name}
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {prepSteps.length > 0 && (
                  <div className="p-4 bg-muted rounded-lg">
                    <h4 className="font-medium mb-2">
                      Preparation Steps ({prepSteps.length})
                    </h4>
                    <div className="space-y-1">
                      {prepSteps.map((s, i) => (
                        <div key={i} className="text-sm flex items-center gap-2">
                          {s.type === "git_clone" ? (
                            <GitBranch className="h-3 w-3" />
                          ) : (
                            <FileCode className="h-3 w-3" />
                          )}
                          {s.type === "git_clone"
                            ? (s.config as { url: string }).url
                            : "Custom script"}
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Navigation Buttons */}
      <div className="flex justify-between">
        <Button
          variant="outline"
          onClick={() => setCurrentStep(currentStep - 1)}
          disabled={currentStep === 1}
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Previous
        </Button>

        {currentStep < 5 ? (
          <Button
            onClick={() => setCurrentStep(currentStep + 1)}
            disabled={!canProceed()}
          >
            Next
            <ArrowRight className="h-4 w-4 ml-2" />
          </Button>
        ) : (
          <Button onClick={handleSubmit} disabled={loading || !canProceed()}>
            {loading ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Creating...
              </>
            ) : (
              <>
                <Sparkles className="h-4 w-4 mr-2" />
                Create Workspace
              </>
            )}
          </Button>
        )}
      </div>
    </div>
  );
}
