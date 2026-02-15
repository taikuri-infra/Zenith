import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { StatCard } from "@/components/stat-card";
import { mockApps, mockDeployments } from "@/lib/mock-data";
import { notFound } from "next/navigation";
import Link from "next/link";

export default async function AppDetailPage({
  params,
}: {
  params: Promise<{ name: string }>;
}) {
  const { name } = await params;
  const app = mockApps.find((a) => a.name === name);

  if (!app) {
    notFound();
  }

  const tabs = ["Overview", "Logs", "Env Vars", "Domains", "Scaling", "Deployments"];

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-lg font-semibold text-white">{app.name}</h1>
                <StatusBadge status={app.status} />
              </div>
              {app.domain && (
                <p className="mt-0.5 text-sm text-neutral-500">{app.domain}</p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button className="rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-3 py-1.5 text-sm transition-colors">
              Redeploy
            </button>
            <button className="rounded-lg border border-border bg-surface-200 hover:bg-surface-300 text-neutral-300 px-3 py-1.5 text-sm transition-colors">
              Settings
            </button>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-4 gap-4">
          <StatCard label="Status" value={app.status} sub={app.status === "running" ? "Healthy" : undefined} />
          <StatCard
            label="Instances"
            value={`${app.replicas.ready}/${app.replicas.total}`}
            sub={`${app.replicas.ready} ready`}
          />
          <StatCard
            label="Avg Response"
            value={app.avgResponse ?? "--"}
            sub="p50 latency"
          />
          <StatCard
            label="Requests/min"
            value={app.reqPerMin ?? 0}
            sub="current load"
          />
        </div>

        {/* Tabs */}
        <div className="border-b border-border">
          <nav className="flex gap-0">
            {tabs.map((tab) => (
              <button
                key={tab}
                className={`px-4 py-2.5 text-sm transition-colors ${
                  tab === "Overview"
                    ? "border-b-2 border-accent-500 text-accent-400 font-medium"
                    : "text-neutral-500 hover:text-neutral-300"
                }`}
              >
                {tab}
              </button>
            ))}
          </nav>
        </div>

        {/* Overview tab content */}
        <div className="grid grid-cols-2 gap-6">
          {/* Source info */}
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <h3 className="mb-3 text-sm font-medium text-white">Source</h3>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs text-neutral-500">Repository</span>
                <span className="text-xs text-neutral-300">{app.source}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs text-neutral-500">Branch</span>
                <span className="text-xs text-neutral-300">{app.branch}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs text-neutral-500">Port</span>
                <span className="font-mono text-xs text-neutral-300">{app.port}</span>
              </div>
              {app.domain && (
                <div className="flex items-center justify-between">
                  <span className="text-xs text-neutral-500">Domain</span>
                  <span className="text-xs text-neutral-300">{app.domain}</span>
                </div>
              )}
            </div>
          </div>

          {/* Resource usage */}
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <h3 className="mb-3 text-sm font-medium text-white">Resources</h3>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs text-neutral-500">CPU</span>
                <span className="font-mono text-xs text-neutral-300">{app.cpu}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs text-neutral-500">Memory</span>
                <span className="font-mono text-xs text-neutral-300">{app.memory}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs text-neutral-500">Last Deploy</span>
                <span className="text-xs text-neutral-300">{app.lastDeploy}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Recent Deployments */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Recent Deployments</h2>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Commit</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Message</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Time</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Duration</th>
                </tr>
              </thead>
              <tbody>
                {mockDeployments.map((dep) => (
                  <tr
                    key={dep.id}
                    className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors"
                  >
                    <td className="px-4 py-3 font-mono text-xs text-accent-400">{dep.commit}</td>
                    <td className="px-4 py-3 text-neutral-300">{dep.message}</td>
                    <td className="px-4 py-3">
                      <StatusBadge status={dep.status} />
                    </td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{dep.createdAt}</td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{dep.duration}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </Shell>
  );
}
