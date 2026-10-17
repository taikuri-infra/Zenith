"use client";
import { useEffect, useState } from "react";
import {
  AlertTriangle,
  Check,
  Copy,
  Eye,
  EyeOff,
  Lock,
  Pencil,
  Plus,
  Trash2,
  Unlock,
  Upload,
} from "lucide-react";
import type { AppEnvVar, DeployApp, Secret } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import { useApi } from "@/hooks/use-api";
import { useToast } from "@/components/toast";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { isManagedEnv, managedEnvLabel } from "@/lib/env-source";

type SecretTarget = "env" | "store";

/**
 * Environment card: per-environment env vars (Production / Staging, add,
 * inline edit, delete, Import .env, Show/Hide values, managed badge,
 * Restart-now banner) plus the app's secret-store entries in the same
 * table, tagged "secret store" (reveal on demand, copy, delete).
 * The two stores load and fail independently.
 */
export function EnvironmentCard({ app }: { app: DeployApp }) {
  const appId = app.id;
  const projectId = app.project_id;
  const api = getApi();
  const { toast } = useToast();

  const [activeEnv, setActiveEnv] = useState<"production" | "staging">("production");
  const [stagingEnvId, setStagingEnvId] = useState<string>("");
  const [envIdsLoaded, setEnvIdsLoaded] = useState(false);

  useEffect(() => {
    if (!projectId) { setEnvIdsLoaded(true); return; }
    // Promise.resolve().then: a missing/throwing client rejects instead of throwing in the effect.
    Promise.resolve().then(() => api.environments.list(projectId)).then((resp) => {
      const staging = resp.environments?.find((e) => e.name === "staging");
      if (staging) setStagingEnvId(staging.id);
      setEnvIdsLoaded(true);
    }).catch(() => setEnvIdsLoaded(true));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  const currentEnvId = activeEnv === "staging" ? stagingEnvId : "";

  const { data: envData, loading, error, refetch } = useApi(
    () => envIdsLoaded ? api.envVarsV2.list(appId, currentEnvId) : Promise.resolve(null),
    [appId, currentEnvId, envIdsLoaded]
  );
  const {
    data: secretsData,
    loading: secretsLoading,
    error: secretsError,
    refetch: refetchSecrets,
  } = useApi(() => api.appsDeploy.listSecrets(appId), [appId]);

  // Add new var
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");
  const [newIsSecret, setNewIsSecret] = useState(false);
  const [secretTarget, setSecretTarget] = useState<SecretTarget>("env");
  const [adding, setAdding] = useState(false);
  const [showValues, setShowValues] = useState(false);

  // Import .env
  const [importing, setImporting] = useState(false);
  const [showImport, setShowImport] = useState(false);
  const [dotEnvContent, setDotEnvContent] = useState("");

  // Inline edit
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");
  const [editIsSecret, setEditIsSecret] = useState(false);
  const [saving, setSaving] = useState(false);

  // Pending restart banner
  const [pendingRestart, setPendingRestart] = useState(false);
  const [applying, setApplying] = useState(false);

  // Delete confirmation (env var id)
  const [deletingId, setDeletingId] = useState<string | null>(null);

  // Secret store: fetched values are cached; `shownSecrets` toggles visibility client-side.
  const [secretValues, setSecretValues] = useState<Record<string, string>>({});
  const [shownSecrets, setShownSecrets] = useState<Record<string, boolean>>({});
  const [revealingKey, setRevealingKey] = useState<string | null>(null);
  const [confirmSecretKey, setConfirmSecretKey] = useState<string | null>(null);
  const [deletingSecretKey, setDeletingSecretKey] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  const items: AppEnvVar[] = envData?.items || [];
  const secrets: Secret[] = secretsData?.secrets || [];
  const toSecretStore = newIsSecret && secretTarget === "store";

  const markPending = () => setPendingRestart(true);

  const handleAdd = async () => {
    if (!newKey.trim()) return;
    if (toSecretStore) {
      if (!newValue.trim() || adding) return;
      // The secrets API puts the key in the URL path unencoded: keep it to [A-Z0-9_].
      const key = newKey.trim().toUpperCase().replace(/[^A-Z0-9_]/g, "");
      if (!key) {
        toast("error", "Secret keys may only contain A-Z, 0-9 and _");
        return;
      }
      setAdding(true);
      try {
        await api.appsDeploy.setSecret(appId, key, newValue);
        setNewKey(""); setNewValue(""); setNewIsSecret(false); setSecretTarget("env");
        // The stored value may have changed: drop any cached reveal.
        setSecretValues((prev) => { const next = { ...prev }; delete next[key]; return next; });
        setShownSecrets((prev) => { const next = { ...prev }; delete next[key]; return next; });
        refetchSecrets();
      } catch {
        toast("error", "Failed to set secret");
      } finally {
        setAdding(false);
      }
      return;
    }
    const key = newKey.trim();
    try {
      await api.envVarsV2.set(appId, [{ key, value: newValue, is_secret: newIsSecret }], currentEnvId);
      setNewKey(""); setNewValue(""); setNewIsSecret(false); setSecretTarget("env");
      refetch(); markPending();
    } catch {
      toast("error", "Failed to set environment variable");
    }
  };

  const handleDelete = async (varId: string) => {
    try {
      await api.envVarsV2.delete(appId, varId);
      setDeletingId(null);
      refetch(); markPending();
    } catch {
      toast("error", "Failed to delete environment variable");
    }
  };

  const startEdit = (ev: AppEnvVar) => {
    setEditingId(ev.id);
    setEditValue(ev.is_secret ? "" : ev.value); // secrets: force re-entry
    setEditIsSecret(ev.is_secret);
  };

  const cancelEdit = () => { setEditingId(null); setEditValue(""); };

  const handleSaveEdit = async (ev: AppEnvVar) => {
    setSaving(true);
    try {
      await api.envVarsV2.set(appId, [{ key: ev.key, value: editValue, is_secret: editIsSecret }], currentEnvId);
      setEditingId(null);
      refetch(); markPending();
    } catch {
      toast("error", "Failed to update environment variable");
    } finally {
      setSaving(false);
    }
  };

  const handleImport = async () => {
    if (!dotEnvContent.trim()) return;
    setImporting(true);
    try {
      const result = await api.envVarsV2.importDotEnv(appId, dotEnvContent, currentEnvId);
      toast("success", `Imported ${result.imported} variable${result.imported !== 1 ? "s" : ""}`);
      setDotEnvContent(""); setShowImport(false);
      refetch(); markPending();
    } catch {
      toast("error", "Failed to import .env file");
    } finally {
      setImporting(false);
    }
  };

  const handleApply = async () => {
    setApplying(true);
    try {
      await api.envVarsV2.apply(appId);
      toast("success", "Rolling restart triggered — changes will be live in ~30 seconds");
      setPendingRestart(false);
    } catch {
      toast("error", "Failed to restart app");
    } finally {
      setApplying(false);
    }
  };

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => setDotEnvContent(ev.target?.result as string ?? "");
    reader.readAsText(file);
    e.target.value = "";
  };

  const handleReveal = async (key: string) => {
    if (shownSecrets[key]) {
      setShownSecrets((prev) => ({ ...prev, [key]: false }));
      return;
    }
    if (key in secretValues) {
      setShownSecrets((prev) => ({ ...prev, [key]: true }));
      return;
    }
    setRevealingKey(key);
    try {
      const result = await api.appsDeploy.getSecretValue(appId, key);
      setSecretValues((prev) => ({ ...prev, [key]: result.value }));
      setShownSecrets((prev) => ({ ...prev, [key]: true }));
    } catch {
      toast("error", "Failed to reveal secret value");
    } finally {
      setRevealingKey(null);
    }
  };

  const handleCopy = async (key: string, value: string) => {
    try {
      await navigator.clipboard.writeText(value);
      setCopiedKey(key);
      setTimeout(() => setCopiedKey(null), 2000);
    } catch {
      toast("error", "Failed to copy value");
    }
  };

  const handleDeleteSecret = async (key: string) => {
    if (deletingSecretKey) return;
    setDeletingSecretKey(key);
    try {
      await api.appsDeploy.deleteSecret(appId, key);
      setConfirmSecretKey(null);
      setSecretValues((prev) => { const next = { ...prev }; delete next[key]; return next; });
      setShownSecrets((prev) => { const next = { ...prev }; delete next[key]; return next; });
      refetchSecrets();
    } catch {
      toast("error", "Failed to delete secret");
    } finally {
      setDeletingSecretKey(null);
    }
  };

  const hasStagingEnv = !!stagingEnvId;
  // envData is null for one render after envIdsLoaded flips (before useApi refetches).
  const envPending = !envIdsLoaded || loading || (!envData && !error);
  const secretsPending = secretsLoading && !secretsData;
  const envRows = error ? [] : items;
  const secretRows = secretsError ? [] : secrets;
  const addDisabled = !newKey.trim() || (toSecretStore && (!newValue.trim() || adding));

  const table = (
    <div className="overflow-hidden rounded-lg border border-border">
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-border bg-surface-100">
            <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Key</th>
            <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Value</th>
            <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500 w-24 text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          {envRows.map((ev) => (
            <tr key={`env:${ev.id}`} className="border-b border-border last:border-0 hover:bg-surface-200/50 transition-colors">
              <td className="whitespace-nowrap px-4 py-2 font-mono text-xs font-medium text-accent-400">
                <div className="flex items-center gap-1.5">
                  {ev.is_secret && <Lock className="h-3 w-3 text-amber-400 shrink-0" />}
                  {ev.key}
                  {isManagedEnv(ev) && (
                    <span className="rounded bg-blue-500/10 px-1.5 py-0.5 text-[10px] text-blue-400">{managedEnvLabel(ev)}</span>
                  )}
                </div>
              </td>
              <td className="px-4 py-2 font-mono text-xs text-neutral-300">
                {editingId === ev.id ? (
                  <div className="flex items-center gap-2">
                    <input
                      autoFocus
                      type={editIsSecret ? "password" : "text"}
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      onKeyDown={(e) => { if (e.key === "Enter") handleSaveEdit(ev); if (e.key === "Escape") cancelEdit(); }}
                      placeholder={ev.is_secret ? "Enter new value..." : ev.value}
                      className="flex-1 rounded border border-accent-500 bg-surface-300 px-2 py-1 text-xs text-white placeholder:text-neutral-600 focus:outline-none font-mono"
                    />
                    <button
                      onClick={() => setEditIsSecret(!editIsSecret)}
                      className={`rounded p-1 transition-colors ${editIsSecret ? "text-amber-400 bg-amber-400/10" : "text-neutral-500 hover:text-neutral-300"}`}
                      title={editIsSecret ? "Mark as plain" : "Mark as secret"}
                    >
                      {editIsSecret ? <Lock className="h-3 w-3" /> : <Unlock className="h-3 w-3" />}
                    </button>
                  </div>
                ) : (
                  <span className="truncate max-w-xs block">
                    {ev.is_secret ? "••••••••" : (showValues ? ev.value : ev.value.length > 40 ? ev.value.slice(0, 40) + "…" : ev.value)}
                  </span>
                )}
              </td>
              <td className="whitespace-nowrap px-4 py-2 text-right">
                {editingId === ev.id ? (
                  <div className="flex items-center justify-end gap-1">
                    <button
                      onClick={() => handleSaveEdit(ev)}
                      disabled={saving}
                      className="rounded px-2 py-1 text-xs bg-accent-500 text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
                    >
                      {saving ? "..." : "Save"}
                    </button>
                    <button
                      onClick={cancelEdit}
                      className="rounded px-2 py-1 text-xs text-neutral-500 hover:text-white transition-colors"
                    >
                      Cancel
                    </button>
                  </div>
                ) : deletingId === ev.id ? (
                  <div className="flex items-center justify-end gap-1">
                    <button
                      onClick={() => handleDelete(ev.id)}
                      className="rounded px-2 py-1 text-xs bg-red-500/20 text-red-400 hover:bg-red-500/30 transition-colors"
                    >
                      Confirm
                    </button>
                    <button
                      onClick={() => setDeletingId(null)}
                      className="rounded px-2 py-1 text-xs text-neutral-500 hover:text-white transition-colors"
                    >
                      Cancel
                    </button>
                  </div>
                ) : !isManagedEnv(ev) && (
                  <div className="flex items-center justify-end gap-1">
                    <button
                      onClick={() => startEdit(ev)}
                      className="rounded p-1 text-neutral-500 hover:text-white transition-colors"
                      title="Edit variable"
                    >
                      <Pencil className="h-3.5 w-3.5" />
                    </button>
                    <button
                      onClick={() => setDeletingId(ev.id)}
                      className="rounded p-1 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 transition-colors"
                      title="Delete variable"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                )}
              </td>
            </tr>
          ))}
          {secretRows.map((secret) => {
            const shown = !!shownSecrets[secret.key] && secret.key in secretValues;
            const value = secretValues[secret.key];
            return (
              <tr key={`secret:${secret.key}`} className="border-b border-border last:border-0 hover:bg-surface-200/50 transition-colors">
                <td className="whitespace-nowrap px-4 py-2 font-mono text-xs font-medium text-accent-400">
                  <div className="flex items-center gap-1.5">
                    <Lock className="h-3 w-3 text-amber-400 shrink-0" />
                    {secret.key}
                    <span className="rounded bg-amber-500/10 px-1.5 py-0.5 text-[10px] text-amber-400">secret store</span>
                  </div>
                </td>
                <td className="px-4 py-2 font-mono text-xs text-neutral-300">
                  {shown ? (
                    <span className="flex items-center gap-2">
                      <span className="max-w-xs truncate">{value}</span>
                      <button
                        onClick={() => handleCopy(secret.key, value)}
                        className="rounded p-0.5 text-neutral-500 hover:text-white transition-colors"
                        title="Copy value"
                        aria-label="Copy value"
                      >
                        {copiedKey === secret.key ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
                      </button>
                    </span>
                  ) : (
                    "••••••••"
                  )}
                </td>
                <td className="whitespace-nowrap px-4 py-2 text-right">
                  {confirmSecretKey === secret.key ? (
                    <div className="flex items-center justify-end gap-1">
                      <button
                        onClick={() => handleDeleteSecret(secret.key)}
                        disabled={deletingSecretKey === secret.key}
                        className="rounded px-2 py-1 text-xs bg-red-500/20 text-red-400 hover:bg-red-500/30 disabled:opacity-50 transition-colors"
                      >
                        {deletingSecretKey === secret.key ? "..." : "Confirm"}
                      </button>
                      <button
                        onClick={() => setConfirmSecretKey(null)}
                        className="rounded px-2 py-1 text-xs text-neutral-500 hover:text-white transition-colors"
                      >
                        Cancel
                      </button>
                    </div>
                  ) : (
                    <div className="flex items-center justify-end gap-1">
                      <button
                        onClick={() => handleReveal(secret.key)}
                        disabled={revealingKey === secret.key}
                        className="flex items-center gap-1 rounded px-2 py-1 text-xs text-neutral-400 hover:bg-surface-300 hover:text-white transition-colors disabled:opacity-50"
                        title={shown ? "Hide value" : "Reveal value"}
                      >
                        {revealingKey === secret.key ? (
                          <div className="h-3 w-3 animate-spin rounded-full border border-neutral-400 border-t-transparent" />
                        ) : shown ? (
                          <EyeOff className="h-3 w-3" />
                        ) : (
                          <Unlock className="h-3 w-3" />
                        )}
                        {shown ? "Hide" : "Reveal"}
                      </button>
                      <button
                        onClick={() => setConfirmSecretKey(secret.key)}
                        className="rounded p-1 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 transition-colors"
                        title="Delete secret"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );

  let list: React.ReactNode;
  if (envPending || secretsPending) {
    list = <PageWithTableSkeleton cols={3} rows={3} />;
  } else if (envRows.length === 0 && secretRows.length === 0) {
    list = error || secretsError ? null : (
      <EmptyState
        title={`No ${activeEnv} environment variables`}
        description={activeEnv === "staging"
          ? "Add staging-specific overrides. Production vars are used as the baseline."
          : "Add environment variables to configure your app."}
      />
    );
  } else {
    list = table;
  }

  return (
    <div className="md:col-span-2 rounded-lg border border-border bg-surface-100 p-5">
      <h2 className="mb-4 text-sm font-medium text-white">Environment</h2>
      <div className="space-y-4">
        {/* Pending restart banner */}
        {pendingRestart && (
          <div className="flex items-center justify-between rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3">
            <p className="text-sm text-amber-300">
              Changes saved to <span className="font-semibold">{activeEnv}</span>. Restart the app to apply them to the running container.
            </p>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setPendingRestart(false)}
                className="text-xs text-neutral-500 hover:text-neutral-300 transition-colors"
              >
                Dismiss
              </button>
              <button
                onClick={handleApply}
                disabled={applying}
                className="rounded-lg bg-amber-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-amber-600 disabled:opacity-50 transition-colors"
              >
                {applying ? "Restarting..." : "Restart now"}
              </button>
            </div>
          </div>
        )}

        {/* Environment toggle + toolbar */}
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex items-center gap-3">
            <div className="flex rounded-lg border border-border overflow-hidden">
              <button
                onClick={() => setActiveEnv("production")}
                className={`flex items-center gap-1.5 px-4 py-1.5 text-sm font-medium transition-colors ${
                  activeEnv === "production"
                    ? "bg-red-600 text-white"
                    : "bg-surface-100 text-neutral-400 hover:text-white"
                }`}
              >
                Production
              </button>
              <button
                onClick={() => setActiveEnv("staging")}
                disabled={!hasStagingEnv}
                title={!hasStagingEnv ? "Staging environment not available on Free plan" : undefined}
                className={`px-4 py-1.5 text-sm font-medium transition-colors ${
                  activeEnv === "staging"
                    ? "bg-green-600 text-white"
                    : "bg-surface-100 text-neutral-400 hover:text-white disabled:opacity-40 disabled:cursor-not-allowed"
                }`}
              >
                Staging
              </button>
            </div>
            {activeEnv === "production" && (
              <span className="flex items-center gap-1 rounded-full border border-red-500/40 bg-red-500/10 px-2.5 py-0.5 text-xs font-medium text-red-400">
                <AlertTriangle className="h-3 w-3" />
                Live environment
              </span>
            )}
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowImport(!showImport)}
              className="flex items-center gap-1.5 rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-xs font-medium text-neutral-400 hover:text-white transition-colors"
            >
              <Upload className="h-3.5 w-3.5" />
              Import .env
            </button>
            <button
              onClick={() => setShowValues(!showValues)}
              className="flex items-center gap-1.5 text-xs text-neutral-500 hover:text-white transition-colors"
            >
              {showValues ? <EyeOff className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
              {showValues ? "Hide values" : "Show values"}
            </button>
          </div>
        </div>

        {/* .env import panel */}
        {showImport && (
          <div className="rounded-lg border border-border bg-surface-100 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium text-neutral-300">Import .env file</p>
              <label className="flex items-center gap-1.5 cursor-pointer rounded border border-border px-2 py-1 text-xs text-neutral-400 hover:text-white transition-colors">
                <Upload className="h-3 w-3" />
                Upload file
                <input type="file" accept=".env,text/plain" className="hidden" onChange={handleFileUpload} />
              </label>
            </div>
            <textarea
              value={dotEnvContent}
              onChange={(e) => setDotEnvContent(e.target.value)}
              placeholder={"DATABASE_URL=postgres://...\nREDIS_URL=redis://...\nAPI_SECRET=mysecret"}
              rows={6}
              className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 font-mono text-xs text-neutral-300 placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none resize-none"
            />
            <p className="text-xs text-neutral-600">
              Supports KEY=VALUE, quoted values, # comments, and <code className="text-neutral-500">export KEY=VALUE</code>.
              Values with &quot;password&quot;, &quot;secret&quot;, &quot;token&quot;, etc. are automatically marked as secret.
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => { setShowImport(false); setDotEnvContent(""); }}
                className="rounded-lg border border-border px-3 py-1.5 text-xs text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleImport}
                disabled={!dotEnvContent.trim() || importing}
                className="rounded-lg bg-accent-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
              >
                {importing ? "Importing..." : `Import to ${activeEnv}`}
              </button>
            </div>
          </div>
        )}

        {/* Add new env var / secret */}
        <div className="flex flex-wrap items-end gap-2">
          <div className="min-w-[10rem] flex-1">
            <label className="mb-1 block text-xs font-medium text-neutral-400">Key</label>
            <input
              type="text"
              value={newKey}
              onChange={(e) => setNewKey(e.target.value.toUpperCase())}
              onKeyDown={(e) => e.key === "Enter" && handleAdd()}
              placeholder="DATABASE_URL"
              className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none font-mono"
            />
          </div>
          <div className="min-w-[10rem] flex-1">
            <label className="mb-1 block text-xs font-medium text-neutral-400">Value</label>
            <input
              type={newIsSecret ? "password" : "text"}
              value={newValue}
              onChange={(e) => setNewValue(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleAdd()}
              placeholder="postgres://..."
              className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none font-mono"
            />
          </div>
          <div className="flex flex-col items-center gap-1">
            <label className="text-xs font-medium text-neutral-400">Secret</label>
            <button
              onClick={() => setNewIsSecret(!newIsSecret)}
              className={`rounded p-1.5 transition-colors ${newIsSecret ? "text-amber-400 bg-amber-400/10" : "text-neutral-500 hover:text-neutral-300"}`}
              title={newIsSecret ? "Mark as plain" : "Mark as secret"}
            >
              {newIsSecret ? <Lock className="h-4 w-4" /> : <Unlock className="h-4 w-4" />}
            </button>
          </div>
          {newIsSecret && (
            <div>
              <label htmlFor={`env-store-${appId}`} className="mb-1 block text-xs font-medium text-neutral-400">Store</label>
              <select
                id={`env-store-${appId}`}
                value={secretTarget}
                onChange={(e) => setSecretTarget(e.target.value as SecretTarget)}
                className="rounded-md border border-border bg-surface-200 px-2 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
              >
                <option value="env">Env var (secret)</option>
                <option value="store">Secret store</option>
              </select>
            </div>
          )}
          <button
            onClick={handleAdd}
            disabled={addDisabled}
            className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
          >
            <Plus className="h-4 w-4" />
            Add
          </button>
        </div>
        {toSecretStore && (
          <p className="text-xs text-neutral-600">
            Secret-store values are encrypted with AES-256-GCM, shared by all environments, and never logged. Keys keep only A-Z, 0-9 and _.
          </p>
        )}

        {/* Per-store errors: one failing store does not hide the other */}
        {!envPending && error && <ErrorState message={error} onRetry={refetch} />}
        {!secretsPending && secretsError && <ErrorState message={secretsError} onRetry={refetchSecrets} />}

        {/* Env var + secret-store list */}
        {list}
        {envRows.some(isManagedEnv) && (
          <p className="text-xs text-neutral-500">Variables from an attached database are read-only; detach the database to remove them.</p>
        )}
      </div>
    </div>
  );
}
