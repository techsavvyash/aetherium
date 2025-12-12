"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Header } from "@/components/header";
import { api } from "@/lib/api";
import type { Worker, QueueStats } from "@/lib/types";
import { Cpu, Activity, CheckCircle, XCircle, Server, HardDrive } from "lucide-react";

export default function WorkersPage() {
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [queueStats, setQueueStats] = useState<QueueStats | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [workersRes, statsRes] = await Promise.all([
        api.listWorkers(),
        api.getQueueStats(),
      ]);

      if (workersRes.data) setWorkers(workersRes.data.workers || []);
      if (statsRes.data) setQueueStats(statsRes.data);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, []);

  const activeWorkers = workers.filter((w) => w.status === "active").length;
  const healthyWorkers = workers.filter((w) => w.is_healthy).length;
  const totalVMs = workers.reduce((acc, w) => acc + (w.vm_count || 0), 0);
  const totalCPUsUsed = workers.reduce((acc, w) => acc + (w.used_cpu_cores || 0), 0);

  return (
    <div className="flex flex-col h-full">
      <Header onRefresh={fetchData} />
      <div className="flex-1 p-6 space-y-6">
        {/* Stats Cards */}
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Active Workers</CardTitle>
              <Cpu className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{activeWorkers}</div>
              <p className="text-xs text-muted-foreground">
                of {workers.length} total
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Healthy Workers</CardTitle>
              <CheckCircle className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{healthyWorkers}</div>
              <p className="text-xs text-muted-foreground">
                {workers.length - healthyWorkers} unhealthy
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total VMs</CardTitle>
              <Server className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{totalVMs}</div>
              <p className="text-xs text-muted-foreground">
                across all workers
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">CPUs Used</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{totalCPUsUsed}</div>
              <p className="text-xs text-muted-foreground">
                {queueStats?.pending || 0} tasks pending
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Queue Stats */}
        {queueStats && (
          <Card>
            <CardHeader>
              <CardTitle>Queue Statistics</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-6 gap-4">
                <div className="text-center">
                  <div className="text-2xl font-bold">{queueStats.pending}</div>
                  <div className="text-sm text-muted-foreground">Pending</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold">{queueStats.active}</div>
                  <div className="text-sm text-muted-foreground">Active</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold">{queueStats.scheduled}</div>
                  <div className="text-sm text-muted-foreground">Scheduled</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold">{queueStats.retry}</div>
                  <div className="text-sm text-muted-foreground">Retry</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold">{queueStats.completed}</div>
                  <div className="text-sm text-muted-foreground">Completed</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold">{queueStats.archived}</div>
                  <div className="text-sm text-muted-foreground">Archived</div>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Workers Table */}
        <Card>
          <CardHeader>
            <CardTitle>Worker Instances</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {loading && workers.length === 0 ? (
              <div className="p-6 text-center text-muted-foreground">
                Loading workers...
              </div>
            ) : workers.length === 0 ? (
              <div className="p-6 text-center text-muted-foreground">
                No workers found. Workers register automatically when they start.
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Worker</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Zone</TableHead>
                    <TableHead>Resources</TableHead>
                    <TableHead>VMs</TableHead>
                    <TableHead>Uptime</TableHead>
                    <TableHead>Last Seen</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {workers.map((worker) => (
                    <TableRow key={worker.id}>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Cpu className="h-4 w-4 text-muted-foreground" />
                          <div>
                            <div className="font-medium">{worker.hostname}</div>
                            <div className="text-xs text-muted-foreground">
                              {worker.address}
                            </div>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Badge
                            variant={
                              worker.status === "active"
                                ? "default"
                                : worker.status === "idle"
                                ? "secondary"
                                : "outline"
                            }
                          >
                            {worker.status}
                          </Badge>
                          {worker.is_healthy ? (
                            <CheckCircle className="h-4 w-4 text-green-500" />
                          ) : (
                            <XCircle className="h-4 w-4 text-red-500" />
                          )}
                        </div>
                      </TableCell>
                      <TableCell>{worker.zone}</TableCell>
                      <TableCell>
                        <div className="text-sm">
                          <div>{worker.used_cpu_cores}/{worker.cpu_cores} CPUs</div>
                          <div className="text-xs text-muted-foreground">
                            {Math.round(worker.memory_usage_percent)}% memory
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <span className="font-medium">{worker.vm_count}</span>
                        <span className="text-muted-foreground">/{worker.max_vms}</span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">{worker.uptime}</span>
                      </TableCell>
                      <TableCell>
                        {new Date(worker.last_seen).toLocaleString()}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
