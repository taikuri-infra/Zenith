import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { StatCard } from "@/components/stat-card";
import { ProgressBar } from "@/components/progress-bar";
import { mockDatabases } from "@/lib/mock-data";
import { notFound } from "next/navigation";
import Link from "next/link";

function parseStoragePercent(used: string, total: string): number {
  const parseNum = (s: string) => parseFloat(s.replace(/[^0-9.]/g, ""));
  const u = parseNum(used);
  const t = parseNum(total);
  return t > 0 ? Math.round((u / t) * 100) : 0;
}

const engineColors: Record<string, string> = {
  postgresql: "bg-blue-500/20 text-blue-400",
  mysql: "bg-orange-500/20 text-orange-400",
  mongodb: "bg-green-500/20 text-green-400",
  redis: "bg-red-500/20 text-red-400",
};

const mockBackups = [
  { id: "bk-1", type: "Automatic", size: "1.2GB", status: "completed" as const, createdAt: "3 hours ago" },
  { id: "bk-2", type: "Automatic", size: "1.2GB", status: "completed" as const, createdAt: "1 day ago" },
  { id: "bk-3", type: "Manual", size: "1.1GB", status: "completed" as const, createdAt: "3 days ago" },
  { id: "bk-4", type: "Automatic", size: "1.1GB", status: "completed" as const, createdAt: "4 days ago" },
];

function buildConnectionUrl(db: (typeof mockDatabases)[number]): string {
  switch (db.engine) {
    case "postgresql":
      return `postgresql://app:***@${db.name}:5432/${db.name.replace("-db", "")}`;
    case "mysql":
      return `mysql://app:***@${db.name}:3306/${db.name.replace("-db", "")}`;
    case "mongodb":
      return `mongodb://app:***@${db.name}:27017/${db.name.replace("-db", "")}`;
    case "redis":
      return `redis://${db.name}:6379`;
    default:
      return "";
  }
}

export default async function DatabaseDetailPage({
  params,
}: {
  params: Promise<{ name: string }>;
}) {
  const { name } = await params;
  const db = mockDatabases.find((d) => d.name === name);

  if (!db) {
    notFound();
  }

  const storagePercent = parseStoragePercent(db.storageUsed, db.storageTotal);
  const connectionPercent = Math.round(
    (db.connections.used / db.connections.total) * 100
  );
  const connectionUrl = buildConnectionUrl(db);

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div
              className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-sm font-bold ${
                engineColors[db.engine] ?? "bg-neutral-500/20 text-neutral-400"
              }`}
            >
              {db.engine[0].toUpperCase()}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-lg font-semibold text-white">{db.name}</h1>
                <span
                  className={`rounded-md px-2 py-0.5 text-xs font-medium capitalize ${
                    engineColors[db.engine] ?? "bg-neutral-500/20 text-neutral-400"
                  }`}
                >
                  {db.engine}
                </span>
                <StatusBadge status={db.status} />
              </div>
              <p className="mt-0.5 text-sm text-neutral-500">
                Version {db.version}
              </p>
            </div>
          </div>
        </div>

        {/* Connection string */}
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <div className="mb-2 flex items-center justify-between">
            <h3 className="text-sm font-medium text-white">Connection String</h3>
            <button className="rounded-md border border-border bg-surface-200 px-2.5 py-1 text-xs text-neutral-400 hover:bg-surface-300 hover:text-white transition-colors">
              Copy
            </button>
          </div>
          <div className="rounded-lg bg-surface-200 p-3">
            <code className="font-mono text-xs text-neutral-300 break-all">
              {connectionUrl}
            </code>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-4 gap-4">
          <StatCard
            label="Engine"
            value={db.engine}
            sub={`Version ${db.version}`}
          />
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-xs font-medium text-neutral-500">Storage</p>
            <p className="mt-1 text-2xl font-semibold text-white">
              {db.storageUsed}
            </p>
            <p className="mt-0.5 text-xs text-neutral-500">
              of {db.storageTotal}
            </p>
            <div className="mt-2">
              <ProgressBar percent={storagePercent} label={`${storagePercent}%`} />
            </div>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-xs font-medium text-neutral-500">Connections</p>
            <p className="mt-1 text-2xl font-semibold text-white">
              {db.connections.used}
            </p>
            <p className="mt-0.5 text-xs text-neutral-500">
              of {db.connections.total} max
            </p>
            <div className="mt-2">
              <ProgressBar percent={connectionPercent} label={`${connectionPercent}%`} />
            </div>
          </div>
          <StatCard
            label="Last Backup"
            value={db.lastBackup ?? "Never"}
            sub="automatic"
          />
        </div>

        {/* Linked Apps */}
        {db.linkedApps.length > 0 && (
          <section>
            <h2 className="mb-3 text-sm font-medium text-white">Linked Apps</h2>
            <div className="flex flex-wrap gap-2">
              {db.linkedApps.map((appName) => (
                <Link
                  key={appName}
                  href={`/apps/${appName}`}
                  className="rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-neutral-300 hover:border-border-hover hover:text-white transition-colors"
                >
                  {appName}
                </Link>
              ))}
            </div>
          </section>
        )}

        {/* Backups */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Recent Backups</h2>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">ID</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Type</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Size</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Created</th>
                </tr>
              </thead>
              <tbody>
                {mockBackups.map((backup) => (
                  <tr
                    key={backup.id}
                    className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors"
                  >
                    <td className="px-4 py-3 font-mono text-xs text-accent-400">{backup.id}</td>
                    <td className="px-4 py-3 text-neutral-300">{backup.type}</td>
                    <td className="px-4 py-3 text-neutral-400">{backup.size}</td>
                    <td className="px-4 py-3">
                      <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-400">
                        <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                        {backup.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{backup.createdAt}</td>
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
