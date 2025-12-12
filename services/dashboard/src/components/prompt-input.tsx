"use client";

import { useRef, useState, useCallback, KeyboardEvent } from "react";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Send,
  Loader2,
  ChevronDown,
  ChevronRight,
  Keyboard,
} from "lucide-react";

interface PromptInputProps {
  onSubmit: (prompt: string, systemPrompt?: string, workingDirectory?: string) => Promise<void>;
  disabled?: boolean;
  defaultWorkingDirectory?: string;
  placeholder?: string;
  aiAssistantName?: string;
}

export function PromptInput({
  onSubmit,
  disabled = false,
  defaultWorkingDirectory = "/workspace",
  placeholder = "Enter your prompt here...",
  aiAssistantName = "AI Assistant",
}: PromptInputProps) {
  const [promptText, setPromptText] = useState("");
  const [systemPrompt, setSystemPrompt] = useState("");
  const [workingDirectory, setWorkingDirectory] = useState(defaultWorkingDirectory);
  const [submitting, setSubmitting] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);

  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const characterCount = promptText.length;
  const maxCharacters = 100000; // 100k character limit

  const handleSubmit = useCallback(async () => {
    if (!promptText.trim() || submitting || disabled) return;

    setSubmitting(true);
    try {
      await onSubmit(
        promptText,
        systemPrompt || undefined,
        workingDirectory || undefined
      );
      setPromptText("");
      // Keep system prompt and working directory for repeat submissions
    } finally {
      setSubmitting(false);
    }
  }, [promptText, systemPrompt, workingDirectory, submitting, disabled, onSubmit]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      // Ctrl+Enter or Cmd+Enter to submit
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
        e.preventDefault();
        handleSubmit();
      }
    },
    [handleSubmit]
  );

  return (
    <div className="space-y-4">
      <div className="relative">
        <Label htmlFor="prompt" className="sr-only">
          Prompt
        </Label>
        <Textarea
          ref={textareaRef}
          id="prompt"
          placeholder={placeholder}
          value={promptText}
          onChange={(e) => setPromptText(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={disabled || submitting}
          className="min-h-[140px] resize-y pr-4 pb-8 font-mono text-sm"
          maxLength={maxCharacters}
        />
        <div className="absolute bottom-2 left-3 flex items-center gap-2 text-xs text-muted-foreground">
          <span
            className={
              characterCount > maxCharacters * 0.9
                ? "text-orange-500"
                : characterCount > maxCharacters * 0.95
                ? "text-red-500"
                : ""
            }
          >
            {characterCount.toLocaleString()} / {maxCharacters.toLocaleString()}
          </span>
        </div>
        <div className="absolute bottom-2 right-3 flex items-center gap-1 text-xs text-muted-foreground">
          <Keyboard className="h-3 w-3" />
          <span>Ctrl+Enter to submit</span>
        </div>
      </div>

      <Collapsible open={showAdvanced} onOpenChange={setShowAdvanced}>
        <CollapsibleTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            className="gap-2 text-muted-foreground"
            type="button"
          >
            {showAdvanced ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            Advanced Options
          </Button>
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-4 pt-4">
          <div>
            <Label htmlFor="systemPrompt">System Prompt (Optional)</Label>
            <Textarea
              id="systemPrompt"
              placeholder="Enter a system prompt to customize behavior..."
              value={systemPrompt}
              onChange={(e) => setSystemPrompt(e.target.value)}
              disabled={disabled || submitting}
              className="min-h-[80px] mt-1 font-mono text-sm"
            />
            <p className="text-xs text-muted-foreground mt-1">
              Provide instructions for how {aiAssistantName} should behave
            </p>
          </div>
          <div>
            <Label htmlFor="workingDir">Working Directory</Label>
            <Input
              id="workingDir"
              placeholder="/workspace"
              value={workingDirectory}
              onChange={(e) => setWorkingDirectory(e.target.value)}
              disabled={disabled || submitting}
              className="mt-1 font-mono"
            />
            <p className="text-xs text-muted-foreground mt-1">
              Directory where the prompt will be executed
            </p>
          </div>
        </CollapsibleContent>
      </Collapsible>

      <div className="flex justify-between items-center">
        <div className="text-xs text-muted-foreground">
          {disabled
            ? "Workspace is not ready yet"
            : `Send a prompt to ${aiAssistantName}`}
        </div>
        <Button
          onClick={handleSubmit}
          disabled={disabled || submitting || !promptText.trim()}
        >
          {submitting ? (
            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
          ) : (
            <Send className="h-4 w-4 mr-2" />
          )}
          {submitting ? "Submitting..." : "Submit Prompt"}
        </Button>
      </div>
    </div>
  );
}
