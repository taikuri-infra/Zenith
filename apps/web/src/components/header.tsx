"use client";

import { Bell, Search, User } from "lucide-react";

export function Header() {
  return (
    <header className="sticky top-0 z-30 flex h-14 items-center justify-between border-b border-border bg-surface-50/80 px-6 backdrop-blur-sm">
      {/* Search */}
      <div className="flex items-center gap-2">
        <button className="flex h-8 items-center gap-2 rounded-md border border-border bg-surface-200 px-3 text-xs text-neutral-500 transition-colors hover:border-border-hover hover:text-neutral-300">
          <Search className="h-3.5 w-3.5" />
          <span>Search...</span>
          <kbd className="ml-4 rounded border border-border bg-surface-300 px-1.5 py-0.5 text-[10px] text-neutral-500">
            ⌘K
          </kbd>
        </button>
      </div>

      <div className="flex items-center gap-2">
        <button className="relative flex h-8 w-8 items-center justify-center rounded-md text-neutral-400 transition-colors hover:bg-surface-300 hover:text-white">
          <Bell className="h-4 w-4" />
        </button>
        <button className="flex h-8 items-center gap-2 rounded-md px-2 text-neutral-400 transition-colors hover:bg-surface-300 hover:text-white">
          <div className="flex h-6 w-6 items-center justify-center rounded-full bg-accent-600">
            <User className="h-3.5 w-3.5 text-white" />
          </div>
          <span className="text-sm">babak</span>
        </button>
      </div>
    </header>
  );
}
