import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ProgressBar } from "@/components/progress-bar";
import {
  mockClusters,
  mockAuditLog,
  mockModules,
  mockPlatformUpdate,
} from "@/lib/mock-data";
import { ArrowUpRight } from "lucide-react";
import Link from "next/link";

const updatesAvailable = mockModules.filter(
  (m) => m.status === "update_available"
).length;

export default function DashboardPage() {
  return (
    <Shell>
      <div className="space-y-6">
        {/* Page title */}
        <h1 className="text-lg font-semibold text-white">Platform Overview</h1>

        {/* Stat cards */}
        <div className="grid grid-cols-4 gap-4">
          <StatCard label="Clusters" value={mockClusters.length} sub="all healthy" />
          <StatCard label="Tenants" value={47} sub="12 active today" />
          <StatCard label="Monthly Cost" value="€127.40" sub="Hetzner Cloud" />
          <StatCard
            label="Updates"
            value={updatesAvailable}
            sub={`${updatesAvailable} available`}
            alert={updatesAvailable > 0}
          />
        </div>

        {/* Clusters table */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Clusters</h2>
            <Link
              href="/clusters"
              className="flex items-center gap-1 text-xs text-neutral-500 transition-colors hover:text-white"
            >
              View all <ArrowUpRight className="h-3 w-3" />
            </Link>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">K8s</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Nodes</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">CPU</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">RAM</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                </tr>
              </thead>
              <tbody>
                {mockClusters.map((cluster) => (
                  <tr
                    key={cluster.name}
                    className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                  >
                    <td className="px-4 py-3">
                      <Link href={`/clusters/${cluster.name}`} className="font-medium text-white hover:text-accent-400 transition-colors">
                        {cluster.name}
                      </Link>
                      <span className="ml-2 text-xs text-neutral-500">{cluster.region}</span>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                      {cluster.k8sVersion}
                      {cluster.upgradeAvailable && (
                        <span className="ml-1.5 text-amber-400">⚠</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-neutral-300">{cluster.nodes}</td>
                    <td className="w-36 px-4 py-3">
                      <ProgressBar percent={cluster.cpuPercent} label={`${cluster.cpuPercent}%`} />
                    </td>
                    <td className="w-36 px-4 py-3">
                      <ProgressBar percent={cluster.ramPercent} label={`${cluster.ramPercent}%`} />
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={cluster.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Two columns: Updates + Activity */}
        <div className="grid grid-cols-2 gap-6">
          {/* Available Updates */}
          <section>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-medium text-white">Available Updates</h2>
              <Link
                href="/updates"
                className="flex items-center gap-1 text-xs text-neutral-500 transition-colors hover:text-white"
              >
                View all <ArrowUpRight className="h-3 w-3" />
              </Link>
            </div>
            <div className="space-y-2">
              {/* Platform update */}
              <div className="rounded-lg border border-accent-600/30 bg-accent-600/5 p-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="rounded bg-accent-600/20 px-1.5 py-0.5 text-[10px] font-medium text-accent-400">
                      NEW
                    </span>
                    <span className="text-sm font-medium text-white">
                      Zenith {mockPlatformUpdate.version}
                    </span>
                  </div>
                  <span className="text-xs text-neutral-500">
                    Current: {mockPlatformUpdate.current}
                  </span>
                </div>
              </div>

              {/* Module updates */}
              {mockModules
                .filter((m) => m.status === "update_available")
                .map((mod) => (
                  <div
                    key={mod.name}
                    className="flex items-center justify-between rounded-lg border border-border bg-surface-100 p-3"
                  >
                    <div>
                      <span className="text-sm text-white">{mod.name}</span>
                      <span className="ml-2 text-xs text-neutral-500">
                        {mod.installed} → {mod.latest}
                      </span>
                    </div>
                    <Link
                      href={`/modules/${encodeURIComponent(mod.name)}`}
                      className="text-xs text-accent-400 hover:text-accent-300"
                    >
                      View
                    </Link>
                  </div>
                ))}
            </div>
          </section>

          {/* Recent Activity */}
          <section>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-medium text-white">Recent Activity</h2>
              <Link
                href="/audit"
                className="flex items-center gap-1 text-xs text-neutral-500 transition-colors hover:text-white"
              >
                View all <ArrowUpRight className="h-3 w-3" />
              </Link>
            </div>
            <div className="space-y-0 rounded-lg border border-border bg-surface-100">
              {mockAuditLog.map((entry, i) => (
                <div
                  key={i}
                  className="flex items-start gap-3 border-b border-border px-3 py-2.5 last:border-0"
                >
                  <span className="mt-px font-mono text-xs text-neutral-500">{entry.time}</span>
                  <div className="min-w-0 flex-1">
                    <span className="text-sm text-neutral-300">
                      <span className="font-medium text-white">{entry.actor}</span>{" "}
                      {entry.action}
                    </span>
                    {entry.cluster && (
                      <span className="ml-1.5 text-xs text-neutral-500">
                        on {entry.cluster}
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </section>
        </div>
      </div>
    </Shell>
  );
}
