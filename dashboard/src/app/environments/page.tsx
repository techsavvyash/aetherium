"use client";

import { useEffect, useState } from "react";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import {
  Layers,
  Plus,
  Trash2,
  RefreshCw,
  ExternalLink,
  Cpu,
  HardDrive,
  GitBranch,
  Wrench,
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

export default function EnvironmentsPage() {
  const [environments, setEnvironments] = useState<Environment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchEnvironments = async () => {
    setLoading(true);
    const result = await api.listEnvironments();
    if (result.error) {
      setError(result.error);
    } else if (result.data) {
      setEnvironments(result.data.environments || []);
      setError(null);
    }
    setLoading(false);
  };

  useEffect(() => {
    void fetchEnvironments();
  }, []);

  const handleDelete = async (id: string) => {
    const result = await api.deleteEnvironment(id);
    if (result.error) {
      setError(result.error);
    } else {
      fetchEnvironments();
    }
  };

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Layers className="h-8 w-8" />
            Environments
          </h1>
          <p className="text-muted-foreground mt-1">
            Reusable templates for VM configuration, tools, and MCP servers
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={fetchEnvironments} disabled={loading}>
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </Button>
          <Link href="/environments/create">
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Environment
            </Button>
          </Link>
        </div>
      </div>

      {error && (
        <Card className="mb-4 border-red-500">
          <CardContent className="pt-6">
            <p className="text-red-500">{error}</p>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Environment Templates</CardTitle>
          <CardDescription>
            {environments.length} environment{environments.length !== 1 ? "s" : ""} available
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center justify-center py-8">
              <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : environments.length === 0 ? (
            <div className="text-center py-8">
              <Layers className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-muted-foreground">No environments yet</p>
              <Link href="/environments/create" className="mt-4 inline-block">
                <Button>
                  <Plus className="h-4 w-4 mr-2" />
                  Create your first environment
                </Button>
              </Link>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Resources</TableHead>
                  <TableHead>Repository</TableHead>
                  <TableHead>Tools</TableHead>
                  <TableHead>MCP Servers</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {environments.map((env) => (
                  <TableRow key={env.id}>
                    <TableCell>
                      <div>
                        <Link
                          href={`/environments/${env.id}`}
                          className="font-medium hover:underline"
                        >
                          {env.name}
                        </Link>
                        {env.description && (
                          <p className="text-sm text-muted-foreground truncate max-w-xs">
                            {env.description}
                          </p>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-col gap-1">
                        <span className="flex items-center gap-1 text-sm">
                          <Cpu className="h-3 w-3" />
                          {env.vcpus} vCPUs
                        </span>
                        <span className="flex items-center gap-1 text-sm">
                          <HardDrive className="h-3 w-3" />
                          {env.memory_mb} MB
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      {env.git_repo_url ? (
                        <div className="flex items-center gap-1 text-sm">
                          <GitBranch className="h-3 w-3" />
                          <span className="truncate max-w-[150px]" title={env.git_repo_url}>
                            {env.git_branch}
                          </span>
                        </div>
                      ) : (
                        <span className="text-muted-foreground text-sm">-</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {env.tools?.slice(0, 3).map((tool) => (
                          <Badge key={tool} variant="secondary" className="text-xs">
                            <Wrench className="h-2 w-2 mr-1" />
                            {tool}
                          </Badge>
                        ))}
                        {(env.tools?.length || 0) > 3 && (
                          <Badge variant="outline" className="text-xs">
                            +{env.tools!.length - 3}
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {env.mcp_servers?.map((mcp) => (
                          <Badge key={mcp.name} variant="outline" className="text-xs">
                            {mcp.name}
                          </Badge>
                        ))}
                        {!env.mcp_servers?.length && (
                          <span className="text-muted-foreground text-sm">-</span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      {new Date(env.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Link href={`/environments/${env.id}`}>
                          <Button variant="ghost" size="sm">
                            <ExternalLink className="h-4 w-4" />
                          </Button>
                        </Link>
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <Button variant="ghost" size="sm">
                              <Trash2 className="h-4 w-4 text-red-500" />
                            </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>Delete Environment</AlertDialogTitle>
                              <AlertDialogDescription>
                                Are you sure you want to delete &quot;{env.name}&quot;?
                                Workspaces using this environment will not be affected.
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>Cancel</AlertDialogCancel>
                              <AlertDialogAction
                                onClick={() => handleDelete(env.id)}
                                className="bg-red-500 hover:bg-red-600"
                              >
                                Delete
                              </AlertDialogAction>
                            </AlertDialogFooter>
                          </AlertDialogContent>
                        </AlertDialog>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
