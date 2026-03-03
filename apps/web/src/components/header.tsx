"use client";

import { useCallback, useRef, useState } from "react";
import Link from "next/link";
import { Bell, LogOut, Search, Settings, User } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { useClickOutside } from "@/hooks/use-click-outside";

export function Header() {
  const { user, logout } = useAuth();

  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [notifOpen, setNotifOpen] = useState(false);

  const userMenuRef = useRef<HTMLDivElement>(null);
  const notifRef = useRef<HTMLDivElement>(null);

  const closeUserMenu = useCallback(() => setUserMenuOpen(false), []);
  const closeNotif = useCallback(() => setNotifOpen(false), []);

  useClickOutside(userMenuRef, closeUserMenu);
  useClickOutside(notifRef, closeNotif);

  const toggleUserMenu = () => {
    setUserMenuOpen((prev) => !prev);
    setNotifOpen(false);
  };

  const toggleNotif = () => {
    setNotifOpen((prev) => !prev);
    setUserMenuOpen(false);
  };

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
        {/* Notifications */}
        <div ref={notifRef} className="relative">
          <button
            onClick={toggleNotif}
            className="relative flex h-8 w-8 items-center justify-center rounded-md text-neutral-400 transition-colors hover:bg-surface-300 hover:text-white"
          >
            <Bell className="h-4 w-4" />
            {/* Badge — uncomment when count > 0
            <span className="absolute right-1 top-1 h-2 w-2 rounded-full bg-accent-500" />
            */}
          </button>

          {notifOpen && (
            <div className="absolute right-0 top-full mt-2 w-80 rounded-lg border border-border bg-surface-50 shadow-xl shadow-black/40">
              <div className="border-b border-border px-4 py-3">
                <h3 className="text-sm font-medium text-white">
                  Notifications
                </h3>
              </div>
              <div className="flex flex-col items-center gap-2 px-4 py-10 text-neutral-500">
                <Bell className="h-6 w-6" />
                <span className="text-sm">No new notifications</span>
              </div>
            </div>
          )}
        </div>

        {/* User menu */}
        <div ref={userMenuRef} className="relative">
          <button
            onClick={toggleUserMenu}
            className="flex h-8 items-center gap-2 rounded-md px-2 text-neutral-400 transition-colors hover:bg-surface-300 hover:text-white"
          >
            <div className="flex h-6 w-6 items-center justify-center rounded-full bg-accent-600">
              <User className="h-3.5 w-3.5 text-white" />
            </div>
            <span className="text-sm">{user?.name || "User"}</span>
          </button>

          {userMenuOpen && (
            <div className="absolute right-0 top-full mt-2 w-56 rounded-lg border border-border bg-surface-50 shadow-xl shadow-black/40">
              {/* User info */}
              <div className="border-b border-border px-4 py-3">
                <p className="text-sm font-medium text-white">
                  {user?.name || "User"}
                </p>
                <p className="mt-0.5 text-xs text-neutral-500">
                  {user?.email || ""}
                </p>
                {user?.role && (
                  <span className="mt-1.5 inline-block rounded-full bg-accent-500/10 px-2 py-0.5 text-xs text-accent-400">
                    {user.role}
                  </span>
                )}
              </div>

              {/* Actions */}
              <div className="p-1.5">
                <Link
                  href="/settings"
                  onClick={closeUserMenu}
                  className="flex w-full items-center gap-2.5 rounded-md px-2.5 py-2 text-sm text-neutral-400 transition-colors hover:bg-surface-300 hover:text-white"
                >
                  <Settings className="h-4 w-4" />
                  Profile
                </Link>
                <button
                  onClick={logout}
                  className="flex w-full items-center gap-2.5 rounded-md px-2.5 py-2 text-sm text-neutral-400 transition-colors hover:bg-red-500/10 hover:text-red-400"
                >
                  <LogOut className="h-4 w-4" />
                  Sign Out
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
