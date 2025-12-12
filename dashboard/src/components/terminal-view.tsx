"use client";

import { useEffect, useRef, useCallback, useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Copy, Check, Terminal as TerminalIcon, Maximize2, Minimize2 } from "lucide-react";

// Dynamic import for ghostty-web to avoid SSR issues
let ghosttyInit: (() => Promise<void>) | null = null;
let GhosttyTerminal: (new (options?: TerminalOptions) => TerminalInstance) | null = null;
let GhosttyFitAddon: (new () => FitAddonInstance) | null = null;

interface TerminalOptions {
  fontSize?: number;
  fontFamily?: string;
  theme?: {
    background?: string;
    foreground?: string;
    cursor?: string;
    cursorAccent?: string;
    selectionBackground?: string;
    black?: string;
    red?: string;
    green?: string;
    yellow?: string;
    blue?: string;
    magenta?: string;
    cyan?: string;
    white?: string;
    brightBlack?: string;
    brightRed?: string;
    brightGreen?: string;
    brightYellow?: string;
    brightBlue?: string;
    brightMagenta?: string;
    brightCyan?: string;
    brightWhite?: string;
  };
  scrollback?: number;
  cursorBlink?: boolean;
  cursorStyle?: "block" | "underline" | "bar";
}

interface TerminalInstance {
  open(element: HTMLElement): void;
  write(data: string): void;
  writeln(data: string): void;
  clear(): void;
  reset(): void;
  dispose(): void;
  onData(callback: (data: string) => void): { dispose(): void };
  focus(): void;
  blur(): void;
  resize?(cols: number, rows: number): void;
  loadAddon?(addon: FitAddonInstance): void;
}

interface FitAddonInstance {
  fit(): void;
  dispose(): void;
}

const defaultTheme = {
  background: "#1a1b26",
  foreground: "#c0caf5",
  cursor: "#c0caf5",
  cursorAccent: "#1a1b26",
  selectionBackground: "#33467c",
  black: "#15161e",
  red: "#f7768e",
  green: "#9ece6a",
  yellow: "#e0af68",
  blue: "#7aa2f7",
  magenta: "#bb9af7",
  cyan: "#7dcfff",
  white: "#a9b1d6",
  brightBlack: "#414868",
  brightRed: "#f7768e",
  brightGreen: "#9ece6a",
  brightYellow: "#e0af68",
  brightBlue: "#7aa2f7",
  brightMagenta: "#bb9af7",
  brightCyan: "#7dcfff",
  brightWhite: "#c0caf5",
};

interface TerminalViewProps {
  output?: string;
  className?: string;
  height?: string;
  readOnly?: boolean;
  onInput?: (data: string) => void;
  title?: string;
  streamingMode?: boolean; // When true, output is treated as append-only chunks
  onClear?: () => void; // Callback when terminal is cleared
}

// Imperative handle for controlling the terminal
export interface TerminalViewHandle {
  clear: () => void;
  write: (data: string) => void;
}

export function TerminalView({
  output = "",
  className = "",
  height = "400px",
  readOnly = true,
  onInput,
  title = "Terminal Output",
  streamingMode = false,
  onClear,
}: TerminalViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const terminalRef = useRef<TerminalInstance | null>(null);
  const [isInitialized, setIsInitialized] = useState(false);
  const [isTerminalReady, setIsTerminalReady] = useState(false);
  const [initError, setInitError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const lastOutputRef = useRef<string>("");

  // Clear terminal function
  const clearTerminal = useCallback(() => {
    if (terminalRef.current) {
      terminalRef.current.clear();
      lastOutputRef.current = "";
      onClear?.();
    }
  }, [onClear]);

  // Initialize ghostty-web
  useEffect(() => {
    let mounted = true;

    async function initGhostty() {
      try {
        // Dynamic import to avoid SSR issues
        const ghostty = await import("ghostty-web");

        if (!mounted) return;

        ghosttyInit = ghostty.init;
        GhosttyTerminal = ghostty.Terminal;
        GhosttyFitAddon = ghostty.FitAddon;

        // Initialize WASM
        await ghosttyInit();

        if (!mounted) return;
        setIsInitialized(true);
      } catch (err) {
        console.error("Failed to initialize ghostty-web:", err);
        if (mounted) {
          setInitError(err instanceof Error ? err.message : "Failed to load terminal");
        }
      }
    }

    initGhostty();

    return () => {
      mounted = false;
    };
  }, []);

  // Create terminal instance
  useEffect(() => {
    if (!isInitialized || !containerRef.current || !GhosttyTerminal) return;

    // Create terminal
    const term = new GhosttyTerminal({
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
      theme: defaultTheme,
      scrollback: 10000,
      cursorBlink: !readOnly,
      cursorStyle: "block",
    });

    // Create and load FitAddon for proper sizing
    let fitAddon: FitAddonInstance | null = null;
    if (GhosttyFitAddon && term.loadAddon) {
      fitAddon = new GhosttyFitAddon();
      term.loadAddon(fitAddon);
    }

    terminalRef.current = term;

    // Open in container
    term.open(containerRef.current);

    // Fit terminal to container size and mark as ready
    if (fitAddon) {
      // Small delay to ensure container is sized
      setTimeout(() => {
        fitAddon?.fit();
        setIsTerminalReady(true);
      }, 100);
    } else {
      // No FitAddon, mark as ready immediately
      setTimeout(() => setIsTerminalReady(true), 100);
    }

    // Handle input if not read-only
    if (!readOnly && onInput) {
      const disposable = term.onData((data) => {
        onInput(data);
      });

      return () => {
        disposable.dispose();
        fitAddon?.dispose();
        term.dispose();
        terminalRef.current = null;
        setIsTerminalReady(false);
      };
    }

    return () => {
      fitAddon?.dispose();
      term.dispose();
      terminalRef.current = null;
      setIsTerminalReady(false);
    };
  }, [isInitialized, readOnly, onInput]);

  // Write output to terminal
  useEffect(() => {
    if (!isTerminalReady || !terminalRef.current) {
      return;
    }

    // In streaming mode, output is already the new delta chunk - just append it
    if (streamingMode) {
      if (output && output !== lastOutputRef.current) {
        // Write the new output directly (it's a delta chunk)
        terminalRef.current.write(output);
        lastOutputRef.current = output;
      }
      return;
    }

    // Non-streaming mode: handle cumulative output with diffing
    if (!output) return;

    // Check if output changed
    if (output !== lastOutputRef.current) {
      console.log("[TerminalView] Writing output:", output.substring(0, 100) + (output.length > 100 ? "..." : ""));

      // Calculate the new content to write
      if (output.startsWith(lastOutputRef.current) && lastOutputRef.current.length > 0) {
        // Incremental update - just write the new part
        const newContent = output.slice(lastOutputRef.current.length);
        if (newContent) {
          terminalRef.current.write(newContent);
        }
      } else {
        // Full replace - clear and write everything
        terminalRef.current.clear();
        // Ensure output ends with newline for proper display
        const formattedOutput = output.endsWith('\n') ? output : output + '\r\n';
        terminalRef.current.write(formattedOutput);
      }
      lastOutputRef.current = output;
    }
  }, [output, isTerminalReady, streamingMode]);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(output);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  }, [output]);

  const toggleFullscreen = useCallback(() => {
    setIsFullscreen((prev) => !prev);
  }, []);

  // Fallback for initialization error
  if (initError) {
    return (
      <div className={`border rounded-lg overflow-hidden ${className}`}>
        <div className="flex items-center justify-between px-3 py-2 bg-muted border-b">
          <div className="flex items-center gap-2">
            <TerminalIcon className="h-4 w-4" />
            <span className="text-sm font-medium">{title}</span>
          </div>
        </div>
        <div
          className="bg-[#1a1b26] text-red-400 p-4 font-mono text-sm"
          style={{ height }}
        >
          <p>Failed to initialize terminal: {initError}</p>
          <p className="mt-2 text-gray-400">Falling back to plain text view:</p>
          <pre className="mt-2 whitespace-pre-wrap text-gray-300">{output}</pre>
        </div>
      </div>
    );
  }

  // Loading state
  if (!isInitialized) {
    return (
      <div className={`border rounded-lg overflow-hidden ${className}`}>
        <div className="flex items-center justify-between px-3 py-2 bg-muted border-b">
          <div className="flex items-center gap-2">
            <TerminalIcon className="h-4 w-4" />
            <span className="text-sm font-medium">{title}</span>
          </div>
        </div>
        <div
          className="bg-[#1a1b26] flex items-center justify-center"
          style={{ height }}
        >
          <div className="flex items-center gap-2 text-gray-400">
            <div className="animate-spin h-5 w-5 border-2 border-gray-400 border-t-transparent rounded-full" />
            <span>Loading terminal...</span>
          </div>
        </div>
      </div>
    );
  }

  const containerStyle = isFullscreen
    ? {
        position: "fixed" as const,
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        zIndex: 50,
        height: "100vh",
      }
    : { height };

  return (
    <div
      className={`border rounded-lg overflow-hidden ${className} ${
        isFullscreen ? "rounded-none" : ""
      }`}
      style={isFullscreen ? { position: "fixed", inset: 0, zIndex: 50 } : undefined}
    >
      <div className="flex items-center justify-between px-3 py-2 bg-muted border-b">
        <div className="flex items-center gap-2">
          <TerminalIcon className="h-4 w-4" />
          <span className="text-sm font-medium">{title}</span>
          {readOnly && (
            <Badge variant="outline" className="text-xs">
              Read-only
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleCopy}
            className="h-7 px-2"
            disabled={!output}
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
          <Button
            variant="ghost"
            size="sm"
            onClick={toggleFullscreen}
            className="h-7 px-2"
          >
            {isFullscreen ? (
              <Minimize2 className="h-3 w-3" />
            ) : (
              <Maximize2 className="h-3 w-3" />
            )}
          </Button>
        </div>
      </div>
      <div
        ref={containerRef}
        className="bg-[#1a1b26]"
        style={containerStyle}
      />
    </div>
  );
}

// Simple text-based fallback component
interface TextTerminalProps {
  output: string;
  className?: string;
  height?: string;
  title?: string;
}

export function TextTerminal({
  output,
  className = "",
  height = "400px",
  title = "Output",
}: TextTerminalProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(output);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  }, [output]);

  return (
    <div className={`border rounded-lg overflow-hidden ${className}`}>
      <div className="flex items-center justify-between px-3 py-2 bg-muted border-b">
        <div className="flex items-center gap-2">
          <TerminalIcon className="h-4 w-4" />
          <span className="text-sm font-medium">{title}</span>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleCopy}
          className="h-7 px-2"
          disabled={!output}
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
      </div>
      <div
        className="bg-[#1a1b26] overflow-auto p-3"
        style={{ height }}
      >
        <pre className="text-sm font-mono text-[#c0caf5] whitespace-pre-wrap">
          {output || <span className="text-gray-500">No output yet...</span>}
        </pre>
      </div>
    </div>
  );
}
