"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { BuildLogViewer } from "@/components/build-log-viewer";
import { useToast } from "@/components/toast";
import { useApi } from "@/hooks/use-api";
import { useDeployLogs } from "@/hooks/use-deploy-logs";
import { type DeployApp, type Deployment, type EnvVar, type Secret, type Release, type AppDatabase, type DatabaseBackup, type AppAuthConfig, type AppAuthUser, type AppBucket, type CustomDomain } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import { useState, useCallback } from "react";
import { useDeployEvents } from "@/hooks/use-deploy-events";
import {
  GitBranch,
  Globe,
  Clock,
  ArrowLeft,
  RotateCcw,
  Trash2,
  Plus,
  Settings,
  Layers,
  Eye,
  EyeOff,
  Terminal,
  KeyRound,
  Tag,
  Rocket,
  Copy,
  Check,
  Lock,
  Unlock,
  Database,
  Shield,
  Users,
  HardDrive,
  Archive,
  RotateCw,
  Download,
  Cog,
  Heart,
} from "lucide-react";
import Link from "next/link";
import { useParams } from "next/navigation";

type Tab = "overview" | "deployments" | "releases" | "logs" | "databases" | "storage" | "auth" | "domains" | "secrets" | "env";

export default function AppDetailPage() {
  const params = useParams();
  const appId = params.name as string;
  const { appsDeploy } = getApi();
  const [activeTab, setActiveTab] = useState<Tab>("overview");

  const { data: app, loading, error, refetch } = useApi(
    () => appsDeploy.get(appId),
    [appId]
  );

  if (loading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={4} rows={3} />
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

  const appType = app.app_type ?? "web";
  const isWeb = appType === "web";

  const allTabs: { key: Tab; label: string; icon: React.ComponentType<{ className?: string }>; webOnly?: boolean }[] = [
    { key: "overview", label: "Overview", icon: Eye },
    { key: "deployments", label: "Deployments", icon: Layers },
    { key: "releases", label: "Releases", icon: Tag },
    { key: "logs", label: "Logs", icon: Terminal },
    { key: "databases", label: "Databases", icon: Database },
    { key: "storage", label: "Storage", icon: HardDrive },
    { key: "auth", label: "Auth", icon: Shield },
    { key: "domains", label: "Domains", icon: Globe, webOnly: true },
    { key: "secrets", label: "Secrets", icon: KeyRound },
    { key: "env", label: "Environment", icon: Settings },
  ];
  const tabs = allTabs.filter((t) => !t.webOnly || isWeb);

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center gap-3">
          <Link
            href="/apps"
            className="rounded-md p-1.5 text-neutral-500 hover:bg-surface-200 hover:text-white transition-colors"
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div className="flex-1">
            <div className="flex items-center gap-3">
              <h1 className="text-lg font-semibold text-white">{app.name}</h1>
              <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${
                appType === "web" ? "bg-blue-500/15 text-blue-400" :
                appType === "worker" ? "bg-amber-500/15 text-amber-400" :
                "bg-purple-500/15 text-purple-400"
              }`}>
                {appType === "web" && <Globe className="h-3 w-3" />}
                {appType === "worker" && <Cog className="h-3 w-3" />}
                {appType === "cron" && <Clock className="h-3 w-3" />}
                {appType}
              </span>
              <StatusBadge status={app.status as "running" | "deploying" | "stopped" | "crashed"} />
            </div>
            <div className="mt-1 flex items-center gap-4 text-xs text-neutral-500">
              <span className="flex items-center gap-1">
                <GitBranch className="h-3 w-3" />
                {app.branch || "main"}
              </span>
              {isWeb && app.subdomain && (
                <span className="flex items-center gap-1">
                  <Globe className="h-3 w-3" />
                  <a
                    href={app.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-accent-400 hover:underline"
                  >
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
        </div>

        {/* Environment selector */}
        <div className="flex items-center gap-2">
          <button className="rounded-full bg-accent-500/15 px-3 py-1 text-xs font-medium text-accent-400">
            production
          </button>
          <button disabled className="rounded-full bg-surface-200 px-3 py-1 text-xs font-medium text-neutral-500 cursor-not-allowed">
            staging
            <span className="ml-1.5 rounded bg-neutral-500/20 px-1 py-0.5 text-[9px] text-neutral-600">Soon</span>
          </button>
          <button disabled className="rounded-full bg-surface-200 px-3 py-1 text-xs font-medium text-neutral-500 cursor-not-allowed">
            dev
            <span className="ml-1.5 rounded bg-neutral-500/20 px-1 py-0.5 text-[9px] text-neutral-600">Soon</span>
          </button>
        </div>

        {/* Tabs */}
        <div className="flex gap-1 border-b border-border">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`flex items-center gap-2 px-4 py-2.5 text-sm transition-colors border-b-2 ${
                activeTab === tab.key
                  ? "border-accent-500 text-accent-400 font-medium"
                  : "border-transparent text-neutral-500 hover:text-white"
              }`}
            >
              <tab.icon className="h-4 w-4" />
              {tab.label}
            </button>
          ))}
        </div>

        {/* Tab content */}
        {activeTab === "overview" && <OverviewTab app={app} />}
        {activeTab === "deployments" && <DeploymentsTab appId={appId} />}
        {activeTab === "releases" && <ReleasesTab appId={appId} />}
        {activeTab === "logs" && <LogsTab appId={appId} />}
        {activeTab === "databases" && <DatabasesTab appId={appId} />}
        {activeTab === "storage" && <StorageTab appId={appId} />}
        {activeTab === "auth" && <AuthTab appId={appId} />}
        {activeTab === "domains" && <DomainsTab appId={appId} />}
        {activeTab === "secrets" && <SecretsTab appId={appId} />}
        {activeTab === "env" && <EnvTab appId={appId} />}
      </div>
    </Shell>
  );
}

function OverviewTab({ app }: { app: DeployApp }) {
  const appType = app.app_type ?? "web";
  const isWeb = appType === "web";

  const details: { label: string; value: string; isLink?: boolean }[] = [
    { label: "Type", value: appType === "web" ? "Web Service" : appType === "worker" ? "Worker" : "Cron Job" },
    { label: "Framework", value: app.framework || "detecting..." },
    ...(isWeb ? [{ label: "Port", value: String(app.port || 8080) }] : []),
    ...(isWeb ? [{ label: "Subdomain", value: app.subdomain }] : []),
    ...(appType === "cron" && app.cron_schedule ? [{ label: "Schedule", value: app.cron_schedule }] : []),
    ...(app.command ? [{ label: "Command", value: app.command }] : []),
    ...(app.repo_url ? [{ label: "Repository", value: app.repo_url, isLink: true }] : []),
    { label: "Branch", value: app.branch || "main" },
    { label: "Status", value: app.status },
  ];

  return (
    <div className="space-y-4">
      {app.status === "sleeping" && (
        <div className="flex items-center gap-3 rounded-lg border border-indigo-500/20 bg-indigo-500/5 px-4 py-3">
          <span className="inline-block h-2 w-2 rounded-full bg-indigo-400 animate-pulse" />
          <div>
            <p className="text-sm font-medium text-indigo-300">This app is sleeping</p>
            <p className="text-xs text-indigo-400/70">
              It will wake automatically on the next HTTP request. Cold start takes ~3 seconds.
            </p>
          </div>
        </div>
      )}
    <div className="grid gap-4 md:grid-cols-2">
      <div className="rounded-lg border border-border bg-surface-100 p-5">
        <h3 className="mb-4 text-sm font-medium text-neutral-400">App Details</h3>
        <dl className="space-y-3">
          {details.map((d) => (
            <div key={d.label} className="flex justify-between text-sm">
              <dt className="text-neutral-500">{d.label}</dt>
              <dd className="font-mono text-neutral-200">
                {d.isLink ? (
                  <a
                    href={d.value}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-accent-400 hover:underline"
                  >
                    {d.value.replace("https://github.com/", "")}
                  </a>
                ) : (
                  d.value
                )}
              </dd>
            </div>
          ))}
        </dl>
      </div>

      <div className="rounded-lg border border-border bg-surface-100 p-5">
        <h3 className="mb-4 text-sm font-medium text-neutral-400">Quick Links</h3>
        <div className="space-y-2">
          {isWeb && app.url && (
            <a
              href={app.url}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 rounded-md px-3 py-2 text-sm text-accent-400 hover:bg-surface-200 transition-colors"
            >
              <Globe className="h-4 w-4" />
              Open App
            </a>
          )}
          <a
            href={app.repo_url}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 rounded-md px-3 py-2 text-sm text-neutral-300 hover:bg-surface-200 transition-colors"
          >
            <GitBranch className="h-4 w-4" />
            View Repository
          </a>
        </div>
      </div>
    </div>

    {/* Health Check (web only) */}
    {isWeb && app.health_status && (
      <div className="rounded-lg border border-border bg-surface-100 p-5">
        <div className="flex items-center gap-2 mb-4">
          <Heart className="h-4 w-4 text-neutral-400" />
          <h3 className="text-sm font-medium text-neutral-400">Health Check</h3>
        </div>
        <div className="grid grid-cols-4 gap-4">
          <div>
            <p className="text-xs text-neutral-500 mb-1">Status</p>
            <div className="flex items-center gap-2">
              <span className={`inline-block h-2 w-2 rounded-full ${
                app.health_status.status === "healthy" ? "bg-emerald-400" :
                app.health_status.status === "unhealthy" ? "bg-red-400" : "bg-neutral-500"
              }`} />
              <span className="text-sm font-medium text-white capitalize">{app.health_status.status}</span>
            </div>
          </div>
          <div>
            <p className="text-xs text-neutral-500 mb-1">Uptime</p>
            <p className="text-sm font-medium text-white">{app.health_status.uptime_percent}%</p>
          </div>
          <div>
            <p className="text-xs text-neutral-500 mb-1">Response Time</p>
            <p className="text-sm font-medium text-white">{app.health_status.response_time_ms}ms</p>
          </div>
          <div>
            <p className="text-xs text-neutral-500 mb-1">Last Check</p>
            <p className="text-sm font-medium text-white">
              {new Date(app.health_status.last_check).toLocaleTimeString()}
            </p>
          </div>
        </div>
        {app.health_check && (
          <div className="mt-3 rounded-md bg-surface-200 px-3 py-2 text-xs text-neutral-500">
            <span className="text-neutral-400">Path:</span> {app.health_check.path} &middot;{" "}
            <span className="text-neutral-400">Interval:</span> {app.health_check.interval_seconds}s &middot;{" "}
            <span className="text-neutral-400">Timeout:</span> {app.health_check.timeout_seconds}s
          </div>
        )}
      </div>
    )}
    </div>
  );
}

function DeploymentsTab({ appId }: { appId: string }) {
  const { appsDeploy } = getApi();
  const { toast } = useToast();
  const {
    data: deployments,
    loading,
    error,
    refetch,
  } = useApi(() => appsDeploy.listDeployments(appId), [appId]);

  // Auto-refresh when deployment events arrive for this app
  const handleDeployEvent = useCallback(
    (event: { app_id: string }) => {
      if (event.app_id === appId) refetch();
    },
    [appId, refetch]
  );
  useDeployEvents(handleDeployEvent);

  if (loading) return <PageWithTableSkeleton cols={5} rows={3} />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;

  const items = deployments?.items || [];

  if (items.length === 0) {
    return <EmptyState title="No deployments yet" description="Push to your repository to trigger a deployment." />;
  }

  const handleRollback = async (deployId: string) => {
    try {
      await appsDeploy.rollback(appId, deployId);
      refetch();
    } catch {
      toast("error", "Failed to rollback deployment");
    }
  };

  return (
    <div className="overflow-hidden rounded-lg border border-border">
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-border bg-surface-100">
            <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">ID</th>
            <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Status</th>
            <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Git SHA</th>
            <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Created</th>
            <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Actions</th>
          </tr>
        </thead>
        <tbody>
          {items.map((d: Deployment) => (
            <tr key={d.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
              <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">
                {d.id.slice(0, 8)}
              </td>
              <td className="whitespace-nowrap px-4 py-3">
                <StatusBadge status={d.status as "running" | "deploying" | "stopped" | "crashed"} />
              </td>
              <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                {d.git_sha?.slice(0, 8) || "—"}
              </td>
              <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-400">
                {new Date(d.created_at).toLocaleString()}
              </td>
              <td className="whitespace-nowrap px-4 py-3">
                {d.status !== "active" && (
                  <button
                    onClick={() => handleRollback(d.id)}
                    className="flex items-center gap-1 rounded px-2 py-1 text-xs text-neutral-400 hover:bg-surface-300 hover:text-white transition-colors"
                    title="Rollback to this deployment"
                  >
                    <RotateCcw className="h-3 w-3" />
                    Rollback
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function EnvTab({ appId }: { appId: string }) {
  const { appsDeploy } = getApi();
  const { toast } = useToast();
  const { data: envData, loading, error, refetch } = useApi(
    () => appsDeploy.getEnvVars(appId),
    [appId]
  );

  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");
  const [showValues, setShowValues] = useState(false);

  if (loading) return <PageWithTableSkeleton cols={3} rows={3} />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;

  const items = envData?.items || [];

  const handleAdd = async () => {
    if (!newKey.trim()) return;
    try {
      await appsDeploy.setEnvVars(appId, { [newKey.trim()]: newValue });
      setNewKey("");
      setNewValue("");
      refetch();
    } catch {
      toast("error", "Failed to set environment variable");
    }
  };

  const handleDelete = async (key: string) => {
    try {
      await appsDeploy.deleteEnvVar(appId, key);
      refetch();
    } catch {
      toast("error", "Failed to delete environment variable");
    }
  };

  return (
    <div className="space-y-4">
      {/* Add new env var */}
      <div className="flex items-end gap-3">
        <div className="flex-1">
          <label className="mb-1 block text-xs font-medium text-neutral-400">Key</label>
          <input
            type="text"
            value={newKey}
            onChange={(e) => setNewKey(e.target.value.toUpperCase())}
            placeholder="DATABASE_URL"
            className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none font-mono"
          />
        </div>
        <div className="flex-1">
          <label className="mb-1 block text-xs font-medium text-neutral-400">Value</label>
          <input
            type="text"
            value={newValue}
            onChange={(e) => setNewValue(e.target.value)}
            placeholder="postgres://..."
            className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none font-mono"
          />
        </div>
        <button
          onClick={handleAdd}
          disabled={!newKey.trim()}
          className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
        >
          <Plus className="h-4 w-4" />
          Add
        </button>
      </div>

      {/* Toggle show/hide values */}
      <div className="flex justify-end">
        <button
          onClick={() => setShowValues(!showValues)}
          className="flex items-center gap-1.5 text-xs text-neutral-500 hover:text-white transition-colors"
        >
          {showValues ? <EyeOff className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
          {showValues ? "Hide values" : "Show values"}
        </button>
      </div>

      {/* Env var list */}
      {items.length === 0 ? (
        <EmptyState
          title="No environment variables"
          description="Add environment variables to configure your app."
        />
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-100">
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Key</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Value</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500 w-20">Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((env: EnvVar) => (
                <tr key={env.key} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-xs font-medium text-accent-400">
                    {env.key}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">
                    {showValues ? env.value : "••••••••••"}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <button
                      onClick={() => handleDelete(env.key)}
                      className="rounded p-1 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 transition-colors"
                      title="Delete variable"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function LogsTab({ appId }: { appId: string }) {
  const { appsDeploy } = getApi();
  const {
    data: deployments,
    loading,
    error,
    refetch,
  } = useApi(() => appsDeploy.listDeployments(appId), [appId]);

  if (loading) return <PageWithTableSkeleton cols={1} rows={4} />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;

  const items = deployments?.items || [];

  if (items.length === 0) {
    return (
      <EmptyState
        title="No deployments yet"
        description="Push to your repository to trigger a build — logs will appear here."
      />
    );
  }

  // Show logs for the most recent deployment
  const latest: Deployment = items[0];

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2 text-xs text-neutral-500">
        <span>Showing logs for deployment</span>
        <code className="rounded bg-surface-200 px-1.5 py-0.5 font-mono text-neutral-300">
          {latest.id.slice(0, 8)}
        </code>
        <span
          className={
            latest.status === "active"
              ? "text-emerald-400"
              : latest.status === "failed"
                ? "text-red-400"
                : "text-amber-400"
          }
        >
          {latest.status}
        </span>
      </div>
      <LogsTabContent appId={appId} deploymentId={latest.id} />
    </div>
  );
}

function LogsTabContent({
  appId,
  deploymentId,
}: {
  appId: string;
  deploymentId: string;
}) {
  const { entries, streaming } = useDeployLogs(appId, deploymentId);
  return <BuildLogViewer entries={entries} streaming={streaming} />;
}

// ---- Databases Tab (Phase 3) ----

const engineBadge: Record<string, { label: string; className: string }> = {
  postgresql: { label: "P", className: "bg-blue-500/20 text-blue-400" },
  mysql: { label: "M", className: "bg-orange-500/20 text-orange-400" },
  redis: { label: "R", className: "bg-red-500/20 text-red-400" },
};

function DatabasesTab({ appId }: { appId: string }) {
  const { appsDeploy } = getApi();
  const { toast } = useToast();
  const { data: dbs, loading, error, refetch } = useApi(
    () => appsDeploy.listAppDatabases(appId),
    [appId]
  );

  const [showCreate, setShowCreate] = useState(false);
  const [engine, setEngine] = useState("postgresql");
  const [creating, setCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [revealedConn, setRevealedConn] = useState<Record<string, string>>({});
  const [copiedId, setCopiedId] = useState<string | null>(null);

  if (loading) return <PageWithTableSkeleton cols={6} rows={2} />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;

  const items: AppDatabase[] = dbs || [];

  const handleCreate = async () => {
    setCreating(true);
    try {
      await appsDeploy.createAppDatabase(appId, { engine });
      setShowCreate(false);
      setEngine("postgresql");
      refetch();
    } catch {
      toast("error", "Failed to create database");
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (dbId: string) => {
    if (deletingId) return;
    setDeletingId(dbId);
    try {
      await appsDeploy.deleteAppDatabase(appId, dbId);
      refetch();
    } catch {
      toast("error", "Failed to delete database");
    } finally {
      setDeletingId(null);
    }
  };

  const handleRevealConn = async (db: AppDatabase) => {
    if (revealedConn[db.id]) {
      setRevealedConn((prev) => {
        const next = { ...prev };
        delete next[db.id];
        return next;
      });
      return;
    }
    try {
      const result = await appsDeploy.getAppDatabase(appId, db.id);
      setRevealedConn((prev) => ({
        ...prev,
        [db.id]: result.connection_string || `${db.engine}://${db.db_user}:***@${db.host}:${db.port}/${db.db_name}`,
      }));
    } catch {
      // fallback
      setRevealedConn((prev) => ({
        ...prev,
        [db.id]: `${db.engine}://${db.db_user}:***@${db.host}:${db.port}/${db.db_name}`,
      }));
    }
  };

  const handleCopyConn = (dbId: string, connStr: string) => {
    navigator.clipboard.writeText(connStr);
    setCopiedId(dbId);
    setTimeout(() => setCopiedId(null), 2000);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-xs text-neutral-500">
          Managed databases attached to this app. Connection strings are auto-injected as environment variables.
        </p>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
        >
          <Plus className="h-4 w-4" />
          Add Database
        </button>
      </div>

      {showCreate && (
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <h3 className="mb-3 text-sm font-medium text-neutral-400">Provision Database</h3>
          <div className="flex items-end gap-3">
            <div className="flex-1">
              <label className="mb-1 block text-xs font-medium text-neutral-400">Engine</label>
              <select
                value={engine}
                onChange={(e) => setEngine(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
              >
                <option value="postgresql">PostgreSQL</option>
                <option value="mysql">MySQL</option>
                <option value="redis">Redis</option>
              </select>
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setShowCreate(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={creating}
                className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
              >
                {creating ? (
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                ) : (
                  <Plus className="h-4 w-4" />
                )}
                Create
              </button>
            </div>
          </div>
          <p className="mt-2 text-xs text-neutral-600">
            One database per engine per app. Connection string auto-injected as DATABASE_URL, REDIS_URL, or MYSQL_URL.
          </p>
        </div>
      )}

      {items.length === 0 ? (
        <EmptyState
          title="No databases"
          description="Add a managed database to get a connection string auto-injected into your app."
        />
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-100">
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Name</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Engine</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Status</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Size</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Connection</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500 w-24">Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((db) => {
                const badge = engineBadge[db.engine] ?? {
                  label: "?",
                  className: "bg-neutral-500/20 text-neutral-400",
                };
                return (
                  <tr key={db.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="whitespace-nowrap px-4 py-3 font-medium text-white">
                      {db.name}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3">
                      <span className={`inline-flex h-5 w-5 items-center justify-center rounded text-[10px] font-bold ${badge.className}`}>
                        {badge.label}
                      </span>
                      <span className="ml-2 text-xs capitalize text-neutral-300">{db.engine}</span>
                    </td>
                    <td className="whitespace-nowrap px-4 py-3">
                      <StatusBadge status={db.status as "ready" | "running" | "creating" | "stopped"} />
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-400">
                      <span className="font-mono">{db.size_mb}</span>
                      <span className="text-neutral-600"> / {db.max_size_mb} MB</span>
                    </td>
                    <td className="whitespace-nowrap px-4 py-3">
                      {revealedConn[db.id] ? (
                        <span className="flex items-center gap-2">
                          <code className="max-w-xs truncate text-xs text-neutral-300">{revealedConn[db.id]}</code>
                          <button
                            onClick={() => handleCopyConn(db.id, revealedConn[db.id])}
                            className="rounded p-0.5 text-neutral-500 hover:text-white transition-colors"
                            title="Copy"
                          >
                            {copiedId === db.id ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
                          </button>
                          <button
                            onClick={() => handleRevealConn(db)}
                            className="rounded p-0.5 text-neutral-500 hover:text-white transition-colors"
                            title="Hide"
                          >
                            <EyeOff className="h-3 w-3" />
                          </button>
                        </span>
                      ) : (
                        <button
                          onClick={() => handleRevealConn(db)}
                          className="flex items-center gap-1 text-xs text-neutral-500 hover:text-accent-400 transition-colors"
                        >
                          <Eye className="h-3 w-3" />
                          Show connection string
                        </button>
                      )}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3">
                      <button
                        onClick={() => handleDelete(db.id)}
                        disabled={deletingId === db.id}
                        className="rounded p-1 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 transition-colors disabled:opacity-50"
                        title="Delete database"
                      >
                        {deletingId === db.id ? (
                          <div className="h-3.5 w-3.5 animate-spin rounded-full border border-red-400 border-t-transparent" />
                        ) : (
                          <Trash2 className="h-3.5 w-3.5" />
                        )}
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Database Backups (Pro+ only) */}
      {items.length > 0 && (
        <div className="mt-6">
          <h3 className="mb-3 text-sm font-medium text-neutral-400 flex items-center gap-2">
            <Archive className="h-4 w-4" />
            Backups
            <span className="rounded bg-amber-500/10 px-1.5 py-0.5 text-[10px] font-medium text-amber-400">Pro+</span>
          </h3>
          {items.map((db) => (
            <DatabaseBackupsSection key={db.id} appId={appId} db={db} />
          ))}
        </div>
      )}
    </div>
  );
}

function DatabaseBackupsSection({ appId, db }: { appId: string; db: AppDatabase }) {
  const { appsDeploy } = getApi();
  const { toast } = useToast();
  const { data: backups, loading, refetch } = useApi(
    () => appsDeploy.listBackups(appId, db.id),
    [appId, db.id]
  );

  const [creatingBackup, setCreatingBackup] = useState(false);
  const [restoringId, setRestoringId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const handleCreateBackup = async () => {
    setCreatingBackup(true);
    try {
      await appsDeploy.createBackup(appId, db.id);
      refetch();
    } catch {
      toast("error", "Failed to create backup");
    } finally {
      setCreatingBackup(false);
    }
  };

  const handleRestore = async (backupId: string) => {
    setRestoringId(backupId);
    try {
      await appsDeploy.restoreBackup(appId, db.id, backupId);
      refetch();
    } catch {
      toast("error", "Failed to restore backup");
    } finally {
      setRestoringId(null);
    }
  };

  const handleDelete = async (backupId: string) => {
    setDeletingId(backupId);
    try {
      await appsDeploy.deleteBackup(appId, db.id, backupId);
      refetch();
    } catch {
      toast("error", "Failed to delete backup");
    } finally {
      setDeletingId(null);
    }
  };

  const backupItems: DatabaseBackup[] = backups || [];

  return (
    <div className="mb-4 rounded-lg border border-border bg-surface-100 p-4">
      <div className="flex items-center justify-between mb-3">
        <span className="text-sm text-white font-medium">{db.name}</span>
        <button
          onClick={handleCreateBackup}
          disabled={creatingBackup}
          className="flex items-center gap-1.5 rounded-md bg-accent-500/10 px-3 py-1.5 text-xs font-medium text-accent-400 hover:bg-accent-500/20 disabled:opacity-50 transition-colors"
        >
          {creatingBackup ? (
            <div className="h-3 w-3 animate-spin rounded-full border border-accent-400 border-t-transparent" />
          ) : (
            <Download className="h-3 w-3" />
          )}
          Create Backup
        </button>
      </div>

      {loading ? (
        <div className="text-xs text-neutral-500">Loading backups...</div>
      ) : backupItems.length === 0 ? (
        <div className="text-xs text-neutral-600">No backups yet. Create one to enable point-in-time recovery.</div>
      ) : (
        <div className="space-y-1.5">
          {backupItems.map((backup) => (
            <div
              key={backup.id}
              className="flex items-center justify-between rounded-md bg-surface-200 px-3 py-2 text-xs"
            >
              <div className="flex items-center gap-3">
                <span className={`inline-block h-1.5 w-1.5 rounded-full ${
                  backup.status === "completed" ? "bg-emerald-400" :
                  backup.status === "pending" || backup.status === "running" ? "bg-amber-400 animate-pulse" :
                  "bg-red-400"
                }`} />
                <span className="font-mono text-neutral-300">{backup.id.slice(0, 8)}</span>
                <span className="text-neutral-500 capitalize">{backup.type}</span>
                <span className="text-neutral-600">{backup.size_mb > 0 ? `${backup.size_mb} MB` : "—"}</span>
                <span className="text-neutral-600">{new Date(backup.created_at).toLocaleString()}</span>
              </div>
              <div className="flex items-center gap-1">
                {backup.status === "completed" && (
                  <button
                    onClick={() => handleRestore(backup.id)}
                    disabled={restoringId === backup.id}
                    className="flex items-center gap-1 rounded px-2 py-1 text-neutral-400 hover:bg-surface-300 hover:text-white transition-colors disabled:opacity-50"
                    title="Restore from this backup"
                  >
                    {restoringId === backup.id ? (
                      <div className="h-3 w-3 animate-spin rounded-full border border-neutral-400 border-t-transparent" />
                    ) : (
                      <RotateCw className="h-3 w-3" />
                    )}
                    Restore
                  </button>
                )}
                <button
                  onClick={() => handleDelete(backup.id)}
                  disabled={deletingId === backup.id}
                  className="rounded p-1 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 transition-colors disabled:opacity-50"
                  title="Delete backup"
                >
                  {deletingId === backup.id ? (
                    <div className="h-3 w-3 animate-spin rounded-full border border-red-400 border-t-transparent" />
                  ) : (
                    <Trash2 className="h-3 w-3" />
                  )}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ---- Storage Tab (Phase 3) ----

function StorageTab({ appId }: { appId: string }) {
  const { appsDeploy } = getApi();
  const { toast } = useToast();
  const { data: buckets, loading, error, refetch } = useApi(
    () => appsDeploy.listAppBuckets(appId),
    [appId]
  );

  const [showCreate, setShowCreate] = useState(false);
  const [bucketName, setBucketName] = useState("");
  const [access, setAccess] = useState("private");
  const [creating, setCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  if (loading) return <PageWithTableSkeleton cols={6} rows={2} />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;

  const items: AppBucket[] = buckets || [];

  const handleCreate = async () => {
    if (!bucketName.trim()) return;
    setCreating(true);
    try {
      await appsDeploy.createAppBucket(appId, { name: bucketName.trim(), access });
      setShowCreate(false);
      setBucketName("");
      setAccess("private");
      refetch();
    } catch {
      toast("error", "Failed to create storage bucket");
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (bucketId: string) => {
    if (deletingId) return;
    setDeletingId(bucketId);
    try {
      await appsDeploy.deleteAppBucket(appId, bucketId);
      refetch();
    } catch {
      toast("error", "Failed to delete storage bucket");
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-xs text-neutral-500">
          S3-compatible object storage. Endpoint and bucket name auto-injected as S3_ENDPOINT and S3_BUCKET env vars.
        </p>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
        >
          <Plus className="h-4 w-4" />
          Add Bucket
        </button>
      </div>

      {showCreate && (
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <h3 className="mb-3 text-sm font-medium text-neutral-400">Create Storage Bucket</h3>
          <div className="flex items-end gap-3">
            <div className="flex-1">
              <label className="mb-1 block text-xs font-medium text-neutral-400">Bucket Name</label>
              <input
                type="text"
                value={bucketName}
                onChange={(e) => setBucketName(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
                placeholder="my-uploads"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none font-mono"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Access</label>
              <select
                value={access}
                onChange={(e) => setAccess(e.target.value)}
                className="rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
              >
                <option value="private">Private</option>
                <option value="public">Public</option>
              </select>
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setShowCreate(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={!bucketName.trim() || creating}
                className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
              >
                {creating ? (
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                ) : (
                  <Plus className="h-4 w-4" />
                )}
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      {items.length === 0 ? (
        <EmptyState
          title="No storage buckets"
          description="Add an S3-compatible storage bucket for file uploads and static assets."
        />
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-100">
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Name</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Access</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Status</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Size</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Objects</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500 w-24">Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((bucket) => (
                <tr key={bucket.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                  <td className="whitespace-nowrap px-4 py-3">
                    <div>
                      <span className="font-medium text-white">{bucket.name}</span>
                      <div className="text-[11px] text-neutral-600 font-mono">{bucket.endpoint}</div>
                    </div>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <span className={`text-xs ${bucket.access === "public" ? "text-amber-400" : "text-neutral-400"}`}>
                      {bucket.access === "public" ? (
                        <><Unlock className="inline h-3 w-3 mr-1" />Public</>
                      ) : (
                        <><Lock className="inline h-3 w-3 mr-1" />Private</>
                      )}
                    </span>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <StatusBadge status={bucket.status as "active" | "running" | "creating" | "stopped"} />
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-400">
                    <span className="font-mono">{bucket.size_mb}</span>
                    <span className="text-neutral-600"> / {bucket.max_size_mb} MB</span>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">
                    {bucket.objects}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <button
                      onClick={() => handleDelete(bucket.id)}
                      disabled={deletingId === bucket.id}
                      className="rounded p-1 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 transition-colors disabled:opacity-50"
                      title="Delete bucket"
                    >
                      {deletingId === bucket.id ? (
                        <div className="h-3.5 w-3.5 animate-spin rounded-full border border-red-400 border-t-transparent" />
                      ) : (
                        <Trash2 className="h-3.5 w-3.5" />
                      )}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ---- Auth Tab (Phase 3) ----

function AuthTab({ appId }: { appId: string }) {
  const { appsDeploy } = getApi();
  const { toast } = useToast();
  const {
    data: authConfig,
    loading: configLoading,
    error: configError,
    refetch: refetchConfig,
  } = useApi(() => appsDeploy.getAuthStatus(appId), [appId]);

  const {
    data: usersData,
    loading: usersLoading,
    error: usersError,
    refetch: refetchUsers,
  } = useApi(() => appsDeploy.listAuthUsers(appId), [appId]);

  const [enabling, setEnabling] = useState(false);
  const [disabling, setDisabling] = useState(false);
  const [deletingUserId, setDeletingUserId] = useState<string | null>(null);

  if (configLoading) return <PageWithTableSkeleton cols={3} rows={2} />;
  if (configError) return <ErrorState message={configError} onRetry={refetchConfig} />;

  const config: AppAuthConfig = authConfig || { enabled: false, user_count: 0, max_users: 0 };
  const users: AppAuthUser[] = usersData?.users || [];

  const handleEnable = async () => {
    setEnabling(true);
    try {
      await appsDeploy.enableAuth(appId);
      refetchConfig();
      refetchUsers();
    } catch {
      toast("error", "Failed to enable authentication");
    } finally {
      setEnabling(false);
    }
  };

  const handleDisable = async () => {
    setDisabling(true);
    try {
      await appsDeploy.disableAuth(appId);
      refetchConfig();
    } catch {
      toast("error", "Failed to disable authentication");
    } finally {
      setDisabling(false);
    }
  };

  const handleDeleteUser = async (userId: string) => {
    setDeletingUserId(userId);
    try {
      await appsDeploy.deleteAuthUser(appId, userId);
      refetchUsers();
      refetchConfig();
    } catch {
      toast("error", "Failed to delete auth user");
    } finally {
      setDeletingUserId(null);
    }
  };

  if (!config.enabled) {
    return (
      <div className="space-y-4">
        <div className="rounded-lg border border-border bg-surface-100 p-8 text-center">
          <Shield className="mx-auto mb-3 h-8 w-8 text-neutral-600" />
          <h3 className="text-sm font-medium text-white mb-1">Built-in Auth</h3>
          <p className="text-xs text-neutral-500 mb-4">
            Enable user authentication for your app. Users can sign up and log in via the Zenith Auth API.
            JWT tokens are issued per-app with isolated user tables.
          </p>
          <button
            onClick={handleEnable}
            disabled={enabling}
            className="inline-flex items-center gap-2 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
          >
            {enabling ? (
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
            ) : (
              <Shield className="h-4 w-4" />
            )}
            Enable Auth
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Auth status card */}
      <div className="rounded-lg border border-border bg-surface-100 p-4">
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <span className="inline-block h-2 w-2 rounded-full bg-emerald-400" />
              <h3 className="text-sm font-medium text-white">Auth Enabled</h3>
            </div>
            <p className="text-xs text-neutral-500">
              <Users className="inline h-3 w-3 mr-1" />
              {config.user_count} / {config.max_users} users
            </p>
          </div>
          <button
            onClick={handleDisable}
            disabled={disabling}
            className="rounded-lg border border-red-500/30 px-3 py-1.5 text-xs text-red-400 hover:bg-red-500/10 disabled:opacity-50 transition-colors"
          >
            {disabling ? "Disabling..." : "Disable Auth"}
          </button>
        </div>
      </div>

      {/* SDK snippet */}
      <div className="rounded-lg border border-border bg-surface-100 p-4">
        <h3 className="mb-2 text-sm font-medium text-neutral-400">Quick Start</h3>
        <div className="rounded-md bg-surface-200 p-3 font-mono text-xs text-neutral-300">
          <div className="text-neutral-600">// Sign up a user</div>
          <div>
            <span className="text-amber-400">fetch</span>(
            <span className="text-emerald-400">&apos;/api/v1/apps/{appId}/auth/signup&apos;</span>, {"{"}
          </div>
          <div className="pl-4">method: <span className="text-emerald-400">&apos;POST&apos;</span>,</div>
          <div className="pl-4">body: JSON.stringify({"{"} email, password, name {"}"})</div>
          <div>{"}"}) <span className="text-neutral-600">// Returns JWT access_token</span></div>
        </div>
      </div>

      {/* Users table */}
      <div>
        <h3 className="mb-3 text-sm font-medium text-neutral-400">Registered Users</h3>
        {usersLoading ? (
          <PageWithTableSkeleton cols={4} rows={3} />
        ) : usersError ? (
          <ErrorState message={usersError} onRetry={refetchUsers} />
        ) : users.length === 0 ? (
          <EmptyState
            title="No users yet"
            description="Users will appear here when they sign up through your app's auth API."
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Email</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Name</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Status</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Joined</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500 w-20">Actions</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-accent-400">
                      {user.email}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-neutral-300">
                      {user.name}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3">
                      {user.verified ? (
                        <span className="inline-flex items-center gap-1 text-xs text-emerald-400">
                          <Check className="h-3 w-3" /> Verified
                        </span>
                      ) : (
                        <span className="text-xs text-neutral-500">Unverified</span>
                      )}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-500">
                      {new Date(user.created_at).toLocaleDateString()}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3">
                      <button
                        onClick={() => handleDeleteUser(user.id)}
                        disabled={deletingUserId === user.id}
                        className="rounded p-1 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 transition-colors disabled:opacity-50"
                        title="Delete user"
                      >
                        {deletingUserId === user.id ? (
                          <div className="h-3.5 w-3.5 animate-spin rounded-full border border-red-400 border-t-transparent" />
                        ) : (
                          <Trash2 className="h-3.5 w-3.5" />
                        )}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

// ---- Domains Tab (Phase 4) ----

function DomainsTab({ appId }: { appId: string }) {
  const { appsDeploy } = getApi();
  const { toast } = useToast();
  const { data: domains, loading, error, refetch } = useApi(
    () => appsDeploy.listDomains(appId),
    [appId]
  );

  const [showAdd, setShowAdd] = useState(false);
  const [domainInput, setDomainInput] = useState("");
  const [adding, setAdding] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  if (loading) return <PageWithTableSkeleton cols={4} rows={2} />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;

  const items: CustomDomain[] = domains || [];

  const handleAdd = async () => {
    if (!domainInput.trim()) return;
    setAdding(true);
    try {
      await appsDeploy.addDomain(appId, domainInput.trim());
      setShowAdd(false);
      setDomainInput("");
      refetch();
    } catch {
      toast("error", "Failed to add custom domain");
    } finally {
      setAdding(false);
    }
  };

  const handleDelete = async (domainId: string) => {
    setDeletingId(domainId);
    try {
      await appsDeploy.deleteDomain(appId, domainId);
      refetch();
    } catch {
      toast("error", "Failed to delete custom domain");
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs text-neutral-500">
            Custom domains for your app. Add a CNAME record pointing to your app&apos;s subdomain.
          </p>
          <p className="text-xs text-neutral-600 mt-1">
            Requires <span className="text-amber-400">Pro</span> plan or higher.
          </p>
        </div>
        <button
          onClick={() => setShowAdd(true)}
          className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
        >
          <Plus className="h-4 w-4" />
          Add Domain
        </button>
      </div>

      {showAdd && (
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <h3 className="mb-2 text-sm font-medium text-neutral-400">Add Custom Domain</h3>
          <p className="mb-3 text-xs text-neutral-600">
            First, add a CNAME record: <code className="text-neutral-400">your-domain.com → {appId}.freezenith.com</code>
          </p>
          <div className="flex items-end gap-3">
            <div className="flex-1">
              <label className="mb-1 block text-xs font-medium text-neutral-400">Domain</label>
              <input
                type="text"
                value={domainInput}
                onChange={(e) => setDomainInput(e.target.value.toLowerCase())}
                placeholder="app.example.com"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none font-mono"
              />
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setShowAdd(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleAdd}
                disabled={!domainInput.trim() || adding}
                className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
              >
                {adding ? (
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                ) : (
                  <Globe className="h-4 w-4" />
                )}
                Add
              </button>
            </div>
          </div>
        </div>
      )}

      {items.length === 0 ? (
        <EmptyState
          title="No custom domains"
          description="Add a custom domain to serve your app from your own URL."
        />
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-100">
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Domain</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Status</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">TLS</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Added</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500 w-20">Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((d) => (
                <tr key={d.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-sm text-accent-400">
                    {d.domain}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <span className={`inline-flex items-center gap-1.5 text-xs ${
                      d.status === "active" ? "text-emerald-400" :
                      d.status === "verified" ? "text-blue-400" :
                      d.status === "pending" ? "text-amber-400" :
                      "text-red-400"
                    }`}>
                      <span className={`h-1.5 w-1.5 rounded-full ${
                        d.status === "active" ? "bg-emerald-400" :
                        d.status === "verified" ? "bg-blue-400" :
                        d.status === "pending" ? "bg-amber-400 animate-pulse" :
                        "bg-red-400"
                      }`} />
                      {d.status}
                    </span>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs">
                    {d.tls_ready ? (
                      <span className="text-emerald-400 flex items-center gap-1">
                        <Lock className="h-3 w-3" /> Active
                      </span>
                    ) : (
                      <span className="text-neutral-500">Pending</span>
                    )}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-500">
                    {new Date(d.created_at).toLocaleDateString()}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <button
                      onClick={() => handleDelete(d.id)}
                      disabled={deletingId === d.id}
                      className="rounded p-1 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 transition-colors disabled:opacity-50"
                      title="Remove domain"
                    >
                      {deletingId === d.id ? (
                        <div className="h-3.5 w-3.5 animate-spin rounded-full border border-red-400 border-t-transparent" />
                      ) : (
                        <Trash2 className="h-3.5 w-3.5" />
                      )}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ---- Secrets Tab (Phase 5) ----

function SecretsTab({ appId }: { appId: string }) {
  const { appsDeploy } = getApi();
  const { toast } = useToast();
  const { data: secretsData, loading, error, refetch } = useApi(
    () => appsDeploy.listSecrets(appId),
    [appId]
  );

  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");
  const [adding, setAdding] = useState(false);
  const [revealedValues, setRevealedValues] = useState<Record<string, string>>({});
  const [revealingKey, setRevealingKey] = useState<string | null>(null);
  const [deletingKey, setDeletingKey] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  if (loading) return <PageWithTableSkeleton cols={3} rows={3} />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;

  const secrets = secretsData?.secrets || [];

  const handleAdd = async () => {
    if (!newKey.trim() || !newValue.trim()) return;
    setAdding(true);
    try {
      await appsDeploy.setSecret(appId, newKey.trim(), newValue);
      setNewKey("");
      setNewValue("");
      refetch();
    } catch {
      toast("error", "Failed to set secret");
    } finally {
      setAdding(false);
    }
  };

  const handleReveal = async (key: string) => {
    if (revealedValues[key]) {
      // Toggle off
      setRevealedValues((prev) => {
        const next = { ...prev };
        delete next[key];
        return next;
      });
      return;
    }
    setRevealingKey(key);
    try {
      const result = await appsDeploy.getSecretValue(appId, key);
      setRevealedValues((prev) => ({ ...prev, [key]: result.value }));
    } catch {
      toast("error", "Failed to reveal secret value");
    } finally {
      setRevealingKey(null);
    }
  };

  const handleCopy = (key: string, value: string) => {
    navigator.clipboard.writeText(value);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const handleDelete = async (key: string) => {
    if (deletingKey) return;
    setDeletingKey(key);
    try {
      await appsDeploy.deleteSecret(appId, key);
      setRevealedValues((prev) => {
        const next = { ...prev };
        delete next[key];
        return next;
      });
      refetch();
    } catch {
      toast("error", "Failed to delete secret");
    } finally {
      setDeletingKey(null);
    }
  };

  return (
    <div className="space-y-4">
      {/* Add new secret */}
      <div className="rounded-lg border border-border bg-surface-100 p-4">
        <h3 className="mb-3 text-sm font-medium text-neutral-400">Add Secret</h3>
        <div className="flex items-end gap-3">
          <div className="flex-1">
            <label className="mb-1 block text-xs font-medium text-neutral-400">Key</label>
            <input
              type="text"
              value={newKey}
              onChange={(e) => setNewKey(e.target.value.toUpperCase().replace(/[^A-Z0-9_]/g, ""))}
              placeholder="MY_SECRET_KEY"
              className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none font-mono"
            />
          </div>
          <div className="flex-1">
            <label className="mb-1 block text-xs font-medium text-neutral-400">Value</label>
            <input
              type="password"
              value={newValue}
              onChange={(e) => setNewValue(e.target.value)}
              placeholder="secret value..."
              className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none font-mono"
            />
          </div>
          <button
            onClick={handleAdd}
            disabled={!newKey.trim() || !newValue.trim() || adding}
            className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
          >
            {adding ? (
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
            ) : (
              <Plus className="h-4 w-4" />
            )}
            Add
          </button>
        </div>
        <p className="mt-2 text-xs text-neutral-600">
          Secrets are encrypted with AES-256-GCM before storage. Values are never logged.
        </p>
      </div>

      {/* Secrets list */}
      {secrets.length === 0 ? (
        <EmptyState
          title="No secrets"
          description="Add encrypted secrets to store sensitive configuration like API keys and database credentials."
        />
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-100">
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                  <Lock className="inline h-3 w-3 mr-1" />
                  Key
                </th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Value</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Created</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500 w-32">Actions</th>
              </tr>
            </thead>
            <tbody>
              {secrets.map((secret: Secret) => (
                <tr key={secret.key} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-xs font-medium text-accent-400">
                    {secret.key}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">
                    {revealedValues[secret.key] ? (
                      <span className="flex items-center gap-2">
                        <span className="max-w-xs truncate">{revealedValues[secret.key]}</span>
                        <button
                          onClick={() => handleCopy(secret.key, revealedValues[secret.key])}
                          className="rounded p-0.5 text-neutral-500 hover:text-white transition-colors"
                          title="Copy value"
                        >
                          {copiedKey === secret.key ? (
                            <Check className="h-3 w-3 text-emerald-400" />
                          ) : (
                            <Copy className="h-3 w-3" />
                          )}
                        </button>
                      </span>
                    ) : (
                      "••••••••••••••••"
                    )}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-500">
                    {new Date(secret.created_at).toLocaleDateString()}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => handleReveal(secret.key)}
                        disabled={revealingKey === secret.key}
                        className="flex items-center gap-1 rounded px-2 py-1 text-xs text-neutral-400 hover:bg-surface-300 hover:text-white transition-colors disabled:opacity-50"
                        title={revealedValues[secret.key] ? "Hide value" : "Reveal value"}
                      >
                        {revealingKey === secret.key ? (
                          <div className="h-3 w-3 animate-spin rounded-full border border-neutral-400 border-t-transparent" />
                        ) : revealedValues[secret.key] ? (
                          <EyeOff className="h-3 w-3" />
                        ) : (
                          <Unlock className="h-3 w-3" />
                        )}
                        {revealedValues[secret.key] ? "Hide" : "Reveal"}
                      </button>
                      <button
                        onClick={() => handleDelete(secret.key)}
                        disabled={deletingKey === secret.key}
                        className="rounded p-1 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 transition-colors disabled:opacity-50"
                        title="Delete secret"
                      >
                        {deletingKey === secret.key ? (
                          <div className="h-3.5 w-3.5 animate-spin rounded-full border border-red-400 border-t-transparent" />
                        ) : (
                          <Trash2 className="h-3.5 w-3.5" />
                        )}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ---- Releases Tab (Phase 6) ----

function ReleasesTab({ appId }: { appId: string }) {
  const { appsDeploy } = getApi();
  const { toast } = useToast();
  const { data: releasesData, loading, error, refetch } = useApi(
    () => appsDeploy.listReleases(appId),
    [appId]
  );

  const [deployingId, setDeployingId] = useState<string | null>(null);
  const [deployedId, setDeployedId] = useState<string | null>(null);

  if (loading) return <PageWithTableSkeleton cols={5} rows={4} />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;

  const releases = releasesData?.releases || [];

  if (releases.length === 0) {
    return (
      <EmptyState
        title="No releases yet"
        description="Push an image using zenith-actions in your CI pipeline to register releases here."
      />
    );
  }

  const handleDeploy = async (releaseId: string) => {
    setDeployingId(releaseId);
    setDeployedId(null);
    try {
      await appsDeploy.deployRelease(appId, releaseId);
      setDeployedId(releaseId);
      setTimeout(() => setDeployedId(null), 3000);
    } catch {
      toast("error", "Failed to deploy release");
    } finally {
      setDeployingId(null);
    }
  };

  return (
    <div className="space-y-3">
      <p className="text-xs text-neutral-500">
        Image versions pushed by your CI pipeline. Click Deploy to roll out a specific version.
      </p>
      <div className="overflow-hidden rounded-lg border border-border">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-border bg-surface-100">
              <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Image</th>
              <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Git SHA</th>
              <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Branch</th>
              <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Message</th>
              <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Date</th>
              <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500 w-28">Actions</th>
            </tr>
          </thead>
          <tbody>
            {releases.map((rel: Release, idx: number) => (
              <tr
                key={rel.id}
                className={`border-b border-border last:border-0 hover:bg-surface-200 transition-colors ${
                  idx === 0 ? "bg-surface-100/50" : ""
                }`}
              >
                <td className="whitespace-nowrap px-4 py-3">
                  <div className="flex items-center gap-2">
                    {idx === 0 && (
                      <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-medium text-emerald-400">
                        latest
                      </span>
                    )}
                    <code className="text-xs text-neutral-300">
                      {rel.image.split(":").pop() || rel.image}
                    </code>
                  </div>
                </td>
                <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                  {rel.git_sha?.slice(0, 8) || "—"}
                </td>
                <td className="whitespace-nowrap px-4 py-3">
                  <span className="flex items-center gap-1 text-xs text-neutral-400">
                    <GitBranch className="h-3 w-3" />
                    {rel.branch || "main"}
                  </span>
                </td>
                <td className="max-w-xs truncate px-4 py-3 text-xs text-neutral-300" title={rel.message}>
                  {rel.message || "—"}
                </td>
                <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-500">
                  {new Date(rel.created_at).toLocaleString()}
                </td>
                <td className="whitespace-nowrap px-4 py-3">
                  {deployedId === rel.id ? (
                    <span className="flex items-center gap-1 text-xs text-emerald-400 font-medium">
                      <Check className="h-3 w-3" />
                      Triggered
                    </span>
                  ) : (
                    <button
                      onClick={() => handleDeploy(rel.id)}
                      disabled={deployingId !== null}
                      className="flex items-center gap-1 rounded-md bg-accent-500/10 px-3 py-1.5 text-xs font-medium text-accent-400 hover:bg-accent-500/20 disabled:opacity-50 transition-colors"
                    >
                      {deployingId === rel.id ? (
                        <div className="h-3 w-3 animate-spin rounded-full border border-accent-400 border-t-transparent" />
                      ) : (
                        <Rocket className="h-3 w-3" />
                      )}
                      Deploy
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
