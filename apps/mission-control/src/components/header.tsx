"use client";

import { Bell, User } from "lucide-react";

interface HeaderProps {
  version?: string;
}

export function Header({ version = "v1.2.1" }: HeaderProps) {
  return (
    <header className="sticky top-0 z-30 flex h-14 items-center justify-between border-b border-border bg-surface-50/80 px-6 backdrop-blur-sm">
      <div className="flex items-center gap-3">
        <span className="text-xs font-medium text-neutral-500">
          Zenith {version}
        </span>
      </div>

      <div className="flex items-center gap-2">
        {/* Notification bell */}
        <button className="relative flex h-8 w-8 items-center justify-center rounded-md text-neutral-400 transition-colors hover:bg-surface-300 hover:text-white">
          <Bell className="h-4 w-4" />
          <span className="absolute right-1 top-1 h-2 w-2 rounded-full bg-accent-500" />
        </button>

        {/* User avatar */}
        <button className="flex h-8 items-center gap-2 rounded-md px-2 text-neutral-400 transition-colors hover:bg-surface-300 hover:text-white">
          <div className="flex h-6 w-6 items-center justify-center rounded-full bg-surface-400">
            <User className="h-3.5 w-3.5" />
          </div>
          <span className="text-sm">admin</span>
        </button>
      </div>
    </header>
  );
}
