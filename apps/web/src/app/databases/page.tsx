"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { Modal } from "@/components/modal";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import { type AppDatabase } from "@/lib/api";
import Link from "next/link";
import { useState } from "react";
import { Plus, Database, Loader2, Copy, Check, AlertTriangle, Download } from "lucide-react";

const engineBadge: Record<string, { label: string; className: string }> = {
  postgresql: { label: "P", className: "bg-blue-500/20 text-blue-400" },
  mysql: { label: "M", className: "bg-orange-500/20 text-orange-400" },
  redis: { label: "R", className: "bg-red-500/20 text-red-400" },
};

const engines = [
  { id: "postgresql", label: "PostgreSQL", enabled: true },
  { id: "mysql", label: "MySQL", enabled: false },
  { id: "mongodb", label: "MongoDB", enabled: false },
  { id: "redis", label: "Redis", enabled: false },
];

function CopyField({ label, value, mono = true }: { label: string; value: string; mono?: boolean }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <div>
      <label className="mb-1 block text-[11px] font-medium text-neutral-500">{label}</label>
      <div className="flex items-center gap-2">
        <div className="flex-1 rounded-md bg-surface-200 px-3 py-2">
          <code className={`text-xs text-white break-all ${mono ? "font-mono" : ""}`}>{value}</code>
        </div>
        <button
          onClick={handleCopy}
          className="shrink-0 rounded-md border border-border p-2 text-neutral-400 hover:text-white hover:border-neutral-500 transition-colors"
        >
          {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
        </button>
      </div>
    </div>
  );
}

export default function DatabasesPage() {
  const { standaloneDatabases } = getApi();
  const [showCreate, setShowCreate] = useState(false);
  const [createEngine, setCreateEngine] = useState("postgresql");
  const [createName, setCreateName] = useState("");
  const [creating, setCreating] = useState(false);
  const [createdDb, setCreatedDb] = useState<AppDatabase | null>(null);

  const {
    data: dbList,
    loading,
    error,
    refetch,
  } = useApi(() => standaloneDatabases.list(), []);

  const handleCreate = async () => {
    if (!createName.trim()) return;
    setCreating(true);
    try {
      const result = await standaloneDatabases.create({ name: createName.trim(), engine: createEngine });
      setShowCreate(false);
      setCreateName("");
      setCreateEngine("postgresql");
      setCreatedDb(result);
      refetch();
    } catch {
      // TODO: error toast
    } finally {
      setCreating(false);
    }
  };

  if (loading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={7} rows={3} />
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

  const dbs: AppDatabase[] = dbList || [];
  const readyCount = dbs.filter((d) => d.status === "ready").length;
  const pgCount = dbs.filter((d) => d.engine === "postgresql").length;
  const redisCount = dbs.filter((d) => d.engine === "redis").length;

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Databases</h1>
            <p className="text-sm text-neutral-500">
              {dbs.length} instances, {readyCount} ready
              {pgCount > 0 ? `, ${pgCount} PostgreSQL` : ""}
              {redisCount > 0 ? `, ${redisCount} Redis` : ""}
            </p>
          </div>
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-2 rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-3 py-1.5 text-sm transition-colors"
          >
            <Plus className="h-4 w-4" />
            Create Database
          </button>
        </div>

        {/* Filter bar */}
        <div className="flex items-center gap-3">
          <div className="relative flex-1">
            <svg
              className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-500"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M21 21l-4.35-4.35M11 19a8 8 0 100-16 8 8 0 000 16z"
              />
            </svg>
            <input
              type="text"
              placeholder="Filter instances..."
              className="w-full rounded-lg border border-border bg-surface-100 py-1.5 pl-9 pr-3 text-sm text-white placeholder:text-neutral-500 focus:border-accent-500 focus:outline-none"
            />
          </div>
          <select className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 focus:border-accent-500 focus:outline-none">
            <option value="">All engines</option>
            <option value="postgresql">PostgreSQL</option>
            <option value="redis">Redis</option>
            <option value="mysql">MySQL</option>
          </select>
        </div>

        {/* Table or Empty State */}
        {dbs.length === 0 ? (
          <EmptyState
            title="No databases yet"
            description="Create a standalone database to get started."
            actionLabel="Create Database"
            onAction={() => setShowCreate(true)}
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Name</th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Engine</th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">App</th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Status</th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Size</th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Port</th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Created</th>
                  </tr>
                </thead>
                <tbody>
                  {dbs.map((db) => {
                    const badge = engineBadge[db.engine] ?? {
                      label: "?",
                      className: "bg-neutral-500/20 text-neutral-400",
                    };
                    return (
                      <tr
                        key={db.id}
                        className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors"
                      >
                        <td className="whitespace-nowrap px-4 py-3">
                          <Link
                            href={`/databases/${db.id}`}
                            className="font-medium text-white underline decoration-neutral-600 underline-offset-2 hover:text-accent-400 hover:decoration-accent-400 transition-colors"
                          >
                            {db.name}
                          </Link>
                        </td>
                        <td className="whitespace-nowrap px-4 py-3">
                          <span className={`inline-flex h-5 w-5 items-center justify-center rounded text-[10px] font-bold ${badge.className}`}>
                            {badge.label}
                          </span>
                          <span className="ml-2 text-xs capitalize text-neutral-300">{db.engine}</span>
                        </td>
                        <td className="whitespace-nowrap px-4 py-3">
                          {db.app_id ? (
                            <Link href={`/apps/${db.app_id}`} className="text-xs text-accent-400 hover:underline">
                              {db.app_id.slice(0, 8)}...
                            </Link>
                          ) : (
                            <span className="inline-flex rounded-full bg-purple-500/15 px-2 py-0.5 text-[10px] font-medium text-purple-400">
                              Standalone
                            </span>
                          )}
                        </td>
                        <td className="whitespace-nowrap px-4 py-3">
                          <StatusBadge status={db.status as "ready" | "running" | "creating" | "stopped"} />
                        </td>
                        <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                          {db.size_mb} / {db.max_size_mb} MB
                        </td>
                        <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">{db.port}</td>
                        <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-400">
                          {new Date(db.created_at).toLocaleDateString()}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      {/* Create Database Modal */}
      {showCreate && (
        <Modal title="Create Database" onClose={() => setShowCreate(false)}>
          <div className="space-y-4">
            <div>
              <label className="mb-2 block text-xs font-medium text-neutral-400">Engine</label>
              <div className="grid grid-cols-2 gap-2">
                {engines.map((eng) => (
                  <button
                    key={eng.id}
                    type="button"
                    disabled={!eng.enabled}
                    onClick={() => setCreateEngine(eng.id)}
                    className={`relative rounded-lg border p-3 text-left transition-colors ${
                      createEngine === eng.id && eng.enabled
                        ? "border-accent-500 bg-accent-500/10"
                        : eng.enabled
                          ? "border-border bg-surface-100 hover:border-neutral-600"
                          : "border-border bg-surface-100 opacity-50 cursor-not-allowed"
                    }`}
                  >
                    <div className="flex items-center gap-2">
                      <Database className="h-4 w-4 text-neutral-400" />
                      <span className="text-sm font-medium text-white">{eng.label}</span>
                    </div>
                    {!eng.enabled && (
                      <span className="absolute right-2 top-2 rounded bg-neutral-500/20 px-1.5 py-0.5 text-[9px] text-neutral-500">
                        Coming Soon
                      </span>
                    )}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-400">Database Name</label>
              <input
                type="text"
                value={createName}
                onChange={(e) => setCreateName(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
                placeholder="my-database"
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
              <p className="mt-1 text-[11px] text-neutral-600">Lowercase letters, numbers, and hyphens only</p>
            </div>

            <div className="flex justify-end">
              <button
                onClick={handleCreate}
                disabled={!createName.trim() || creating}
                className="flex items-center gap-2 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
              >
                {creating && <Loader2 className="h-4 w-4 animate-spin" />}
                {creating ? "Creating..." : "Create"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Credentials Modal — shown after successful creation */}
      {createdDb && (() => {
        const password = createdDb.db_password ?? "";
        const connStr = createdDb.connection_string || `${createdDb.engine}://${createdDb.db_user}:${password}@${createdDb.host}:${createdDb.port}/${createdDb.db_name}`;

        const handleDownloadEnv = () => {
          const envContent = [
            `# Database credentials for ${createdDb.name}`,
            `# Generated by Zenith on ${new Date().toISOString()}`,
            ``,
            `DATABASE_URL=${connStr}`,
            `DB_HOST=${createdDb.host}`,
            `DB_PORT=${createdDb.port}`,
            `DB_NAME=${createdDb.db_name}`,
            `DB_USER=${createdDb.db_user}`,
            `DB_PASSWORD=${password}`,
          ].join("\n");
          const blob = new Blob([envContent + "\n"], { type: "text/plain" });
          const url = URL.createObjectURL(blob);
          const a = document.createElement("a");
          a.href = url;
          a.download = `${createdDb.name}.env`;
          a.click();
          URL.revokeObjectURL(url);
        };

        return (
          <Modal title="Database Created" onClose={() => setCreatedDb(null)}>
            <div className="space-y-4">
              {/* Warning banner */}
              <div className="flex items-start gap-3 rounded-lg border border-amber-500/30 bg-amber-500/5 px-4 py-3">
                <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 shrink-0" />
                <div>
                  <p className="text-sm font-medium text-amber-300">Save your credentials now</p>
                  <p className="text-xs text-amber-400/70 mt-0.5">
                    The password will not be shown again. Make sure to copy or download before closing.
                  </p>
                </div>
              </div>

              <CopyField label="Host" value={createdDb.host} />
              <CopyField label="Port" value={String(createdDb.port)} />
              <CopyField label="Database" value={createdDb.db_name} />
              <CopyField label="Username" value={createdDb.db_user} />
              <CopyField label="Password" value={password} />
              <CopyField label="Connection String" value={connStr} />

              <div className="flex items-center justify-between pt-2">
                <button
                  onClick={handleDownloadEnv}
                  className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white hover:border-neutral-500 transition-colors"
                >
                  <Download className="h-4 w-4" />
                  Download .env
                </button>
                <button
                  onClick={() => setCreatedDb(null)}
                  className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
                >
                  I&apos;ve saved my credentials
                </button>
              </div>
            </div>
          </Modal>
        );
      })()}
    </Shell>
  );
}
