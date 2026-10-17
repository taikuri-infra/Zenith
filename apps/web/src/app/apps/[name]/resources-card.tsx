"use client";
import { useState } from "react";
import { ChevronDown, ChevronRight, Plus, Trash2, Download, RotateCw, Eye, EyeOff, Copy, Check } from "lucide-react";
import type { DeployApp, AppDatabase, AppBucket, DatabaseBackup } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import { useApi } from "@/hooks/use-api";
import { useToast } from "@/components/toast";
import { ErrorState } from "@/components/error-state";
import { StatusBadge } from "@/components/status-badge";

const inputCls =
  "rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none";
const primaryBtn =
  "rounded-lg bg-accent-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors";
const ghostBtn = "rounded-lg border border-border px-3 py-1.5 text-sm text-neutral-400 hover:text-white transition-colors";
const rowBtn = "rounded p-1 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 disabled:opacity-50 transition-colors";

// A missing API method must never throw during render.
function missing<T>(): Promise<T[]> {
  return Promise.resolve([]);
}

export function ResourcesCard({ app }: { app: DeployApp }) {
  return (
    <section className="md:col-span-2 rounded-lg border border-border bg-surface-100 p-5">
      <h2 className="mb-4 text-sm font-medium text-white">Resources</h2>
      <div className="space-y-6">
        <DatabasesSection appId={app.id} />
        <BucketsSection appId={app.id} />
      </div>
    </section>
  );
}

function DatabasesSection({ appId }: { appId: string }) {
  const { toast } = useToast();
  const { data, loading, error, refetch } = useApi<AppDatabase[]>(() => {
    const { appsDeploy } = getApi();
    return appsDeploy.listAppDatabases ? appsDeploy.listAppDatabases(appId) : missing<AppDatabase>();
  }, [appId]);
  const [showAdd, setShowAdd] = useState(false);
  const [engine, setEngine] = useState("postgresql");
  const [creating, setCreating] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const create = async () => {
    setCreating(true);
    try {
      await getApi().appsDeploy.createAppDatabase(appId, { engine });
      setShowAdd(false);
      setEngine("postgresql");
      refetch();
    } catch {
      toast("error", "Failed to create database");
    } finally {
      setCreating(false);
    }
  };

  const remove = async (id: string) => {
    if (deletingId) return;
    setDeletingId(id);
    try {
      await getApi().appsDeploy.deleteAppDatabase(appId, id);
      refetch();
    } catch {
      toast("error", "Failed to delete database");
    } finally {
      setDeletingId(null);
    }
  };

  const items = data || [];
  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <h3 className="text-xs font-medium uppercase tracking-wide text-neutral-500">Databases</h3>
        <button onClick={() => setShowAdd(true)} className={primaryBtn}>
          <Plus className="mr-1 inline h-3.5 w-3.5" />
          Add database
        </button>
      </div>
      {showAdd && (
        <div className="mb-3 flex items-end gap-3 rounded-lg border border-border bg-surface-200/40 p-3">
          <div className="flex-1">
            <label htmlFor="res-db-engine" className="mb-1 block text-xs font-medium text-neutral-400">Engine</label>
            <select id="res-db-engine" value={engine} onChange={(e) => setEngine(e.target.value)} className={`w-full ${inputCls}`}>
              <option value="postgresql">PostgreSQL</option>
              <option value="mysql">MySQL</option>
              <option value="redis">Redis</option>
            </select>
          </div>
          <button onClick={() => setShowAdd(false)} className={ghostBtn}>Cancel</button>
          <button onClick={create} disabled={creating} className={primaryBtn}>Create database</button>
        </div>
      )}
      {loading ? (
        <div className="text-xs text-neutral-500">Loading databases...</div>
      ) : error ? (
        <ErrorState message={error} onRetry={refetch} />
      ) : items.length === 0 ? (
        <div className="text-xs text-neutral-600">No databases. Add one to get a connection string injected into your app.</div>
      ) : (
        <ul className="divide-y divide-border rounded-lg border border-border">
          {items.map((db) => (
            <li key={db.id}>
              <div className="flex items-center gap-3 px-3 py-2">
                <button
                  onClick={() => setExpanded(expanded === db.id ? null : db.id)}
                  aria-label={`${expanded === db.id ? "Collapse" : "Expand"} ${db.name}`}
                  className="text-neutral-500 hover:text-white"
                >
                  {expanded === db.id ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                </button>
                <span className="flex-1 truncate text-sm font-medium text-white">{db.name}</span>
                <span className="text-xs capitalize text-neutral-400">{db.engine}</span>
                <StatusBadge status={db.status as "ready" | "running" | "creating" | "stopped"} />
                <button onClick={() => remove(db.id)} disabled={deletingId === db.id} className={rowBtn} title="Delete database" aria-label={`Delete ${db.name}`}>
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
              {expanded === db.id && (
                <div className="space-y-3 border-t border-border bg-surface-200/30 px-3 py-3">
                  <ConnectionDetails appId={appId} db={db} />
                  <Backups appId={appId} db={db} />
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function ConnectionDetails({ appId, db }: { appId: string; db: AppDatabase }) {
  const { toast } = useToast();
  const [conn, setConn] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const fallback = `${db.engine}://${db.db_user}:***@${db.host}:${db.port}/${db.db_name}`;

  const reveal = async () => {
    if (conn) return setConn(null);
    try {
      const r = await getApi().appsDeploy.getAppDatabase(appId, db.id);
      setConn(r.connection_string || fallback);
    } catch {
      setConn(fallback);
    }
  };
  const copy = async () => {
    if (!conn) return;
    try {
      await navigator.clipboard.writeText(conn);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast("error", "Failed to copy connection string");
    }
  };

  return (
    <div className="flex items-center gap-2 text-xs">
      <span className="text-neutral-500">Size</span>
      <span className="font-mono text-neutral-300">{db.size_mb} / {db.max_size_mb} MB</span>
      <span className="ml-3 flex items-center gap-2">
        {conn ? (
          <>
            <code className="max-w-xs truncate text-neutral-300">{conn}</code>
            <button onClick={copy} title="Copy" className="text-neutral-500 hover:text-white">
              {copied ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
            </button>
            <button onClick={reveal} title="Hide" className="text-neutral-500 hover:text-white"><EyeOff className="h-3 w-3" /></button>
          </>
        ) : (
          <button onClick={reveal} className="flex items-center gap-1 text-neutral-500 hover:text-accent-400">
            <Eye className="h-3 w-3" />
            Show connection string
          </button>
        )}
      </span>
    </div>
  );
}

function Backups({ appId, db }: { appId: string; db: AppDatabase }) {
  const { toast } = useToast();
  const { data, loading, error, refetch } = useApi<DatabaseBackup[]>(() => {
    const { appsDeploy } = getApi();
    return appsDeploy.listBackups ? appsDeploy.listBackups(appId, db.id) : missing<DatabaseBackup>();
  }, [appId, db.id]);
  const [busy, setBusy] = useState<string | null>(null);

  const run = async (key: string, fn: () => Promise<unknown>, err: string) => {
    setBusy(key);
    try {
      await fn();
      refetch();
    } catch {
      toast("error", err);
    } finally {
      setBusy(null);
    }
  };

  const items = data || [];
  const api = () => getApi().appsDeploy;
  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <span className="text-xs font-medium text-neutral-400">Backups</span>
        <button
          onClick={() => run("create", () => api().createBackup(appId, db.id), "Failed to create backup")}
          disabled={busy === "create"}
          className="flex items-center gap-1 rounded-md bg-accent-500/10 px-2.5 py-1 text-xs font-medium text-accent-400 hover:bg-accent-500/20 disabled:opacity-50"
        >
          <Download className="h-3 w-3" />
          Create backup
        </button>
      </div>
      {loading ? (
        <div className="text-xs text-neutral-500">Loading backups...</div>
      ) : error ? (
        <ErrorState message={error} onRetry={refetch} />
      ) : items.length === 0 ? (
        <div className="text-xs text-neutral-600">No backups yet.</div>
      ) : (
        <div className="space-y-1.5">
          {items.map((b) => (
            <div key={b.id} className="flex items-center justify-between rounded-md bg-surface-200 px-3 py-2 text-xs">
              <div className="flex items-center gap-3">
                <span className="font-mono text-neutral-300">{b.id.slice(0, 8)}</span>
                <span className="capitalize text-neutral-500">{b.status}</span>
                <span className="text-neutral-600">{b.size_mb > 0 ? `${b.size_mb} MB` : "—"}</span>
                <span className="text-neutral-600">{new Date(b.created_at).toLocaleString()}</span>
              </div>
              <div className="flex items-center gap-1">
                {b.status === "completed" && (
                  <button
                    onClick={() => run(b.id, () => api().restoreBackup(appId, db.id, b.id), "Failed to restore backup")}
                    disabled={busy === b.id}
                    className="flex items-center gap-1 rounded px-2 py-1 text-neutral-400 hover:text-white disabled:opacity-50"
                  >
                    <RotateCw className="h-3 w-3" />
                    Restore
                  </button>
                )}
                <button
                  onClick={() => run(b.id, () => api().deleteBackup(appId, db.id, b.id), "Failed to delete backup")}
                  disabled={busy === b.id}
                  className={rowBtn}
                  aria-label="Delete backup"
                >
                  <Trash2 className="h-3 w-3" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function BucketsSection({ appId }: { appId: string }) {
  const { toast } = useToast();
  const { data, loading, error, refetch } = useApi<AppBucket[]>(() => {
    const { appsDeploy } = getApi();
    return appsDeploy.listAppBuckets ? appsDeploy.listAppBuckets(appId) : missing<AppBucket>();
  }, [appId]);
  const [showAdd, setShowAdd] = useState(false);
  const [name, setName] = useState("");
  const [access, setAccess] = useState("private");
  const [creating, setCreating] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const create = async () => {
    if (!name.trim()) return;
    setCreating(true);
    try {
      await getApi().appsDeploy.createAppBucket(appId, { name: name.trim(), access });
      setShowAdd(false);
      setName("");
      setAccess("private");
      refetch();
    } catch {
      toast("error", "Failed to create storage bucket");
    } finally {
      setCreating(false);
    }
  };

  const remove = async (id: string) => {
    if (deletingId) return;
    setDeletingId(id);
    try {
      await getApi().appsDeploy.deleteAppBucket(appId, id);
      refetch();
    } catch {
      toast("error", "Failed to delete storage bucket");
    } finally {
      setDeletingId(null);
    }
  };

  const items = data || [];
  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <h3 className="text-xs font-medium uppercase tracking-wide text-neutral-500">Buckets</h3>
        <button onClick={() => setShowAdd(true)} className={primaryBtn}>
          <Plus className="mr-1 inline h-3.5 w-3.5" />
          Add bucket
        </button>
      </div>
      {showAdd && (
        <div className="mb-3 flex items-end gap-3 rounded-lg border border-border bg-surface-200/40 p-3">
          <div className="flex-1">
            <label htmlFor="res-bucket-name" className="mb-1 block text-xs font-medium text-neutral-400">Bucket name</label>
            <input
              id="res-bucket-name"
              value={name}
              onChange={(e) => setName(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
              placeholder="my-uploads"
              className={`w-full font-mono ${inputCls}`}
            />
          </div>
          <div>
            <label htmlFor="res-bucket-access" className="mb-1 block text-xs font-medium text-neutral-400">Access</label>
            <select id="res-bucket-access" value={access} onChange={(e) => setAccess(e.target.value)} className={inputCls}>
              <option value="private">Private</option>
              <option value="public">Public</option>
            </select>
          </div>
          <button onClick={() => setShowAdd(false)} className={ghostBtn}>Cancel</button>
          <button onClick={create} disabled={!name.trim() || creating} className={primaryBtn}>Create bucket</button>
        </div>
      )}
      {loading ? (
        <div className="text-xs text-neutral-500">Loading buckets...</div>
      ) : error ? (
        <ErrorState message={error} onRetry={refetch} />
      ) : items.length === 0 ? (
        <div className="text-xs text-neutral-600">No storage buckets.</div>
      ) : (
        <ul className="divide-y divide-border rounded-lg border border-border">
          {items.map((b) => (
            <li key={b.id}>
              <div className="flex items-center gap-3 px-3 py-2">
                <button
                  onClick={() => setExpanded(expanded === b.id ? null : b.id)}
                  aria-label={`${expanded === b.id ? "Collapse" : "Expand"} ${b.name}`}
                  className="text-neutral-500 hover:text-white"
                >
                  {expanded === b.id ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                </button>
                <span className="flex-1 truncate text-sm font-medium text-white">{b.name}</span>
                <span className={`text-xs capitalize ${b.access === "public" ? "text-amber-400" : "text-neutral-400"}`}>{b.access}</span>
                <StatusBadge status={b.status as "active" | "running" | "creating" | "stopped"} />
                <button onClick={() => remove(b.id)} disabled={deletingId === b.id} className={rowBtn} title="Delete bucket" aria-label={`Delete ${b.name}`}>
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
              {expanded === b.id && (
                <div className="flex gap-4 border-t border-border bg-surface-200/30 px-3 py-2 text-xs text-neutral-400">
                  <span className="font-mono text-neutral-500">{b.endpoint}</span>
                  <span>{b.size_mb} / {b.max_size_mb} MB</span>
                  <span>{b.objects} objects</span>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
