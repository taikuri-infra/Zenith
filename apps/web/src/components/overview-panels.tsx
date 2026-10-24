"use client";

import Link from "next/link";
import { Database, GitBranch, Box, History, Rocket } from "lucide-react";
import { ProgressBar } from "@/components/progress-bar";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import type { ActivityEvent, AppDatabase, DeployApp, Deployment, MetricsOverview } from "@/lib/api";
import {
  appTypeLabel, formatMemory, formatPercent, shortRef, sourceKind, sourceLabel, statusTone, timeAgo, toneDot, toneText,
} from "@/lib/app-view";

const engineBadge: Record<string, { label: string; name: string; className: string }> = {
  postgresql: { label: "P", name: "PostgreSQL", className: "bg-blue-500/20 text-blue-400" },
  mysql: { label: "M", name: "MySQL", className: "bg-orange-500/20 text-orange-400" },
  redis: { label: "R", name: "Redis", className: "bg-red-500/20 text-red-400" },
  rabbitmq: { label: "Q", name: "RabbitMQ", className: "bg-amber-500/20 text-amber-400" },
  kafka: { label: "K", name: "Kafka", className: "bg-purple-500/20 text-purple-400" },
};

export function engineName(engine: string): string {
  return engineBadge[engine]?.name ?? engine;
}

function Panel({ title, icon, action, children }: {
  title: string;
  icon: React.ReactNode;
  action?: { href: string; label: string };
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-xl border border-border bg-surface-100">
      <header className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2 text-sm font-medium text-white">
          <span className="text-accent-400">{icon}</span>
          {title}
        </div>
        {action && (
          <Link href={action.href} className="text-xs text-neutral-500 transition-colors hover:text-white">
            {action.label}
          </Link>
        )}
      </header>
      {children}
    </section>
  );
}

function Empty({ text }: { text: string }) {
  return <p className="px-4 py-8 text-center text-xs text-neutral-500">{text}</p>;
}

export function ApplicationsPanel({ apps, metrics }: { apps: DeployApp[]; metrics: Record<string, MetricsOverview> }) {
  const rows = [...apps].sort((a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at)).slice(0, 8);
  return (
    <Panel title="Applications" icon={<Box className="h-4 w-4" />} action={{ href: "/apps", label: `View all (${apps.length})` }}>
      {rows.length === 0 ? (
        <Empty text="No applications yet." />
      ) : (
        <ul className="divide-y divide-border">
          {rows.map((app) => {
            const m = metrics[app.id];
            const tone = statusTone(app.status);
            return (
              <li key={app.id}>
                <Link href={`/apps/${app.id}`} className="grid grid-cols-[1fr_auto] items-center gap-3 px-4 py-3 transition-colors hover:bg-surface-200 sm:grid-cols-[1.4fr_1fr_auto]">
                  <div className="flex min-w-0 items-center gap-3">
                    <span className={`h-2 w-2 flex-shrink-0 rounded-full ${toneDot[tone]}`} />
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-white">{app.name}</p>
                      <p className="flex items-center gap-1 truncate text-[11px] text-neutral-500">
                        {sourceKind(app) === "git" ? <GitBranch className="h-3 w-3 flex-shrink-0" /> : <Box className="h-3 w-3 flex-shrink-0" />}
                        <span className="truncate">{sourceLabel(app)}</span>
                      </p>
                    </div>
                  </div>
                  <div className="hidden min-w-0 sm:block">
                    {m ? (
                      <div className="space-y-1">
                        <ProgressBar percent={m.cpu_percent} size="sm" label={`CPU ${formatPercent(m.cpu_percent)}`} />
                        <p className="text-[11px] text-neutral-500">{formatMemory(m.memory_mb)} RAM · {m.pod_count} pod{m.pod_count === 1 ? "" : "s"}</p>
                      </div>
                    ) : (
                      <p className="text-[11px] text-neutral-600">{app.status === "running" ? "collecting metrics…" : "—"}</p>
                    )}
                  </div>
                  <div className="flex flex-col items-end gap-0.5 text-right">
                    <span className={`text-xs ${toneText[tone]}`}>{app.status}</span>
                    <span className="text-[11px] text-neutral-500">{appTypeLabel(app.app_type)} · {timeAgo(app.updated_at)}</span>
                  </div>
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </Panel>
  );
}

export function DatabasesPanel({ dbs }: { dbs: AppDatabase[] }) {
  return (
    <Panel title="Databases" icon={<Database className="h-4 w-4" />} action={{ href: "/databases", label: `View all (${dbs.length})` }}>
      {dbs.length === 0 ? (
        <Empty text="No databases yet." />
      ) : (
        <ul className="divide-y divide-border">
          {dbs.slice(0, 6).map((db) => {
            const badge = engineBadge[db.engine] ?? { label: "?", name: db.engine, className: "bg-neutral-500/20 text-neutral-400" };
            const pct = db.max_size_mb > 0 ? Math.round((db.size_mb / db.max_size_mb) * 100) : 0;
            const tone = statusTone(db.status);
            return (
              <li key={db.id} className="flex items-center gap-3 px-4 py-2.5">
                <span className={`inline-flex h-6 w-6 flex-shrink-0 items-center justify-center rounded text-[10px] font-bold ${badge.className}`}>{badge.label}</span>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <span className="truncate text-sm text-white">{db.name}</span>
                    <span className={`text-[11px] ${toneText[tone]}`}>{db.status}</span>
                  </div>
                  <div className="mt-1">
                    <ProgressBar percent={pct} size="sm" label={`${db.size_mb} / ${db.max_size_mb} MB`} />
                  </div>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </Panel>
  );
}

function useRecentDeployments(apps: DeployApp[]) {
  const top = [...apps].sort((a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at)).slice(0, 5);
  const key = top.map((a) => a.id).join(",");
  return useApi<{ app: DeployApp; deployment: Deployment }[]>(async () => {
    const { appsDeploy: appsApi } = getApi();
    const settled = await Promise.allSettled(top.map((a) => appsApi.listDeployments(a.id, 1)));
    const out: { app: DeployApp; deployment: Deployment }[] = [];
    settled.forEach((r, i) => {
      if (r.status === "fulfilled" && r.value?.items?.[0]) out.push({ app: top[i], deployment: r.value.items[0] });
    });
    return out.sort((a, b) => Date.parse(b.deployment.created_at) - Date.parse(a.deployment.created_at));
  }, [key]);
}

export function RecentDeploymentsPanel({ apps }: { apps: DeployApp[] }) {
  const { data, loading } = useRecentDeployments(apps);
  const rows = data ?? [];
  return (
    <Panel title="Recent deployments" icon={<Rocket className="h-4 w-4" />}>
      {loading && rows.length === 0 ? (
        <Empty text="Loading…" />
      ) : rows.length === 0 ? (
        <Empty text="No deployments yet." />
      ) : (
        <ul className="divide-y divide-border">
          {rows.map(({ app, deployment }) => {
            const tone = statusTone(deployment.status);
            return (
              <li key={deployment.id}>
                <Link href={`/apps/${app.id}`} className="flex items-center gap-3 px-4 py-2.5 transition-colors hover:bg-surface-200">
                  <span className={`h-2 w-2 flex-shrink-0 rounded-full ${toneDot[tone]}`} />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm text-white">{app.name}</p>
                    <p className="truncate font-mono text-[11px] text-neutral-500">{shortRef(deployment.image_tag || deployment.git_sha)}</p>
                  </div>
                  <div className="text-right">
                    <p className={`text-xs ${toneText[tone]}`}>{deployment.status}</p>
                    <p className="text-[11px] text-neutral-500">{timeAgo(deployment.created_at)}</p>
                  </div>
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </Panel>
  );
}

export function ActivityPanel() {
  const { data } = useApi<ActivityEvent[]>(() => getApi().activity.list(), []);
  const rows = (data ?? []).slice(0, 6);
  return (
    <Panel title="Activity" icon={<History className="h-4 w-4" />} action={{ href: "/activity", label: "View all" }}>
      {rows.length === 0 ? (
        <Empty text="No activity yet." />
      ) : (
        <ul className="divide-y divide-border">
          {rows.map((e) => (
            <li key={e.id} className="px-4 py-2.5">
              <p className="truncate text-sm text-white">{e.title}</p>
              <p className="truncate text-[11px] text-neutral-500">{e.description} · {timeAgo(e.created_at)}</p>
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}

