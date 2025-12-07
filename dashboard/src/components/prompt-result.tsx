"use client";

import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  CheckCircle2,
  XCircle,
  Clock,
  Loader2,
  ChevronDown,
  ChevronRight,
  Copy,
  Check,
} from "lucide-react";
import type { PromptTask } from "@/lib/types";

interface PromptResultProps {
  prompt: PromptTask;
  defaultExpanded?: boolean;
}

function getStatusBadge(status: string) {
  switch (status.toLowerCase()) {
    case "completed":
      return <Badge className="bg-green-500">Completed</Badge>;
    case "running":
      return (
        <Badge className="bg-blue-500">
          <Loader2 className="h-3 w-3 mr-1 animate-spin" />
          Running
        </Badge>
      );
    case "pending":
      return <Badge className="bg-yellow-500">Pending</Badge>;
    case "failed":
      return <Badge variant="destructive">Failed</Badge>;
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
}

function getStatusIcon(status: string) {
  switch (status.toLowerCase()) {
    case "completed":
      return <CheckCircle2 className="h-4 w-4 text-green-500" />;
    case "running":
      return <Loader2 className="h-4 w-4 text-blue-500 animate-spin" />;
    case "pending":
      return <Clock className="h-4 w-4 text-yellow-500" />;
    case "failed":
      return <XCircle className="h-4 w-4 text-red-500" />;
    default:
      return <Clock className="h-4 w-4 text-muted-foreground" />;
  }
}

interface CopyButtonProps {
  text: string;
  className?: string;
}

function CopyButton({ text, className = "" }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  }, [text]);

  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={handleCopy}
      className={`h-6 px-2 ${className}`}
    >
      {copied ? (
        <>
          <Check className="h-3 w-3 mr-1" />
          Copied
        </>
      ) : (
        <>
          <Copy className="h-3 w-3 mr-1" />
          Copy
        </>
      )}
    </Button>
  );
}

interface OutputBlockProps {
  label: string;
  content: string;
  variant: "stdout" | "stderr" | "error" | "prompt";
  maxHeight?: string;
}

function OutputBlock({
  label,
  content,
  variant,
  maxHeight = "384px",
}: OutputBlockProps) {
  const bgColors = {
    stdout: "bg-green-50 dark:bg-green-950 border-green-200 dark:border-green-800",
    stderr: "bg-red-50 dark:bg-red-950 border-red-200 dark:border-red-800",
    error: "bg-red-50 dark:bg-red-950 border-red-200 dark:border-red-800",
    prompt: "bg-muted border-border",
  };

  const labelColors = {
    stdout: "text-green-600 dark:text-green-400",
    stderr: "text-red-600 dark:text-red-400",
    error: "text-red-600 dark:text-red-400",
    prompt: "text-muted-foreground",
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-1">
        <Label className={`text-xs ${labelColors[variant]}`}>{label}</Label>
        <CopyButton text={content} />
      </div>
      <ScrollArea className={`rounded border ${bgColors[variant]}`} style={{ maxHeight }}>
        <pre className="p-3 text-sm whitespace-pre-wrap font-mono overflow-x-auto">
          {content}
        </pre>
      </ScrollArea>
    </div>
  );
}

export function PromptResult({ prompt, defaultExpanded = false }: PromptResultProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    const minutes = Math.floor(ms / 60000);
    const seconds = ((ms % 60000) / 1000).toFixed(0);
    return `${minutes}m ${seconds}s`;
  };

  const hasOutput = prompt.stdout || prompt.stderr || prompt.error;
  const isRunning = prompt.status.toLowerCase() === "running";
  const isPending = prompt.status.toLowerCase() === "pending";

  return (
    <Collapsible open={expanded} onOpenChange={setExpanded}>
      <div className="border rounded-lg p-4 transition-colors hover:bg-muted/50">
        <CollapsibleTrigger asChild>
          <div className="flex items-start justify-between cursor-pointer">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-2 flex-wrap">
                {getStatusIcon(prompt.status)}
                {getStatusBadge(prompt.status)}
                {prompt.exit_code !== undefined && prompt.exit_code !== null && (
                  <Badge
                    variant={prompt.exit_code === 0 ? "outline" : "destructive"}
                    className="font-mono"
                  >
                    Exit: {prompt.exit_code}
                  </Badge>
                )}
                <span className="text-xs text-muted-foreground">
                  {new Date(prompt.created_at).toLocaleString()}
                </span>
                {prompt.duration_ms && (
                  <span className="text-xs text-muted-foreground">
                    ({formatDuration(prompt.duration_ms)})
                  </span>
                )}
              </div>
              <p className="text-sm line-clamp-2 text-muted-foreground font-mono">
                {prompt.prompt}
              </p>
            </div>
            <Button variant="ghost" size="sm" className="ml-2 shrink-0">
              {expanded ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
            </Button>
          </div>
        </CollapsibleTrigger>

        <CollapsibleContent className="pt-4 space-y-4">
          <OutputBlock
            label="Full Prompt"
            content={prompt.prompt}
            variant="prompt"
            maxHeight="200px"
          />

          {prompt.system_prompt && (
            <OutputBlock
              label="System Prompt"
              content={prompt.system_prompt}
              variant="prompt"
              maxHeight="150px"
            />
          )}

          {prompt.working_directory && (
            <div>
              <Label className="text-xs text-muted-foreground">
                Working Directory
              </Label>
              <code className="mt-1 block text-sm bg-muted px-2 py-1 rounded font-mono">
                {prompt.working_directory}
              </code>
            </div>
          )}

          {isRunning && (
            <div className="flex items-center gap-2 p-4 bg-blue-50 dark:bg-blue-950 rounded-lg border border-blue-200 dark:border-blue-800">
              <Loader2 className="h-5 w-5 animate-spin text-blue-500" />
              <span className="text-sm text-blue-700 dark:text-blue-300">
                Prompt is currently being executed...
              </span>
            </div>
          )}

          {isPending && (
            <div className="flex items-center gap-2 p-4 bg-yellow-50 dark:bg-yellow-950 rounded-lg border border-yellow-200 dark:border-yellow-800">
              <Clock className="h-5 w-5 text-yellow-500" />
              <span className="text-sm text-yellow-700 dark:text-yellow-300">
                Prompt is queued and waiting to be processed...
              </span>
            </div>
          )}

          {prompt.stdout && (
            <OutputBlock
              label="Output (stdout)"
              content={prompt.stdout}
              variant="stdout"
            />
          )}

          {prompt.stderr && (
            <OutputBlock
              label="Errors (stderr)"
              content={prompt.stderr}
              variant="stderr"
            />
          )}

          {prompt.error && (
            <OutputBlock
              label="Error"
              content={prompt.error}
              variant="error"
              maxHeight="200px"
            />
          )}

          {!hasOutput && !isRunning && !isPending && (
            <div className="text-center py-4 text-muted-foreground text-sm">
              No output recorded
            </div>
          )}
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}

interface PromptListProps {
  prompts: PromptTask[];
  emptyMessage?: string;
  aiAssistantName?: string;
}

export function PromptList({
  prompts,
  emptyMessage = "No prompts submitted yet",
  aiAssistantName = "AI Assistant",
}: PromptListProps) {
  if (prompts.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        <Clock className="h-12 w-12 mx-auto mb-4 opacity-50" />
        <p>{emptyMessage}</p>
        <p className="text-sm mt-1">
          Submit your first prompt to start working with {aiAssistantName}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {prompts.map((prompt, index) => (
        <PromptResult
          key={prompt.id}
          prompt={prompt}
          // Auto-expand the first (most recent) prompt if it's running
          defaultExpanded={index === 0 && prompt.status.toLowerCase() === "running"}
        />
      ))}
    </div>
  );
}
