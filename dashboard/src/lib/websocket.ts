// WebSocket client for real-time streaming from Aetherium workspaces

export type MessageType =
  | "prompt"
  | "response"
  | "stream"
  | "error"
  | "status"
  | "ping"
  | "pong";

export interface IncomingMessage {
  type: MessageType;
  session_id?: string;
  message_id?: string;
  content?: string;
  chunk?: string; // For streaming: delta content only
  exit_code?: number | null;
  error?: string;
  timestamp: string;
}

export interface OutgoingMessage {
  type: "prompt" | "ping";
  prompt?: string;
  system_prompt?: string;
  working_directory?: string;
  environment?: Record<string, unknown>;
}

export type StreamHandler = (chunk: string) => void;
export type CompleteHandler = (exitCode: number | null, error?: string) => void;
export type ErrorHandler = (error: string) => void;
export type StatusHandler = (message: string) => void;

interface WorkspaceWebSocketOptions {
  onStream?: StreamHandler;
  onComplete?: CompleteHandler;
  onError?: ErrorHandler;
  onStatus?: StatusHandler;
  onOpen?: () => void;
  onClose?: () => void;
}

export class WorkspaceWebSocket {
  private ws: WebSocket | null = null;
  private url: string;
  private options: WorkspaceWebSocketOptions;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private pingInterval: ReturnType<typeof setInterval> | null = null;

  constructor(workspaceId: string, options: WorkspaceWebSocketOptions = {}) {
    const wsBase =
      process.env.NEXT_PUBLIC_WS_URL ||
      (typeof window !== "undefined"
        ? window.location.origin.replace(/^http/, "ws")
        : "ws://localhost:8080");
    this.url = `${wsBase}/api/v1/workspaces/${workspaceId}/ws`;
    this.options = options;
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = () => {
        console.log("[WebSocket] Connected to workspace");
        this.reconnectAttempts = 0;
        this.startPingInterval();
        this.options.onOpen?.();
      };

      this.ws.onmessage = (event) => {
        this.handleMessage(event.data);
      };

      this.ws.onclose = (event) => {
        console.log("[WebSocket] Connection closed:", event.code, event.reason);
        this.stopPingInterval();
        this.options.onClose?.();

        // Attempt reconnection if not a clean close
        if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
          this.reconnectAttempts++;
          console.log(
            `[WebSocket] Reconnecting in ${this.reconnectDelay}ms (attempt ${this.reconnectAttempts})`
          );
          setTimeout(() => this.connect(), this.reconnectDelay);
          this.reconnectDelay *= 2; // Exponential backoff
        }
      };

      this.ws.onerror = (error) => {
        console.error("[WebSocket] Error:", error);
        this.options.onError?.("WebSocket connection error");
      };
    } catch (error) {
      console.error("[WebSocket] Failed to connect:", error);
      this.options.onError?.(
        error instanceof Error ? error.message : "Failed to connect"
      );
    }
  }

  private handleMessage(data: string): void {
    try {
      const message: IncomingMessage = JSON.parse(data);
      console.log("[WebSocket] Received:", message.type, message.content?.substring(0, 50) || message.chunk?.substring(0, 50) || "");

      switch (message.type) {
        case "stream":
          // Delta content - append only
          if (message.chunk) {
            this.options.onStream?.(message.chunk);
          }
          break;

        case "response":
          // Legacy cumulative content - for backward compatibility
          // If we have a chunk, use that as delta; otherwise use content
          if (message.chunk) {
            this.options.onStream?.(message.chunk);
          }
          // Check for completion
          if (message.exit_code !== undefined && message.exit_code !== null) {
            this.options.onComplete?.(message.exit_code, message.error);
          }
          break;

        case "status":
          this.options.onStatus?.(message.content || "");
          break;

        case "error":
          this.options.onError?.(message.error || "Unknown error");
          if (message.exit_code !== undefined && message.exit_code !== null) {
            this.options.onComplete?.(message.exit_code, message.error);
          }
          break;

        case "pong":
          // Ping response, ignore
          break;

        default:
          console.log("[WebSocket] Unhandled message type:", message.type);
      }
    } catch (error) {
      console.error("[WebSocket] Failed to parse message:", error, data);
    }
  }

  sendPrompt(
    prompt: string,
    systemPrompt?: string,
    workingDirectory?: string
  ): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      this.options.onError?.("WebSocket is not connected");
      return;
    }

    const message: OutgoingMessage = {
      type: "prompt",
      prompt,
      system_prompt: systemPrompt,
      working_directory: workingDirectory,
    };

    this.ws.send(JSON.stringify(message));
    console.log("[WebSocket] Sent prompt:", prompt.substring(0, 50));
  }

  private startPingInterval(): void {
    this.pingInterval = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: "ping" }));
      }
    }, 30000); // Ping every 30 seconds
  }

  private stopPingInterval(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  disconnect(): void {
    this.stopPingInterval();
    if (this.ws) {
      this.ws.close(1000, "Client disconnecting");
      this.ws = null;
    }
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

// React hook for using WorkspaceWebSocket
import { useEffect, useRef, useState, useCallback } from "react";

export interface UseWorkspaceStreamOptions {
  workspaceId: string;
  autoConnect?: boolean;
}

export interface UseWorkspaceStreamResult {
  output: string;
  isConnected: boolean;
  isStreaming: boolean;
  exitCode: number | null;
  error: string | null;
  status: string | null;
  connect: () => void;
  disconnect: () => void;
  sendPrompt: (prompt: string, systemPrompt?: string, workingDirectory?: string) => void;
  clearOutput: () => void;
}

export function useWorkspaceStream({
  workspaceId,
  autoConnect = false,
}: UseWorkspaceStreamOptions): UseWorkspaceStreamResult {
  const wsRef = useRef<WorkspaceWebSocket | null>(null);
  const [output, setOutput] = useState("");
  const [isConnected, setIsConnected] = useState(false);
  const [isStreaming, setIsStreaming] = useState(false);
  const [exitCode, setExitCode] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);

  const connect = useCallback(() => {
    if (wsRef.current?.isConnected()) return;

    wsRef.current = new WorkspaceWebSocket(workspaceId, {
      onStream: (chunk) => {
        setOutput((prev) => prev + chunk);
        setIsStreaming(true);
      },
      onComplete: (code, err) => {
        setExitCode(code);
        if (err) setError(err);
        setIsStreaming(false);
      },
      onError: (err) => {
        setError(err);
        setIsStreaming(false);
      },
      onStatus: (msg) => {
        setStatus(msg);
      },
      onOpen: () => {
        setIsConnected(true);
        setError(null);
      },
      onClose: () => {
        setIsConnected(false);
        setIsStreaming(false);
      },
    });

    wsRef.current.connect();
  }, [workspaceId]);

  const disconnect = useCallback(() => {
    wsRef.current?.disconnect();
    wsRef.current = null;
    setIsConnected(false);
    setIsStreaming(false);
  }, []);

  const sendPrompt = useCallback(
    (prompt: string, systemPrompt?: string, workingDirectory?: string) => {
      // Clear previous output when starting new prompt
      setOutput("");
      setExitCode(null);
      setError(null);
      setStatus(null);
      setIsStreaming(true);

      wsRef.current?.sendPrompt(prompt, systemPrompt, workingDirectory);
    },
    []
  );

  const clearOutput = useCallback(() => {
    setOutput("");
    setExitCode(null);
    setError(null);
    setStatus(null);
  }, []);

  // Auto-connect if requested
  useEffect(() => {
    if (autoConnect) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [autoConnect, connect, disconnect]);

  return {
    output,
    isConnected,
    isStreaming,
    exitCode,
    error,
    status,
    connect,
    disconnect,
    sendPrompt,
    clearOutput,
  };
}
