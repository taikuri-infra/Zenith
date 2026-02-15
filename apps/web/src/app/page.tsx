import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ProgressBar } from "@/components/progress-bar";
import { mockApps, mockDatabases, mockPlanets, projectName } from "@/lib/mock-data";
import { ArrowUpRight } from "lucide-react";
import Link from "next/link";

export default function OverviewPage() {
  const runningApps = mockApps.filter((a) => a.status === "running").length;
  const totalCpu = mockPlanets.reduce((s, p) => s + p.cpuCores, 0);
  const totalRam = mockPlanets.reduce((s, p) => s + p.ramGb, 0);
  const avgCpu = Math.round(mockPlanets.reduce((s, p) => s + p.cpuPercent, 0) / mockPlanets.length);
  const avgRam = Math.round(mockPlanets.reduce((s, p) => s + p.ramPercent, 0) / mockPlanets.length);

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">{projectName}</h1>
          <p className="text-sm text-neutral-500">Project overview</p>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-4 gap-4">
          <StatCard label="Apps" value={`${runningApps}/${mockApps.length}`} sub={`${runningApps} running`} />
          <StatCard label="Databases" value={mockDatabases.length} sub="all healthy" />
          <StatCard label="Planets" value={mockPlanets.length} sub={`${totalCpu} vCPU, ${totalRam}GB RAM`} />
          <StatCard label="Domains" value={3} sub="SSL active" />
        </div>

        {/* Resource usage */}
        <div className="grid grid-cols-2 gap-4">
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="mb-2 text-xs font-medium text-neutral-500">CPU Usage</p>
            <ProgressBar percent={avgCpu} label={`${avgCpu}% avg`} size="md" />
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="mb-2 text-xs font-medium text-neutral-500">Memory Usage</p>
            <ProgressBar percent={avgRam} label={`${avgRam}% avg`} size="md" />
          </div>
        </div>

        {/* Apps table */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Apps</h2>
            <Link href="/apps" className="flex items-center gap-1 text-xs text-neutral-500 hover:text-white">
              View all <ArrowUpRight className="h-3 w-3" />
            </Link>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Instances</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">CPU</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Memory</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Deploy</th>
                </tr>
              </thead>
              <tbody>
                {mockApps.map((app) => (
                  <tr key={app.name} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="px-4 py-3">
                      <Link href={`/apps/${app.name}`} className="font-medium text-white hover:text-accent-400 transition-colors">
                        {app.name}
                      </Link>
                      {app.domain && <span className="ml-2 text-xs text-neutral-500">{app.domain}</span>}
                    </td>
                    <td className="px-4 py-3"><StatusBadge status={app.status} /></td>
                    <td className="px-4 py-3 text-neutral-300">{app.replicas.ready}/{app.replicas.total}</td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">{app.cpu}</td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">{app.memory}</td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{app.lastDeploy}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Databases + Planets side by side */}
        <div className="grid grid-cols-2 gap-6">
          <section>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-medium text-white">Databases</h2>
              <Link href="/databases" className="flex items-center gap-1 text-xs text-neutral-500 hover:text-white">
                View all <ArrowUpRight className="h-3 w-3" />
              </Link>
            </div>
            <div className="space-y-2">
              {mockDatabases.map((db) => (
                <Link
                  key={db.name}
                  href={`/databases/${db.name}`}
                  className="flex items-center justify-between rounded-lg border border-border bg-surface-100 p-3 transition-colors hover:border-border-hover"
                >
                  <div>
                    <span className="text-sm font-medium text-white">{db.name}</span>
                    <span className="ml-2 text-xs text-neutral-500 capitalize">{db.engine} {db.version}</span>
                  </div>
                  <div className="text-right">
                    <span className="text-xs text-neutral-400">{db.storageUsed} / {db.storageTotal}</span>
                  </div>
                </Link>
              ))}
            </div>
          </section>

          <section>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-medium text-white">Planets</h2>
              <Link href="/planets" className="flex items-center gap-1 text-xs text-neutral-500 hover:text-white">
                View all <ArrowUpRight className="h-3 w-3" />
              </Link>
            </div>
            <div className="space-y-2">
              {mockPlanets.slice(0, 3).map((p) => (
                <div key={p.name} className="rounded-lg border border-border bg-surface-100 p-3">
                  <div className="mb-2 flex items-center justify-between">
                    <span className="text-sm font-medium text-white">{p.name}</span>
                    <span className="text-xs text-neutral-500">{p.size}</span>
                  </div>
                  <div className="space-y-1.5">
                    <div className="flex items-center gap-2">
                      <span className="w-8 text-[10px] text-neutral-500">CPU</span>
                      <ProgressBar percent={p.cpuPercent} label={`${p.cpuPercent}%`} />
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="w-8 text-[10px] text-neutral-500">RAM</span>
                      <ProgressBar percent={p.ramPercent} label={`${p.ramPercent}%`} />
                    </div>
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
