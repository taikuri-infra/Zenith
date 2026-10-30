"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { AlertTriangle, ArrowLeft, Clock, Cog, ExternalLink, GitBranch, Globe, Terminal } from "lucide-react";
import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { AppDetailSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useApi } from "@/hooks/use-api";
import { type DeployApp, type PodStatus, monitoring } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import { getDiagnosticHint } from "./diagnostics";
import { SourceCard } from "./source-card";
import { DeploymentsCard } from "./deployments-card";
import { EnvironmentCard } from "./environment-card";
import { ResourcesCard } from "./resources-card";
import { ContainersCard } from "./containers-card";
import { DomainsCard } from "./domains-card";

export default function AppDetailPage() {
  const params = useParams();
  const appId = params.name as string;
  const { appsDeploy } = getApi();

  const { data: app, loading, error, refetch } = useApi(() => appsDeploy.get(appId), [appId]);

  if (loading) {
    return (
      <Shell>
        <AppDetailSkeleton />
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

  if (!app) {
    return (
      <Shell>
        <EmptyState title="App not found" description="This app does not exist." />
      </Shell>
    );
  }

  const isWeb = (app.app_type ?? "web") === "web";

  return (
    <Shell>
      <div className="space-y-6">
        <AppHeader app={app} />
        <AppBanners app={app} />
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          {/* Row 1: Source + Domains (web); Source alone, full width, otherwise. */}
          <SourceCard app={app} wide={!isWeb} />
          {isWeb && <DomainsCard app={app} />}
          <DeploymentsCard app={app} />
          <EnvironmentCard key={app.id} app={app} />
          <ContainersCard app={app} />
          <ResourcesCard app={app} />
        </div>
      </div>
    </Shell>
  );
}

function AppHeader({ app }: { app: DeployApp }) {
  const appType = app.app_type ?? "web";
  const isWeb = appType === "web";

  return (
    <div className="flex flex-wrap items-center gap-3">
      <Link
        href="/apps"
        aria-label="Back to apps"
        className="rounded-md p-1.5 text-neutral-500 transition-colors hover:bg-surface-200 hover:text-white"
      >
        <ArrowLeft className="h-4 w-4" />
      </Link>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-3">
          <h1 className="text-lg font-semibold text-white">{app.name}</h1>
          <span
            className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${
              appType === "web"
                ? "bg-blue-500/15 text-blue-400"
                : appType === "worker"
                  ? "bg-amber-500/15 text-amber-400"
                  : "bg-purple-500/15 text-purple-400"
            }`}
          >
            {appType === "web" && <Globe className="h-3 w-3" />}
            {appType === "worker" && <Cog className="h-3 w-3" />}
            {appType === "cron" && <Clock className="h-3 w-3" />}
            {appType}
          </span>
          <StatusBadge status={app.status as "running" | "deploying" | "stopped" | "crashed"} />
        </div>
        <div className="mt-1 flex flex-wrap items-center gap-4 text-xs text-neutral-500">
          {app.repo_url && (
            <span className="flex items-center gap-1">
              <GitBranch className="h-3 w-3" />
              {app.branch || "main"}
            </span>
          )}
          {isWeb && app.subdomain && (
            <span className="flex items-center gap-1">
              <Globe className="h-3 w-3" />
              <a href={app.url} target="_blank" rel="noopener noreferrer" className="text-accent-400 hover:underline">
                {app.subdomain}
              </a>
            </span>
          )}
          {appType === "cron" && app.cron_schedule && (
            <span className="flex items-center gap-1 font-mono">
              <Clock className="h-3 w-3" />
              {app.cron_schedule}
            </span>
          )}
          <span className="flex items-center gap-1">
            <Clock className="h-3 w-3" />
            {new Date(app.created_at).toLocaleDateString()}
          </span>
        </div>
      </div>
      <div className="flex items-center gap-2">
        {isWeb && app.url && (
          <a
            href={app.url}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs font-medium text-neutral-300 transition-colors hover:bg-surface-200 hover:text-white"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            Open
          </a>
        )}
        <Link
          href={`/logs?app=${app.id}`}
          className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs font-medium text-neutral-300 transition-colors hover:bg-surface-200 hover:text-white"
        >
          <Terminal className="h-3.5 w-3.5" />
          Logs
        </Link>
      </div>
    </div>
  );
}

function AppBanners({ app }: { app: DeployApp }) {
  const [diagnostic, setDiagnostic] = useState<PodStatus | null>(null);

  useEffect(() => {
    if (app.status === "failed" || app.status === "error") {
      monitoring
        .getPods(app.id)
        .then((res) => {
          const unhealthy = res.pods.find((p) => p.status_reason || !p.ready);
          if (unhealthy) setDiagnostic(unhealthy);
        })
        .catch(() => {});
    }
  }, [app.id, app.status]);

  if (!diagnostic && app.status !== "sleeping") return null;

  return (
    <div className="space-y-4">
      {diagnostic && (
        <div className="flex items-start gap-3 rounded-lg border border-red-500/30 bg-red-500/5 px-4 py-3">
          <AlertTriangle className="mt-0.5 h-5 w-5 flex-shrink-0 text-red-400" />
          <div>
            <p className="text-sm font-medium text-red-300">{diagnostic.status_reason || "Deployment Failed"}</p>
            <p className="mt-0.5 text-xs text-red-400/80">{getDiagnosticHint(diagnostic)}</p>
            {diagnostic.status_message && (
              <p className="mt-1 font-mono text-xs text-red-400/60">{diagnostic.status_message}</p>
            )}
            {diagnostic.last_exit_code !== undefined && diagnostic.last_exit_code !== 0 && (
              <p className="mt-1 text-xs text-red-400/60">Exit code: {diagnostic.last_exit_code}</p>
            )}
          </div>
        </div>
      )}
      {app.status === "sleeping" && (
        <div className="flex items-center gap-3 rounded-lg border border-indigo-500/20 bg-indigo-500/5 px-4 py-3">
          <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-indigo-400" />
          <div>
            <p className="text-sm font-medium text-indigo-300">This app is sleeping</p>
            <p className="text-xs text-indigo-400/70">
              It will wake automatically on the next HTTP request. Cold start takes ~3 seconds.
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
