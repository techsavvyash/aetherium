"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import type { Workspace } from "@/lib/types";
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
  Sparkles,
  Plus,
  Trash2,
  RefreshCw,
  ExternalLink,
  Bot,
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

function getStatusBadge(status: string) {
  switch (status.toLowerCase()) {
    case "ready":
      return <Badge className="bg-green-500">Ready</Badge>;
    case "creating":
      return <Badge className="bg-blue-500">Creating</Badge>;
    case "preparing":
      return <Badge className="bg-yellow-500">Preparing</Badge>;
    case "failed":
      return <Badge variant="destructive">Failed</Badge>;
    case "stopped":
      return <Badge variant="secondary">Stopped</Badge>;
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
}

function getAIAssistantBadge(assistant: string) {
  switch (assistant.toLowerCase()) {
    case "claude-code":
      return (
        <Badge variant="outline" className="gap-1">
          <Bot className="h-3 w-3" />
          Claude Code
        </Badge>
      );
    case "ampcode":
    case "amp":
      return (
        <Badge variant="outline" className="gap-1">
          <Bot className="h-3 w-3" />
          Ampcode
        </Badge>
      );
    default:
      return <Badge variant="outline">{assistant}</Badge>;
  }
}

export default function WorkspacesPage() {
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchWorkspaces = async () => {
    setLoading(true);
    const result = await api.listWorkspaces();
    if (result.error) {
      setError(result.error);
    } else if (result.data) {
      setWorkspaces(result.data.workspaces || []);
      setError(null);
    }
    setLoading(false);
  };

  useEffect(() => {
    void fetchWorkspaces();
  }, []);

  const handleDelete = async (id: string) => {
    const result = await api.deleteWorkspace(id);
    if (result.error) {
      setError(result.error);
    } else {
      fetchWorkspaces();
    }
  };

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Sparkles className="h-8 w-8" />
            AI Workspaces
          </h1>
          <p className="text-muted-foreground mt-1">
            Manage isolated environments for Claude Code and Ampcode
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={fetchWorkspaces} disabled={loading}>
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </Button>
          <Link href="/workspaces/create">
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Workspace
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
          <CardTitle>Workspaces</CardTitle>
          <CardDescription>
            {workspaces.length} workspace{workspaces.length !== 1 ? "s" : ""} available
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center justify-center py-8">
              <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : workspaces.length === 0 ? (
            <div className="text-center py-8">
              <Sparkles className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-muted-foreground">No workspaces yet</p>
              <Link href="/workspaces/create" className="mt-4 inline-block">
                <Button>
                  <Plus className="h-4 w-4 mr-2" />
                  Create your first workspace
                </Button>
              </Link>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>AI Assistant</TableHead>
                  <TableHead>Working Directory</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {workspaces.map((workspace) => (
                  <TableRow key={workspace.id}>
                    <TableCell>
                      <div>
                        <Link
                          href={`/workspaces/${workspace.id}`}
                          className="font-medium hover:underline"
                        >
                          {workspace.name}
                        </Link>
                        {workspace.description && (
                          <p className="text-sm text-muted-foreground truncate max-w-xs">
                            {workspace.description}
                          </p>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>{getStatusBadge(workspace.status)}</TableCell>
                    <TableCell>{getAIAssistantBadge(workspace.ai_assistant)}</TableCell>
                    <TableCell>
                      <code className="text-xs bg-muted px-1 py-0.5 rounded">
                        {workspace.working_directory}
                      </code>
                    </TableCell>
                    <TableCell>
                      {new Date(workspace.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Link href={`/workspaces/${workspace.id}`}>
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
                              <AlertDialogTitle>Delete Workspace</AlertDialogTitle>
                              <AlertDialogDescription>
                                Are you sure you want to delete &quot;{workspace.name}&quot;?
                                This will also delete the associated VM and all data.
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>Cancel</AlertDialogCancel>
                              <AlertDialogAction
                                onClick={() => handleDelete(workspace.id)}
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
