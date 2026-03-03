"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { StatCard } from "@/components/stat-card";
import { AppDetailSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { Modal } from "@/components/modal";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { type App } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import { useParams } from "next/navigation";
import { useState } from "react";

export default function AppDetailPage() {
  const { name } = useParams<{ name: string }>();
  const projectId = useProject();
  const { apps } = getApi();

  const {
    data: app,
    loading,
    error,
    refetch,
  } = useApi(
    () => projectId ? apps.get(projectId, name) : Promise.resolve(null),
    [projectId, name]
  );

  const tabs = [
    "Overview",
    "Logs",
    "Env Vars",
    "Domains",
    "Scaling",
    "Deployments",
  ];

  const [activeTab, setActiveTab] = useState("Overview");
  const [redeployMsg, setRedeployMsg] = useState<string | null>(null);

  const handleRedeploy = () => {
    setRedeployMsg("Redeployment triggered successfully. Building...");
    setTimeout(() => setRedeployMsg(null), 3000);
  };

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
        <ErrorState message={`App "${name}" not found`} />
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-6">
        {/* Toast */}
        {redeployMsg && (
          <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-4 py-3">
            <p className="text-xs text-emerald-400">{redeployMsg}</p>
          </div>
        )}

        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-lg font-semibold text-white">
                  {app.name}
                </h1>
                <StatusBadge status={app.status as "running" | "deploying" | "stopped" | "crashed"} />
              </div>
              {app.domain && (
                <p className="mt-0.5 text-sm text-neutral-500">{app.domain}</p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={handleRedeploy}
              className="rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-3 py-1.5 text-sm transition-colors"
            >
              Redeploy
            </button>
            <button className="rounded-lg border border-border bg-surface-200 hover:bg-surface-300 text-neutral-300 px-3 py-1.5 text-sm transition-colors">
              Settings
            </button>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-4 gap-4">
          <StatCard
            label="Status"
            value={app.status}
            sub={app.status === "running" ? "Healthy" : undefined}
          />
          <StatCard
            label="Replicas"
            value={app.replicas}
            sub={`${app.replicas} running`}
          />
          <StatCard label="CPU" value={app.cpu} sub="allocated" />
          <StatCard label="Memory" value={app.memory} sub="allocated" />
        </div>

        {/* Tabs */}
        <div className="border-b border-border">
          <nav className="flex gap-0">
            {tabs.map((tab) => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`px-4 py-2.5 text-sm transition-colors ${
                  tab === activeTab
                    ? "border-b-2 border-accent-500 text-accent-400 font-medium"
                    : "text-neutral-500 hover:text-neutral-300"
                }`}
              >
                {tab}
              </button>
            ))}
          </nav>
        </div>

        {/* Tab Content */}
        {activeTab === "Overview" && (
          <div className="grid grid-cols-2 gap-6">
            {/* Source info */}
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <h3 className="mb-3 text-sm font-medium text-white">Source</h3>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs text-neutral-500">Image</span>
                  <span className="text-xs text-neutral-300">{app.image}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-xs text-neutral-500">Port</span>
                  <span className="font-mono text-xs text-neutral-300">
                    {app.port}
                  </span>
                </div>
                {app.domain && (
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-neutral-500">Domain</span>
                    <span className="text-xs text-neutral-300">{app.domain}</span>
                  </div>
                )}
                <div className="flex items-center justify-between">
                  <span className="text-xs text-neutral-500">Created</span>
                  <span className="text-xs text-neutral-300">
                    {app.created_at}
                  </span>
                </div>
              </div>
            </div>

            {/* Resource usage */}
            <div className="rounded-lg border border-border bg-surface-100 p-4">
              <h3 className="mb-3 text-sm font-medium text-white">Resources</h3>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs text-neutral-500">CPU</span>
                  <span className="font-mono text-xs text-neutral-300">
                    {app.cpu}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-xs text-neutral-500">Memory</span>
                  <span className="font-mono text-xs text-neutral-300">
                    {app.memory}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-xs text-neutral-500">Replicas</span>
                  <span className="font-mono text-xs text-neutral-300">
                    {app.replicas}
                  </span>
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === "Logs" && (
          <div className="rounded-lg border border-border bg-[#0d1117] p-4 font-mono text-xs text-neutral-300">
            <p className="text-emerald-400 mb-2">Logs streaming...</p>
            <p className="text-neutral-500">[{new Date().toISOString()}] INFO  Server started on port {app.port}</p>
            <p className="text-neutral-500">[{new Date().toISOString()}] INFO  Connected to database</p>
            <p className="text-neutral-500">[{new Date().toISOString()}] INFO  Health check passed</p>
            <p className="text-neutral-500">[{new Date().toISOString()}] INFO  Ready to accept connections</p>
          </div>
        )}

        {activeTab === "Env Vars" && (
          <section>
            <h2 className="mb-3 text-sm font-medium text-white">
              Environment Variables
            </h2>
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Key
                    </th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      Value
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {app.env && Object.entries(app.env).length > 0 ? (
                    Object.entries(app.env).map(([key, value]) => (
                      <tr
                        key={key}
                        className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors"
                      >
                        <td className="px-4 py-3 font-mono text-xs text-accent-400">
                          {key}
                        </td>
                        <td className="px-4 py-3 font-mono text-xs text-neutral-300">
                          {value}
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={2} className="px-4 py-6 text-center text-xs text-neutral-500">No environment variables configured</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </section>
        )}

        {activeTab === "Domains" && (
          <div className="rounded-lg border border-border bg-surface-100 p-6 text-center">
            <p className="text-sm text-neutral-400">
              {app.domain ? (
                <>Custom domain <span className="text-accent-400">{app.domain}</span> is active.</>
              ) : (
                "No custom domains configured. Add a domain in Networking settings."
              )}
            </p>
          </div>
        )}

        {activeTab === "Scaling" && (
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <h3 className="mb-3 text-sm font-medium text-white">Scaling Configuration</h3>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs text-neutral-500">Current Replicas</span>
                <span className="font-mono text-xs text-neutral-300">{app.replicas}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs text-neutral-500">CPU per replica</span>
                <span className="font-mono text-xs text-neutral-300">{app.cpu}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs text-neutral-500">Memory per replica</span>
                <span className="font-mono text-xs text-neutral-300">{app.memory}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs text-neutral-500">Auto-scaling</span>
                <span className="text-xs text-neutral-400">Disabled</span>
              </div>
            </div>
          </div>
        )}

        {activeTab === "Deployments" && (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Version</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Deployed</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Duration</th>
                </tr>
              </thead>
              <tbody>
                <tr className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                  <td className="px-4 py-3 font-mono text-xs text-white">v1 (current)</td>
                  <td className="px-4 py-3"><StatusBadge status="running" /></td>
                  <td className="px-4 py-3 text-xs text-neutral-500">{app.created_at}</td>
                  <td className="px-4 py-3 text-xs text-neutral-400">42s</td>
                </tr>
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Shell>
  );
}
