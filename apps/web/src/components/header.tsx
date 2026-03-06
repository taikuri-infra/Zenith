"use client";

import { useCallback, useRef, useState } from "react";
import Link from "next/link";
import {
  Bell,
  LogOut,
  Search,
  Settings,
  User,
  Rocket,
  CheckCircle,
  XCircle,
  AlertTriangle,
  AlertCircle,
} from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { useClickOutside } from "@/hooks/use-click-outside";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import type { Notification } from "@/lib/api";

const notifIcons: Record<string, { icon: React.ComponentType<{ className?: string }>; color: string }> = {
  deploy_started: { icon: Rocket, color: "text-amber-400" },
  deploy_success: { icon: CheckCircle, color: "text-emerald-400" },
  deploy_failed: { icon: XCircle, color: "text-red-400" },
  app_crashed: { icon: AlertTriangle, color: "text-red-400" },
  plan_warning: { icon: AlertCircle, color: "text-amber-400" },
};

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export function Header() {
  const { user, logout } = useAuth();
  const { notifications: notifApi } = getApi();

  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [notifOpen, setNotifOpen] = useState(false);

  const userMenuRef = useRef<HTMLDivElement>(null);
  const notifRef = useRef<HTMLDivElement>(null);

  const closeUserMenu = useCallback(() => setUserMenuOpen(false), []);
  const closeNotif = useCallback(() => setNotifOpen(false), []);

  useClickOutside(userMenuRef, closeUserMenu);
  useClickOutside(notifRef, closeNotif);

  const { data: notifData, refetch: refetchNotifs } = useApi(
    () => notifApi.list(),
    []
  );

  const notifList: Notification[] = notifData ?? [];
  const unreadCount = notifList.filter((n) => !n.read).length;

  const toggleUserMenu = () => {
    setUserMenuOpen((prev) => !prev);
    setNotifOpen(false);
  };

  const toggleNotif = () => {
    setNotifOpen((prev) => !prev);
    setUserMenuOpen(false);
  };

  const handleMarkAllRead = async () => {
    try {
      await notifApi.markAllRead();
      refetchNotifs();
    } catch {
      // ignore
    }
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
            {unreadCount > 0 && (
              <span className="absolute right-1 top-1 flex h-4 w-4 items-center justify-center rounded-full bg-accent-500 text-[9px] font-bold text-white">
                {unreadCount}
              </span>
            )}
          </button>

          {notifOpen && (
            <div className="absolute right-0 top-full mt-2 w-96 rounded-lg border border-border bg-surface-50 shadow-xl shadow-black/40">
              <div className="border-b border-border px-4 py-3 flex items-center justify-between">
                <h3 className="text-sm font-medium text-white">
                  Notifications
                </h3>
                {unreadCount > 0 && (
                  <span className="rounded-full bg-accent-500/20 px-2 py-0.5 text-[10px] font-medium text-accent-400">
                    {unreadCount} new
                  </span>
                )}
              </div>

              {notifList.length === 0 ? (
                <div className="flex flex-col items-center gap-2 px-4 py-10 text-neutral-500">
                  <Bell className="h-6 w-6" />
                  <span className="text-sm">No new notifications</span>
                </div>
              ) : (
                <>
                  <div className="max-h-80 overflow-y-auto divide-y divide-border">
                    {notifList.map((notif) => {
                      const cfg = notifIcons[notif.type] ?? { icon: Bell, color: "text-neutral-400" };
                      const Icon = cfg.icon;
                      return (
                        <div
                          key={notif.id}
                          className={`flex items-start gap-3 px-4 py-3 ${
                            !notif.read ? "bg-surface-100/50" : ""
                          }`}
                        >
                          <Icon className={`h-4 w-4 mt-0.5 shrink-0 ${cfg.color}`} />
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-2">
                              <p className="text-sm font-medium text-white truncate">{notif.title}</p>
                              {!notif.read && (
                                <span className="inline-block h-1.5 w-1.5 rounded-full bg-accent-500 shrink-0" />
                              )}
                            </div>
                            <p className="text-xs text-neutral-500 mt-0.5 truncate">{notif.description}</p>
                            <p className="text-[10px] text-neutral-600 mt-1">{timeAgo(notif.created_at)}</p>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                  {unreadCount > 0 && (
                    <div className="border-t border-border px-4 py-2.5">
                      <button
                        onClick={handleMarkAllRead}
                        className="w-full text-center text-xs text-accent-400 hover:text-accent-300 transition-colors"
                      >
                        Mark all as read
                      </button>
                    </div>
                  )}
                </>
              )}
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
