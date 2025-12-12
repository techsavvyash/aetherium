"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Header } from "@/components/header";
import { api } from "@/lib/api";
import type { VM, QueueStats } from "@/lib/types";
import { Server, Activity, CheckCircle, Clock } from "lucide-react";

export default function DashboardPage() {
  const [vms, setVMs] = useState<VM[]>([]);
  const [queueStats, setQueueStats] = useState<QueueStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [healthStatus, setHealthStatus] = useState<string | null>(null);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [vmsRes, statsRes, healthRes] = await Promise.all([
        api.listVMs(),
        api.getQueueStats(),
        api.health(),
      ]);

      if (vmsRes.data) setVMs(vmsRes.data.vms || []);
      if (statsRes.data) setQueueStats(statsRes.data);
      if (healthRes.data) setHealthStatus(healthRes.data.status);
    } catch (error) {
      console.error("Failed to fetch data:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, []);

  const runningVMs = vms.filter((vm) => vm.status.toLowerCase() === "running").length;
  const stoppedVMs = vms.filter((vm) => vm.status.toLowerCase() === "stopped").length;

  return (
    <div className="flex flex-col h-full">
      <Header onRefresh={fetchData} />
      <div className="flex-1 p-6 space-y-6">
        {/* Health Status */}
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">System Status:</span>
          <Badge variant={healthStatus === "ok" || healthStatus === "healthy" ? "default" : "destructive"}>
            {healthStatus || "Unknown"}
          </Badge>
        </div>

        {/* Stats Cards */}
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total VMs</CardTitle>
              <Server className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{vms.length}</div>
              <p className="text-xs text-muted-foreground">
                {runningVMs} running, {stoppedVMs} stopped
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Active Tasks</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{queueStats?.active || 0}</div>
              <p className="text-xs text-muted-foreground">
                {queueStats?.pending || 0} pending
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Completed</CardTitle>
              <CheckCircle className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{queueStats?.completed || 0}</div>
              <p className="text-xs text-muted-foreground">
                {queueStats?.failed || 0} failed
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Scheduled</CardTitle>
              <Clock className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{queueStats?.scheduled || 0}</div>
              <p className="text-xs text-muted-foreground">
                {queueStats?.retry || 0} retrying
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Recent VMs */}
        <Card>
          <CardHeader>
            <CardTitle>Recent Virtual Machines</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <p className="text-sm text-muted-foreground">Loading...</p>
            ) : vms.length === 0 ? (
              <p className="text-sm text-muted-foreground">No virtual machines found</p>
            ) : (
              <div className="space-y-4">
                {vms.slice(0, 5).map((vm) => (
                  <div
                    key={vm.id}
                    className="flex items-center justify-between rounded-lg border p-4"
                  >
                    <div className="flex items-center gap-4">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                        <Server className="h-5 w-5 text-primary" />
                      </div>
                      <div>
                        <p className="font-medium">{vm.name}</p>
                        <p className="text-sm text-muted-foreground">
                          {vm.vcpu_count} vCPU, {vm.memory_mb} MB
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <Badge
                        variant={
                          vm.status.toLowerCase() === "running"
                            ? "default"
                            : vm.status.toLowerCase() === "stopped"
                            ? "secondary"
                            : "outline"
                        }
                      >
                        {vm.status}
                      </Badge>
                      <span className="text-sm text-muted-foreground">
                        {new Date(vm.created_at).toLocaleDateString()}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
