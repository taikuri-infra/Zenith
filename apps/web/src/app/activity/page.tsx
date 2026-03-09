"use client";

import { Shell } from "@/components/shell";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import { Rocket, Database, Plus, CreditCard, Globe, Search } from "lucide-react";
import type { ActivityEvent } from "@/lib/api";
import { useState, useMemo } from "react";

const typeConfig: Record<string, { icon: React.ComponentType<{ className?: string }>; color: string; bg: string; label: string }> = {
  deploy: { icon: Rocket, color: "text-blue-400", bg: "bg-blue-500/15", label: "Deploy" },
  db_create: { icon: Database, color: "text-emerald-400", bg: "bg-emerald-500/15", label: "Database" },
  app_create: { icon: Plus, color: "text-purple-400", bg: "bg-purple-500/15", label: "App" },
  plan_change: { icon: CreditCard, color: "text-amber-400", bg: "bg-amber-500/15", label: "Plan" },
  domain_add: { icon: Globe, color: "text-cyan-400", bg: "bg-cyan-500/15", label: "Domain" },
};

const EVENT_TYPES = [
  { value: "all", label: "All events" },
  { value: "deploy", label: "Deployments" },
  { value: "db_create", label: "Database ops" },
  { value: "app_create", label: "App created" },
  { value: "plan_change", label: "Plan changes" },
  { value: "domain_add", label: "Domains" },
];

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

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  const today = new Date();
  const yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);

  if (d.toDateString() === today.toDateString()) return "Today";
  if (d.toDateString() === yesterday.toDateString()) return "Yesterday";
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
}

export default function ActivityPage() {
  const { activity } = getApi();
  const [typeFilter, setTypeFilter] = useState("all");
  const [searchQuery, setSearchQuery] = useState("");

  const { data, loading, error, refetch } = useApi(() => activity.list(), []);

  const events: ActivityEvent[] = Array.isArray(data) ? data : [];

  const filtered = useMemo(() => {
    let result = events;

    if (typeFilter !== "all") {
      result = result.filter((e) => e.type === typeFilter);
    }

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      result = result.filter(
        (e) =>
          e.title.toLowerCase().includes(q) ||
          e.description.toLowerCase().includes(q)
      );
    }

    return result;
  }, [events, typeFilter, searchQuery]);

  // Group events by date
  const grouped = useMemo(() => {
    const groups: { date: string; events: ActivityEvent[] }[] = [];
    for (const event of filtered) {
      const date = formatDate(event.created_at);
      const existing = groups.find((g) => g.date === date);
      if (existing) {
        existing.events.push(event);
      } else {
        groups.push({ date, events: [event] });
      }
    }
    return groups;
  }, [filtered]);

  if (loading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={3} rows={5} />
      </Shell>
    );
  }

  if (error) {
    return (
      <Shell>
        <ErrorState message={error} onRetry={refetch} />
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Activity</h1>
          <p className="text-sm text-neutral-500">
            Timeline of project events
          </p>
        </div>

        {/* Filters */}
        <div className="flex items-center gap-3">
          <div className="relative flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-500" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search events..."
              className="w-full rounded-lg border border-border bg-surface-100 py-1.5 pl-9 pr-3 text-sm text-white placeholder:text-neutral-500 focus:border-accent-500 focus:outline-none"
            />
          </div>

          <select
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value)}
            className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 focus:border-accent-500 focus:outline-none"
          >
            {EVENT_TYPES.map((t) => (
              <option key={t.value} value={t.value}>{t.label}</option>
            ))}
          </select>

          <span className="text-xs text-neutral-600 shrink-0">
            {filtered.length} / {events.length}
          </span>
        </div>

        {/* Active filters */}
        {(typeFilter !== "all" || searchQuery.trim()) && (
          <div className="flex items-center gap-2 flex-wrap">
            {typeFilter !== "all" && (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-accent-500/10 px-2.5 py-1 text-xs text-accent-400">
                Type: {EVENT_TYPES.find((t) => t.value === typeFilter)?.label}
                <button onClick={() => setTypeFilter("all")} className="hover:text-white">&times;</button>
              </span>
            )}
            {searchQuery.trim() && (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-accent-500/10 px-2.5 py-1 text-xs text-accent-400">
                Search: &ldquo;{searchQuery}&rdquo;
                <button onClick={() => setSearchQuery("")} className="hover:text-white">&times;</button>
              </span>
            )}
            <button
              onClick={() => { setTypeFilter("all"); setSearchQuery(""); }}
              className="text-xs text-neutral-500 hover:text-white transition-colors"
            >
              Clear all
            </button>
          </div>
        )}

        {/* Timeline */}
        {filtered.length === 0 ? (
          <EmptyState
            title="No events found"
            description={events.length > 0 ? "Try adjusting your filters." : "Events will appear here as you deploy apps and manage resources."}
          />
        ) : (
          <div className="space-y-6">
            {grouped.map((group) => (
              <div key={group.date}>
                {/* Date header */}
                <div className="sticky top-14 z-10 mb-3 flex items-center gap-3">
                  <span className="text-xs font-semibold text-neutral-400 bg-surface px-1">{group.date}</span>
                  <div className="flex-1 h-px bg-border" />
                  <span className="text-[10px] text-neutral-600">{group.events.length} event{group.events.length !== 1 ? "s" : ""}</span>
                </div>

                <div className="relative ml-4">
                  {/* Vertical line */}
                  <div className="absolute left-3 top-2 bottom-2 w-px bg-border" />

                  <div className="space-y-0">
                    {group.events.map((event) => {
                      const cfg = typeConfig[event.type] ?? typeConfig.deploy;
                      const Icon = cfg.icon;
                      return (
                        <div key={event.id} className="relative flex gap-4 py-3">
                          <div className={`relative z-10 flex h-7 w-7 shrink-0 items-center justify-center rounded-full ${cfg.bg}`}>
                            <Icon className={`h-3.5 w-3.5 ${cfg.color}`} />
                          </div>

                          <div className="flex-1 rounded-lg border border-border bg-surface-100 p-4">
                            <div className="flex items-start justify-between gap-3">
                              <div className="min-w-0">
                                <div className="flex items-center gap-2">
                                  <h3 className="text-sm font-medium text-white">{event.title}</h3>
                                  <span className={`rounded-full px-1.5 py-0.5 text-[9px] font-medium ${cfg.bg} ${cfg.color}`}>
                                    {cfg.label}
                                  </span>
                                </div>
                                <p className="mt-1 text-xs text-neutral-500">{event.description}</p>
                              </div>
                              <span className="text-[10px] text-neutral-600 shrink-0 mt-0.5">
                                {timeAgo(event.created_at)}
                              </span>
                            </div>
                            <p className="mt-2 text-[10px] text-neutral-600">
                              {new Date(event.created_at).toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit" })}
                            </p>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </Shell>
  );
}
