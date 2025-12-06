"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Header } from "@/components/header";
import { api } from "@/lib/api";
import { toast } from "sonner";
import { RefreshCw, CheckCircle, XCircle } from "lucide-react";

export default function SettingsPage() {
  const [apiUrl, setApiUrl] = useState("");
  const [healthStatus, setHealthStatus] = useState<"checking" | "healthy" | "unhealthy">("checking");

  useEffect(() => {
    // Get the current API URL from env or default
    setApiUrl(process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080");
    checkHealth();
  }, []);

  const checkHealth = async () => {
    setHealthStatus("checking");
    try {
      const res = await api.health();
      if (res.data?.status === "ok" || res.data?.status === "healthy") {
        setHealthStatus("healthy");
      } else {
        setHealthStatus("unhealthy");
      }
    } catch {
      setHealthStatus("unhealthy");
    }
  };

  return (
    <div className="flex flex-col h-full">
      <Header />
      <div className="flex-1 p-6 space-y-6">
        <div className="max-w-2xl space-y-6">
          {/* API Connection */}
          <Card>
            <CardHeader>
              <CardTitle>API Connection</CardTitle>
              <CardDescription>
                Configure the connection to the Aetherium API Gateway
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">API URL</label>
                <div className="flex gap-2">
                  <Input
                    value={apiUrl}
                    onChange={(e) => setApiUrl(e.target.value)}
                    placeholder="http://localhost:8080"
                  />
                  <Button variant="outline" onClick={checkHealth}>
                    <RefreshCw className="h-4 w-4" />
                  </Button>
                </div>
                <p className="text-xs text-muted-foreground">
                  Set the NEXT_PUBLIC_API_URL environment variable to change this
                </p>
              </div>

              <div className="flex items-center gap-2">
                <span className="text-sm">Status:</span>
                {healthStatus === "checking" ? (
                  <span className="text-muted-foreground">Checking...</span>
                ) : healthStatus === "healthy" ? (
                  <span className="flex items-center gap-1 text-green-600">
                    <CheckCircle className="h-4 w-4" />
                    Connected
                  </span>
                ) : (
                  <span className="flex items-center gap-1 text-red-600">
                    <XCircle className="h-4 w-4" />
                    Disconnected
                  </span>
                )}
              </div>
            </CardContent>
          </Card>

          {/* About */}
          <Card>
            <CardHeader>
              <CardTitle>About Aetherium</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Aetherium is a distributed task execution platform that runs isolated
                workloads in Firecracker microVMs or Docker containers.
              </p>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-muted-foreground">Version:</span>{" "}
                  <span className="font-mono">1.0.0</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Dashboard:</span>{" "}
                  <span className="font-mono">Next.js 15</span>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Features */}
          <Card>
            <CardHeader>
              <CardTitle>Features</CardTitle>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li>Async task queue with Redis/Asynq</li>
                <li>VM lifecycle management (create, execute, delete)</li>
                <li>Command execution via vsock (host-VM communication)</li>
                <li>PostgreSQL state persistence</li>
                <li>Tool auto-installation (nodejs, bun, go, python, etc.)</li>
                <li>REST API Gateway</li>
                <li>Kubernetes deployment with Helm</li>
              </ul>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
