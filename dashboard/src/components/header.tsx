"use client";

import { usePathname } from "next/navigation";
import { Button } from "@/components/ui/button";
import { RefreshCw, Bell } from "lucide-react";

const pageTitles: Record<string, string> = {
  "/": "Dashboard",
  "/vms": "Virtual Machines",
  "/workers": "Workers",
  "/tasks": "Tasks",
  "/api-explorer": "API Explorer",
  "/settings": "Settings",
};

export function Header({ onRefresh }: { onRefresh?: () => void }) {
  const pathname = usePathname();
  const title = pageTitles[pathname] || "Dashboard";

  return (
    <header className="flex h-16 items-center justify-between border-b bg-background px-6">
      <h1 className="text-xl font-semibold">{title}</h1>
      <div className="flex items-center gap-2">
        {onRefresh && (
          <Button variant="ghost" size="icon" onClick={onRefresh}>
            <RefreshCw className="h-4 w-4" />
          </Button>
        )}
        <Button variant="ghost" size="icon">
          <Bell className="h-4 w-4" />
        </Button>
      </div>
    </header>
  );
}
