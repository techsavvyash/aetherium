"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Header } from "@/components/header";
import { api } from "@/lib/api";
import { toast } from "sonner";
import { Send, Copy, Clock } from "lucide-react";

interface RequestHistory {
  id: string;
  method: string;
  path: string;
  status: number;
  timestamp: Date;
  duration: number;
}

const presetRequests = [
  {
    name: "Health Check",
    method: "GET",
    path: "/api/v1/health",
    body: "",
  },
  {
    name: "List VMs",
    method: "GET",
    path: "/api/v1/vms",
    body: "",
  },
  {
    name: "Create VM",
    method: "POST",
    path: "/api/v1/vms",
    body: JSON.stringify(
      {
        name: "test-vm",
        vcpus: 2,
        memory_mb: 512,
        additional_tools: ["go", "python"],
      },
      null,
      2
    ),
  },
  {
    name: "Execute Command",
    method: "POST",
    path: "/api/v1/vms/{vm_id}/execute",
    body: JSON.stringify(
      {
        command: "ls",
        args: ["-la", "/home"],
      },
      null,
      2
    ),
  },
  {
    name: "List Tasks",
    method: "GET",
    path: "/api/v1/tasks",
    body: "",
  },
  {
    name: "List Workers",
    method: "GET",
    path: "/api/v1/workers",
    body: "",
  },
  {
    name: "Queue Stats",
    method: "GET",
    path: "/api/v1/queue/stats",
    body: "",
  },
];

export default function ApiExplorerPage() {
  const [method, setMethod] = useState("GET");
  const [path, setPath] = useState("/api/v1/health");
  const [body, setBody] = useState("");
  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState<{
    status: number;
    data: unknown;
    headers: Record<string, string>;
    duration: number;
  } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [history, setHistory] = useState<RequestHistory[]>([]);

  const handleSend = async () => {
    setLoading(true);
    setError(null);
    const startTime = Date.now();

    try {
      const result = await api.customRequest(
        method,
        path,
        method !== "GET" && method !== "DELETE" && body ? body : undefined
      );

      const duration = Date.now() - startTime;

      setResponse({
        ...result,
        duration,
      });

      setHistory((prev) => [
        {
          id: crypto.randomUUID(),
          method,
          path,
          status: result.status,
          timestamp: new Date(),
          duration,
        },
        ...prev.slice(0, 19),
      ]);
    } catch (err) {
      const duration = Date.now() - startTime;
      setError(err instanceof Error ? err.message : "Request failed");
      setHistory((prev) => [
        {
          id: crypto.randomUUID(),
          method,
          path,
          status: 0,
          timestamp: new Date(),
          duration,
        },
        ...prev.slice(0, 19),
      ]);
    } finally {
      setLoading(false);
    }
  };

  const loadPreset = (preset: (typeof presetRequests)[0]) => {
    setMethod(preset.method);
    setPath(preset.path);
    setBody(preset.body);
    setResponse(null);
    setError(null);
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success("Copied to clipboard");
  };

  const getStatusColor = (status: number) => {
    if (status >= 200 && status < 300) return "default";
    if (status >= 400 && status < 500) return "secondary";
    if (status >= 500) return "destructive";
    return "outline";
  };

  return (
    <div className="flex flex-col h-full">
      <Header />
      <div className="flex-1 p-6 overflow-auto">
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* Sidebar - Presets and History */}
          <div className="lg:col-span-1 space-y-6">
            {/* Presets */}
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Quick Actions</CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <div className="space-y-1 p-2">
                  {presetRequests.map((preset, index) => (
                    <Button
                      key={index}
                      variant="ghost"
                      className="w-full justify-start text-sm h-8"
                      onClick={() => loadPreset(preset)}
                    >
                      <Badge
                        variant="outline"
                        className="mr-2 text-xs font-mono"
                      >
                        {preset.method}
                      </Badge>
                      {preset.name}
                    </Button>
                  ))}
                </div>
              </CardContent>
            </Card>

            {/* History */}
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">History</CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                {history.length === 0 ? (
                  <p className="text-sm text-muted-foreground p-4">
                    No requests yet
                  </p>
                ) : (
                  <div className="space-y-1 p-2 max-h-64 overflow-auto">
                    {history.map((item) => (
                      <Button
                        key={item.id}
                        variant="ghost"
                        className="w-full justify-start text-xs h-auto py-2"
                        onClick={() => {
                          setMethod(item.method);
                          setPath(item.path);
                        }}
                      >
                        <div className="flex flex-col items-start gap-1 w-full">
                          <div className="flex items-center gap-2">
                            <Badge
                              variant={getStatusColor(item.status)}
                              className="text-xs font-mono"
                            >
                              {item.status || "ERR"}
                            </Badge>
                            <span className="font-mono">{item.method}</span>
                          </div>
                          <span className="text-muted-foreground truncate w-full text-left">
                            {item.path}
                          </span>
                        </div>
                      </Button>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Main Content - Request/Response */}
          <div className="lg:col-span-3 space-y-6">
            {/* Request Builder */}
            <Card>
              <CardHeader>
                <CardTitle>Request</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex gap-2">
                  <Select value={method} onValueChange={setMethod}>
                    <SelectTrigger className="w-28">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="GET">GET</SelectItem>
                      <SelectItem value="POST">POST</SelectItem>
                      <SelectItem value="PUT">PUT</SelectItem>
                      <SelectItem value="PATCH">PATCH</SelectItem>
                      <SelectItem value="DELETE">DELETE</SelectItem>
                    </SelectContent>
                  </Select>
                  <Input
                    value={path}
                    onChange={(e) => setPath(e.target.value)}
                    placeholder="/api/v1/..."
                    className="flex-1 font-mono"
                  />
                  <Button onClick={handleSend} disabled={loading}>
                    <Send className="h-4 w-4 mr-2" />
                    {loading ? "Sending..." : "Send"}
                  </Button>
                </div>

                {(method === "POST" ||
                  method === "PUT" ||
                  method === "PATCH") && (
                  <div className="space-y-2">
                    <label className="text-sm font-medium">Request Body</label>
                    <Textarea
                      value={body}
                      onChange={(e) => setBody(e.target.value)}
                      placeholder='{"key": "value"}'
                      className="font-mono min-h-32"
                    />
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Response */}
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle>Response</CardTitle>
                  {response && (
                    <div className="flex items-center gap-4">
                      <Badge variant={getStatusColor(response.status)}>
                        {response.status}
                      </Badge>
                      <div className="flex items-center gap-1 text-sm text-muted-foreground">
                        <Clock className="h-3 w-3" />
                        {response.duration}ms
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() =>
                          copyToClipboard(JSON.stringify(response.data, null, 2))
                        }
                      >
                        <Copy className="h-4 w-4" />
                      </Button>
                    </div>
                  )}
                </div>
              </CardHeader>
              <CardContent>
                {error ? (
                  <div className="bg-destructive/10 text-destructive p-4 rounded-lg">
                    <pre className="text-sm">{error}</pre>
                  </div>
                ) : response ? (
                  <Tabs defaultValue="body">
                    <TabsList>
                      <TabsTrigger value="body">Body</TabsTrigger>
                      <TabsTrigger value="headers">Headers</TabsTrigger>
                    </TabsList>
                    <TabsContent value="body">
                      <pre className="bg-black text-green-400 p-4 rounded-lg text-sm overflow-auto max-h-96">
                        {typeof response.data === "string"
                          ? response.data
                          : JSON.stringify(response.data, null, 2)}
                      </pre>
                    </TabsContent>
                    <TabsContent value="headers">
                      <pre className="bg-muted p-4 rounded-lg text-sm overflow-auto">
                        {Object.entries(response.headers)
                          .map(([k, v]) => `${k}: ${v}`)
                          .join("\n")}
                      </pre>
                    </TabsContent>
                  </Tabs>
                ) : (
                  <p className="text-muted-foreground text-center py-8">
                    Send a request to see the response
                  </p>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
}
