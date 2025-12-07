"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";
import type { Environment } from "@/lib/types";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Layers,
  ArrowLeft,
  RefreshCw,
  Cpu,
  HardDrive,
  GitBranch,
  Wrench,
  Server,
  Clock,
  FolderOpen,
  Variable,
  Trash2,
  Plus,
} from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";

export default function EnvironmentDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id as string;

  const [environment, setEnvironment] = useState<Environment | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchEnvironment = async () => {
    setLoading(true);
    const result = await api.getEnvironment(id);
    if (result.error) {
      setError(result.error);
    } else if (result.data) {
      setEnvironment(result.data);
      setError(null);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (id) {
      void fetchEnvironment();
    }
  }, [id]);

  const handleDelete = async () => {
    const result = await api.deleteEnvironment(id);
    if (result.error) {
      setError(result.error);
    } else {
      router.push("/environments");
    }
  };

  if (loading) {
    return (
      <div className="p-8 flex items-center justify-center">
        <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error || !environment) {
    return (
      <div className="p-8">
        <Card className="border-red-500">
          <CardContent className="pt-6">
            <p className="text-red-500">{error || "Environment not found"}</p>
            <Link href="/environments" className="mt-4 inline-block">
              <Button variant="outline">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Environments
              </Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-8">
        <div className="flex items-center gap-4">
          <Link href="/environments">
            <Button variant="ghost">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold flex items-center gap-2">
              <Layers className="h-8 w-8" />
              {environment.name}
            </h1>
            {environment.description && (
              <p className="text-muted-foreground mt-1">{environment.description}</p>
            )}
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={fetchEnvironment}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
          <Link href={`/workspaces/create?environment=${environment.id}`}>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Workspace
            </Button>
          </Link>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="destructive">
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete Environment</AlertDialogTitle>
                <AlertDialogDescription>
                  Are you sure you want to delete &quot;{environment.name}&quot;?
                  Workspaces using this environment will not be affected.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  onClick={handleDelete}
                  className="bg-red-500 hover:bg-red-600"
                >
                  Delete
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Resources */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Cpu className="h-5 w-5" />
              VM Resources
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-2 text-muted-foreground">
                <Cpu className="h-4 w-4" />
                vCPUs
              </span>
              <span className="font-mono">{environment.vcpus}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-2 text-muted-foreground">
                <HardDrive className="h-4 w-4" />
                Memory
              </span>
              <span className="font-mono">{environment.memory_mb} MB</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-2 text-muted-foreground">
                <Clock className="h-4 w-4" />
                Idle Timeout
              </span>
              <span className="font-mono">
                {Math.floor(environment.idle_timeout_seconds / 60)} min
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-2 text-muted-foreground">
                <FolderOpen className="h-4 w-4" />
                Working Directory
              </span>
              <code className="text-sm bg-muted px-2 py-1 rounded">
                {environment.working_directory}
              </code>
            </div>
          </CardContent>
        </Card>

        {/* Repository */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <GitBranch className="h-5 w-5" />
              Repository
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {environment.git_repo_url ? (
              <>
                <div>
                  <span className="text-sm text-muted-foreground">URL</span>
                  <p className="font-mono text-sm break-all">
                    {environment.git_repo_url}
                  </p>
                </div>
                <div>
                  <span className="text-sm text-muted-foreground">Branch</span>
                  <p className="font-mono">{environment.git_branch}</p>
                </div>
              </>
            ) : (
              <p className="text-muted-foreground">No repository configured</p>
            )}
          </CardContent>
        </Card>

        {/* Tools */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Wrench className="h-5 w-5" />
              Tools
            </CardTitle>
            <CardDescription>
              {environment.tools?.length || 0} tool(s) will be installed
            </CardDescription>
          </CardHeader>
          <CardContent>
            {environment.tools?.length ? (
              <div className="flex flex-wrap gap-2">
                {environment.tools.map((tool) => (
                  <Badge key={tool} variant="secondary">
                    <Wrench className="h-3 w-3 mr-1" />
                    {tool}
                  </Badge>
                ))}
              </div>
            ) : (
              <p className="text-muted-foreground">No tools configured</p>
            )}
          </CardContent>
        </Card>

        {/* MCP Servers */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Server className="h-5 w-5" />
              MCP Servers
            </CardTitle>
            <CardDescription>
              Model Context Protocol servers for Claude Code
            </CardDescription>
          </CardHeader>
          <CardContent>
            {environment.mcp_servers?.length ? (
              <div className="space-y-3">
                {environment.mcp_servers.map((server) => (
                  <div
                    key={server.name}
                    className="p-3 border rounded space-y-1"
                  >
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{server.name}</span>
                      <Badge variant="outline">{server.type}</Badge>
                    </div>
                    <p className="text-sm text-muted-foreground font-mono">
                      {server.type === "stdio"
                        ? `${server.command} ${server.args?.join(" ")}`
                        : server.url}
                    </p>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-muted-foreground">No MCP servers configured</p>
            )}
          </CardContent>
        </Card>

        {/* Environment Variables */}
        <Card className="md:col-span-2">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Variable className="h-5 w-5" />
              Environment Variables
            </CardTitle>
          </CardHeader>
          <CardContent>
            {environment.env_vars && Object.keys(environment.env_vars).length > 0 ? (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                {Object.entries(environment.env_vars).map(([key, value]) => (
                  <div
                    key={key}
                    className="p-2 border rounded font-mono text-sm"
                  >
                    <span className="text-blue-500">{key}</span>
                    <span className="text-muted-foreground">=</span>
                    <span>{value}</span>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-muted-foreground">No environment variables configured</p>
            )}
          </CardContent>
        </Card>

        {/* Metadata */}
        <Card className="md:col-span-2">
          <CardHeader>
            <CardTitle>Metadata</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div>
                <span className="text-muted-foreground">ID</span>
                <p className="font-mono truncate" title={environment.id}>
                  {environment.id}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">Created</span>
                <p>{new Date(environment.created_at).toLocaleString()}</p>
              </div>
              <div>
                <span className="text-muted-foreground">Updated</span>
                <p>{new Date(environment.updated_at).toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
