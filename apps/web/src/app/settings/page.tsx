"use client";

import { Shell } from "@/components/shell";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useApi } from "@/hooks/use-api";
import { useToast } from "@/components/toast";
import { apiKeys, sessions, mfa, webhooks, ipWhitelist, compliance, autoscaler, type APIKey, type Session, type UserWebhook, type WebhookDelivery, type IPWhitelistEntry, type ComplianceCheck, type AutoscalerStatus, type HetznerNode, type AutoscaleEvent } from "@/lib/api";
import { useState } from "react";
import {
  KeyRound,
  Monitor,
  Smartphone,
  Plus,
  Copy,
  Check,
  Trash2,
  Shield,
  Settings,
  ShieldCheck,
  Webhook,
  Eye,
  EyeOff,
  ToggleLeft,
  ToggleRight,
  Lock,
  CheckCircle2,
  XCircle,
  AlertCircle,
  MinusCircle,
  Server,
  ArrowUpCircle,
  ArrowDownCircle,
  Activity,
} from "lucide-react";

type SettingsTab = "api-keys" | "sessions" | "mfa" | "webhooks" | "security" | "infrastructure" | "general";

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>("api-keys");

  const tabs: { key: SettingsTab; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
    { key: "api-keys", label: "API Keys", icon: KeyRound },
    { key: "mfa", label: "MFA", icon: ShieldCheck },
    { key: "webhooks", label: "Webhooks", icon: Webhook },
    { key: "sessions", label: "Sessions", icon: Shield },
    { key: "security", label: "Security", icon: Lock },
    { key: "infrastructure", label: "Infra", icon: Server },
    { key: "general", label: "General", icon: Settings },
  ];

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Settings</h1>
          <p className="text-sm text-neutral-500">Manage your account settings</p>
        </div>

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

        {activeTab === "api-keys" && <APIKeysTab />}
        {activeTab === "mfa" && <MFATab />}
        {activeTab === "webhooks" && <WebhooksTab />}
        {activeTab === "sessions" && <SessionsTab />}
        {activeTab === "security" && <SecurityTab />}
        {activeTab === "infrastructure" && <InfrastructureTab />}
        {activeTab === "general" && <GeneralTab />}
      </div>
    </Shell>
  );
}

function APIKeysTab() {
  const { toast } = useToast();
  const { data, loading, error, refetch } = useApi(() => apiKeys.list(), []);

  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [scopes, setScopes] = useState<string[]>(["read"]);
  const [creating, setCreating] = useState(false);
  const [newKey, setNewKey] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  if (loading) return <PageWithTableSkeleton cols={5} rows={3} />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;

  const items: APIKey[] = data?.items || [];

  const handleCreate = async () => {
    if (!name.trim()) return;
    setCreating(true);
    try {
      const result = await apiKeys.create(name.trim(), scopes);
      setNewKey(result.key || null);
      setShowCreate(false);
      setName("");
      setScopes(["read"]);
      refetch();
    } catch {
      toast("error", "Failed to create API key");
    } finally {
      setCreating(false);
    }
  };

  const handleCopyKey = () => {
    if (newKey) {
      navigator.clipboard.writeText(newKey);
      setCopiedKey(true);
      setTimeout(() => setCopiedKey(false), 2000);
    }
  };

  const handleDelete = async (id: string) => {
    setDeletingId(id);
    try {
      await apiKeys.delete(id);
      refetch();
    } catch {
      toast("error", "Failed to delete API key");
    } finally {
      setDeletingId(null);
    }
  };

  const toggleScope = (scope: string) => {
    setScopes((prev) =>
      prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope]
    );
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-xs text-neutral-500">
          API keys for CI/CD pipelines and programmatic access. Keys are shown only once on creation.
        </p>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
        >
          <Plus className="h-4 w-4" />
          Create Key
        </button>
      </div>

      {newKey && (
        <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-4">
          <p className="mb-2 text-xs font-medium text-emerald-400">
            API key created. Copy it now — it won&apos;t be shown again.
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded bg-surface-200 px-3 py-2 font-mono text-xs text-white">
              {newKey}
            </code>
            <button
              onClick={handleCopyKey}
              className="rounded-md bg-emerald-500 px-3 py-2 text-xs font-medium text-white hover:bg-emerald-600 transition-colors"
            >
              {copiedKey ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
            </button>
          </div>
          <button
            onClick={() => setNewKey(null)}
            className="mt-2 text-xs text-neutral-500 hover:text-white transition-colors"
          >
            Dismiss
          </button>
        </div>
      )}

      {showCreate && (
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <h3 className="mb-3 text-sm font-medium text-neutral-400">Create API Key</h3>
          <div className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="CI Pipeline"
                className="w-full max-w-sm rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-400">Scopes</label>
              <div className="flex gap-2">
                {["read", "deploy", "admin"].map((scope) => (
                  <button
                    key={scope}
                    onClick={() => toggleScope(scope)}
                    className={`rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                      scopes.includes(scope)
                        ? "bg-accent-500/20 text-accent-400 border border-accent-500/30"
                        : "bg-surface-200 text-neutral-500 border border-border hover:text-white"
                    }`}
                  >
                    {scope}
                  </button>
                ))}
              </div>
            </div>
            <div className="flex gap-2 pt-2">
              <button
                onClick={() => setShowCreate(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={!name.trim() || creating}
                className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
              >
                {creating ? (
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                ) : (
                  <KeyRound className="h-4 w-4" />
                )}
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      {items.length === 0 ? (
        <EmptyState
          title="No API keys"
          description="Create an API key for CI/CD integration or programmatic access."
        />
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-100">
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Name</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Key</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Scopes</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Last Used</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Created</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500 w-20">Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((key) => (
                <tr key={key.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                  <td className="whitespace-nowrap px-4 py-3 font-medium text-white">{key.name}</td>
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                    {key.key_prefix}...
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <div className="flex gap-1">
                      {key.scopes.map((scope) => (
                        <span
                          key={scope}
                          className="rounded bg-surface-300 px-1.5 py-0.5 text-[10px] text-neutral-400"
                        >
                          {scope}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-500">
                    {key.last_used_at ? new Date(key.last_used_at).toLocaleDateString() : "Never"}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-500">
                    {new Date(key.created_at).toLocaleDateString()}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <button
                      onClick={() => handleDelete(key.id)}
                      disabled={deletingId === key.id}
                      className="rounded p-1 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 transition-colors disabled:opacity-50"
                      title="Revoke key"
                    >
                      {deletingId === key.id ? (
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

function MFATab() {
  const { data, loading, error, refetch } = useApi(() => mfa.getStatus(), []);
  const [enabling, setEnabling] = useState(false);
  const [enrollData, setEnrollData] = useState<{ secret: string; otpauth_uri: string; backup_codes: string[] } | null>(null);
  const [verifyCode, setVerifyCode] = useState("");
  const [verifying, setVerifying] = useState(false);
  const [showBackupCodes, setShowBackupCodes] = useState(false);
  const [disableCode, setDisableCode] = useState("");
  const [disabling, setDisabling] = useState(false);
  const [showDisable, setShowDisable] = useState(false);
  const [regenerating, setRegenerating] = useState(false);
  const [newBackupCodes, setNewBackupCodes] = useState<string[] | null>(null);

  if (loading) return <PageWithTableSkeleton cols={3} rows={2} />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;

  const status = data?.status || "disabled";
  const backupCount = data?.backup_codes || 0;

  const handleEnable = async () => {
    setEnabling(true);
    try {
      const result = await mfa.enable();
      setEnrollData(result);
    } catch {
      // Plan restriction or error
    } finally {
      setEnabling(false);
    }
  };

  const handleVerify = async () => {
    if (verifyCode.length !== 6) return;
    setVerifying(true);
    try {
      await mfa.verify(verifyCode);
      setEnrollData(null);
      setVerifyCode("");
      refetch();
    } catch {
      // Invalid code
    } finally {
      setVerifying(false);
    }
  };

  const handleDisable = async () => {
    if (!disableCode) return;
    setDisabling(true);
    try {
      await mfa.disable(disableCode);
      setShowDisable(false);
      setDisableCode("");
      refetch();
    } catch {
      // Invalid code
    } finally {
      setDisabling(false);
    }
  };

  const handleRegenerate = async () => {
    setRegenerating(true);
    try {
      const result = await mfa.regenerateBackupCodes();
      setNewBackupCodes(result.backup_codes);
    } catch {
      // Error
    } finally {
      setRegenerating(false);
    }
  };

  return (
    <div className="space-y-4">
      <p className="text-xs text-neutral-500">
        Two-factor authentication adds an extra layer of security. Requires Pro plan or higher.
      </p>

      {/* Enrollment flow */}
      {enrollData && (
        <div className="rounded-lg border border-accent-500/30 bg-accent-500/5 p-5 space-y-4">
          <h3 className="text-sm font-medium text-white">Set up authenticator app</h3>
          <p className="text-xs text-neutral-400">
            Scan this QR code with your authenticator app (Google Authenticator, Authy, 1Password), or enter the secret manually.
          </p>
          <div className="rounded-lg bg-surface-200 p-4">
            <p className="mb-1 text-[10px] font-medium text-neutral-500 uppercase">Manual entry secret</p>
            <code className="block font-mono text-sm text-accent-400 break-all">{enrollData.secret}</code>
          </div>
          <div>
            <p className="mb-2 text-xs font-medium text-neutral-400">Backup codes (save these somewhere safe)</p>
            <div className="grid grid-cols-2 gap-1.5">
              {enrollData.backup_codes.map((code) => (
                <code key={code} className="rounded bg-surface-200 px-2 py-1 text-center font-mono text-xs text-neutral-300">
                  {code}
                </code>
              ))}
            </div>
          </div>
          <div className="flex items-center gap-2 pt-2">
            <input
              type="text"
              value={verifyCode}
              onChange={(e) => setVerifyCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
              placeholder="Enter 6-digit code"
              className="w-40 rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none font-mono"
            />
            <button
              onClick={handleVerify}
              disabled={verifyCode.length !== 6 || verifying}
              className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
            >
              {verifying ? "Verifying..." : "Verify & Enable"}
            </button>
            <button
              onClick={() => { setEnrollData(null); setVerifyCode(""); }}
              className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Status card */}
      {!enrollData && (
        <div className="rounded-lg border border-border bg-surface-100 p-5">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className={`flex h-10 w-10 items-center justify-center rounded-full ${
                status === "enabled" ? "bg-emerald-500/20" : "bg-surface-200"
              }`}>
                <ShieldCheck className={`h-5 w-5 ${status === "enabled" ? "text-emerald-400" : "text-neutral-500"}`} />
              </div>
              <div>
                <p className="text-sm font-medium text-white">
                  Two-Factor Authentication
                </p>
                <p className="text-xs text-neutral-500">
                  {status === "enabled"
                    ? `Enabled — ${backupCount} backup codes remaining`
                    : "Not enabled"}
                </p>
              </div>
            </div>
            {status === "enabled" ? (
              <button
                onClick={() => setShowDisable(true)}
                className="rounded-lg border border-red-500/30 px-3 py-1.5 text-xs text-red-400 hover:bg-red-500/10 transition-colors"
              >
                Disable
              </button>
            ) : (
              <button
                onClick={handleEnable}
                disabled={enabling}
                className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
              >
                {enabling ? "Setting up..." : "Enable MFA"}
              </button>
            )}
          </div>

          {/* Disable form */}
          {showDisable && (
            <div className="mt-4 pt-4 border-t border-border space-y-3">
              <p className="text-xs text-neutral-400">Enter a TOTP code or backup code to disable MFA.</p>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={disableCode}
                  onChange={(e) => setDisableCode(e.target.value)}
                  placeholder="Code"
                  className="w-40 rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none font-mono"
                />
                <button
                  onClick={handleDisable}
                  disabled={!disableCode || disabling}
                  className="rounded-lg bg-red-500/30 px-4 py-2 text-sm font-medium text-red-400 hover:bg-red-500/40 disabled:opacity-50 transition-colors"
                >
                  {disabling ? "Disabling..." : "Confirm Disable"}
                </button>
                <button
                  onClick={() => { setShowDisable(false); setDisableCode(""); }}
                  className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
                >
                  Cancel
                </button>
              </div>
            </div>
          )}

          {/* Regenerate backup codes */}
          {status === "enabled" && !showDisable && (
            <div className="mt-4 pt-4 border-t border-border">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs font-medium text-neutral-400">Backup Codes</p>
                  <p className="text-[11px] text-neutral-500">Generate new backup codes (invalidates old ones).</p>
                </div>
                <button
                  onClick={handleRegenerate}
                  disabled={regenerating}
                  className="rounded-lg border border-border px-3 py-1.5 text-xs text-neutral-400 hover:text-white disabled:opacity-50 transition-colors"
                >
                  {regenerating ? "Regenerating..." : "Regenerate Codes"}
                </button>
              </div>
              {newBackupCodes && (
                <div className="mt-3 rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-3">
                  <div className="flex items-center justify-between mb-2">
                    <p className="text-xs font-medium text-emerald-400">New backup codes generated. Save them now.</p>
                    <button
                      onClick={() => setShowBackupCodes(!showBackupCodes)}
                      className="text-neutral-400 hover:text-white transition-colors"
                    >
                      {showBackupCodes ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </button>
                  </div>
                  {showBackupCodes && (
                    <div className="grid grid-cols-2 gap-1.5">
                      {newBackupCodes.map((code) => (
                        <code key={code} className="rounded bg-surface-200 px-2 py-1 text-center font-mono text-xs text-neutral-300">
                          {code}
                        </code>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function WebhooksTab() {
  const { data, loading, error, refetch } = useApi(() => webhooks.list(), []);
  const [showCreate, setShowCreate] = useState(false);
  const [url, setUrl] = useState("");
  const [selectedEvents, setSelectedEvents] = useState<string[]>(["deploy.success"]);
  const [creating, setCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [viewingDeliveries, setViewingDeliveries] = useState<string | null>(null);
  const [deliveries, setDeliveries] = useState<WebhookDelivery[]>([]);
  const [loadingDeliveries, setLoadingDeliveries] = useState(false);

  const allEvents = ["deploy.success", "deploy.failed", "app.sleeping", "app.waking", "db.created", "limit.reached"];

  if (loading) return <PageWithTableSkeleton cols={4} rows={3} />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;

  const items: UserWebhook[] = data?.items || [];

  const handleCreate = async () => {
    if (!url.trim() || selectedEvents.length === 0) return;
    setCreating(true);
    try {
      await webhooks.create(url.trim(), selectedEvents);
      setShowCreate(false);
      setUrl("");
      setSelectedEvents(["deploy.success"]);
      refetch();
    } catch {
      // Error
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    setDeletingId(id);
    try {
      await webhooks.delete(id);
      refetch();
    } catch {
      // Error
    } finally {
      setDeletingId(null);
    }
  };

  const handleViewDeliveries = async (id: string) => {
    if (viewingDeliveries === id) {
      setViewingDeliveries(null);
      return;
    }
    setLoadingDeliveries(true);
    setViewingDeliveries(id);
    try {
      const result = await webhooks.listDeliveries(id);
      setDeliveries(result.items || []);
    } catch {
      setDeliveries([]);
    } finally {
      setLoadingDeliveries(false);
    }
  };

  const toggleEvent = (event: string) => {
    setSelectedEvents((prev) =>
      prev.includes(event) ? prev.filter((e) => e !== event) : [...prev, event]
    );
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-xs text-neutral-500">
          Receive HTTP callbacks when events happen. Requires Pro plan or higher.
        </p>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
        >
          <Plus className="h-4 w-4" />
          Add Webhook
        </button>
      </div>

      {showCreate && (
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <h3 className="mb-3 text-sm font-medium text-neutral-400">New Webhook</h3>
          <div className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Endpoint URL</label>
              <input
                type="url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://example.com/webhooks"
                className="w-full max-w-lg rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-400">Events</label>
              <div className="flex flex-wrap gap-2">
                {allEvents.map((event) => (
                  <button
                    key={event}
                    onClick={() => toggleEvent(event)}
                    className={`rounded-md px-2.5 py-1.5 text-xs font-medium transition-colors ${
                      selectedEvents.includes(event)
                        ? "bg-accent-500/20 text-accent-400 border border-accent-500/30"
                        : "bg-surface-200 text-neutral-500 border border-border hover:text-white"
                    }`}
                  >
                    {event}
                  </button>
                ))}
              </div>
            </div>
            <div className="flex gap-2 pt-2">
              <button
                onClick={() => setShowCreate(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={!url.trim() || selectedEvents.length === 0 || creating}
                className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
              >
                {creating ? (
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                ) : (
                  <Webhook className="h-4 w-4" />
                )}
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      {items.length === 0 ? (
        <EmptyState
          title="No webhooks"
          description="Add a webhook to receive HTTP callbacks for deployment and app events."
        />
      ) : (
        <div className="space-y-3">
          {items.map((webhook) => (
            <div key={webhook.id} className="rounded-lg border border-border bg-surface-100">
              <div className="flex items-center justify-between p-4">
                <div className="flex items-center gap-3 min-w-0">
                  <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${
                    webhook.active ? "bg-emerald-500/20" : "bg-surface-200"
                  }`}>
                    {webhook.active ? (
                      <ToggleRight className="h-4 w-4 text-emerald-400" />
                    ) : (
                      <ToggleLeft className="h-4 w-4 text-neutral-500" />
                    )}
                  </div>
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-white truncate">{webhook.url}</p>
                    <div className="mt-1 flex flex-wrap gap-1">
                      {webhook.events.map((event) => (
                        <span key={event} className="rounded bg-surface-300 px-1.5 py-0.5 text-[10px] text-neutral-400">
                          {event}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0 ml-4">
                  <button
                    onClick={() => handleViewDeliveries(webhook.id)}
                    className="rounded p-1.5 text-neutral-500 hover:bg-surface-200 hover:text-white transition-colors"
                    title="View deliveries"
                  >
                    <Eye className="h-3.5 w-3.5" />
                  </button>
                  <button
                    onClick={() => handleDelete(webhook.id)}
                    disabled={deletingId === webhook.id}
                    className="rounded p-1.5 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 transition-colors disabled:opacity-50"
                    title="Delete webhook"
                  >
                    {deletingId === webhook.id ? (
                      <div className="h-3.5 w-3.5 animate-spin rounded-full border border-red-400 border-t-transparent" />
                    ) : (
                      <Trash2 className="h-3.5 w-3.5" />
                    )}
                  </button>
                </div>
              </div>

              {viewingDeliveries === webhook.id && (
                <div className="border-t border-border p-4">
                  <h4 className="mb-2 text-xs font-medium text-neutral-400">Recent Deliveries</h4>
                  {loadingDeliveries ? (
                    <div className="flex items-center gap-2 text-xs text-neutral-500">
                      <div className="h-3 w-3 animate-spin rounded-full border border-neutral-400 border-t-transparent" />
                      Loading...
                    </div>
                  ) : deliveries.length === 0 ? (
                    <p className="text-xs text-neutral-500">No deliveries yet.</p>
                  ) : (
                    <div className="space-y-2">
                      {deliveries.map((d) => (
                        <div key={d.id} className="flex items-center justify-between rounded bg-surface-200 px-3 py-2">
                          <div className="flex items-center gap-3">
                            <span className={`inline-block h-2 w-2 rounded-full ${
                              d.status === "success" ? "bg-emerald-400" : "bg-red-400"
                            }`} />
                            <span className="text-xs text-neutral-300">{d.event}</span>
                            {d.status_code && (
                              <span className="text-[10px] text-neutral-500">{d.status_code}</span>
                            )}
                          </div>
                          <span className="text-[10px] text-neutral-500">
                            {new Date(d.created_at).toLocaleString()}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function SessionsTab() {
  const { toast } = useToast();
  const { data, loading, error, refetch } = useApi(() => sessions.list(), []);
  const [revokingId, setRevokingId] = useState<string | null>(null);
  const [revokingAll, setRevokingAll] = useState(false);

  if (loading) return <PageWithTableSkeleton cols={5} rows={3} />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;

  const items: Session[] = data?.items || [];

  const handleRevoke = async (id: string) => {
    setRevokingId(id);
    try {
      await sessions.revoke(id);
      refetch();
    } catch {
      toast("error", "Failed to revoke session");
    } finally {
      setRevokingId(null);
    }
  };

  const handleRevokeAll = async () => {
    setRevokingAll(true);
    try {
      await sessions.revokeAll();
      refetch();
    } catch {
      toast("error", "Failed to revoke all sessions");
    } finally {
      setRevokingAll(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-xs text-neutral-500">
          Active sessions across your devices. Revoke sessions you don&apos;t recognize.
        </p>
        {items.length > 0 && (
          <button
            onClick={handleRevokeAll}
            disabled={revokingAll}
            className="rounded-lg border border-red-500/30 px-3 py-1.5 text-xs text-red-400 hover:bg-red-500/10 disabled:opacity-50 transition-colors"
          >
            {revokingAll ? "Revoking..." : "Revoke All"}
          </button>
        )}
      </div>

      {items.length === 0 ? (
        <EmptyState
          title="No active sessions"
          description="Session tracking will show your active login sessions."
        />
      ) : (
        <div className="space-y-2">
          {items.map((session) => (
            <div
              key={session.id}
              className={`flex items-center justify-between rounded-lg border p-4 transition-colors ${
                session.current
                  ? "border-accent-500/30 bg-accent-500/5"
                  : "border-border bg-surface-100"
              }`}
            >
              <div className="flex items-center gap-4">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-surface-200">
                  {session.device === "Mobile" ? (
                    <Smartphone className="h-5 w-5 text-neutral-400" />
                  ) : (
                    <Monitor className="h-5 w-5 text-neutral-400" />
                  )}
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-white">{session.device}</span>
                    {session.current && (
                      <span className="rounded-full bg-accent-500/20 px-2 py-0.5 text-[10px] font-medium text-accent-400">
                        Current
                      </span>
                    )}
                  </div>
                  <div className="mt-0.5 text-xs text-neutral-500">
                    {session.ip_address} &middot; Last seen {new Date(session.last_seen_at).toLocaleString()}
                  </div>
                </div>
              </div>
              {!session.current && (
                <button
                  onClick={() => handleRevoke(session.id)}
                  disabled={revokingId === session.id}
                  className="rounded-lg border border-red-500/30 px-3 py-1.5 text-xs text-red-400 hover:bg-red-500/10 disabled:opacity-50 transition-colors"
                >
                  {revokingId === session.id ? "Revoking..." : "Revoke"}
                </button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function SecurityTab() {
  const { data: complianceData, loading: complianceLoading } = useApi(() => compliance.getStatus(), []);
  const { data: ipData, loading: ipLoading, refetch: ipRefetch } = useApi(() => ipWhitelist.list(), []);
  const [showAddIP, setShowAddIP] = useState(false);
  const [cidr, setCidr] = useState("");
  const [description, setDescription] = useState("");
  const [adding, setAdding] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const handleAddIP = async () => {
    if (!cidr.trim()) return;
    setAdding(true);
    try {
      await ipWhitelist.add(cidr.trim(), description.trim());
      setShowAddIP(false);
      setCidr("");
      setDescription("");
      ipRefetch();
    } catch {
      // Error
    } finally {
      setAdding(false);
    }
  };

  const handleDeleteIP = async (id: string) => {
    setDeletingId(id);
    try {
      await ipWhitelist.delete(id);
      ipRefetch();
    } catch {
      // Error
    } finally {
      setDeletingId(null);
    }
  };

  const ipEntries: IPWhitelistEntry[] = ipData?.items || [];
  const checks: ComplianceCheck[] = complianceData?.checks || [];
  const summary = complianceData?.summary;

  const statusIcon = (status: string) => {
    switch (status) {
      case "pass": return <CheckCircle2 className="h-4 w-4 text-emerald-400" />;
      case "fail": return <XCircle className="h-4 w-4 text-red-400" />;
      case "partial": return <AlertCircle className="h-4 w-4 text-amber-400" />;
      default: return <MinusCircle className="h-4 w-4 text-neutral-500" />;
    }
  };

  return (
    <div className="space-y-6">
      {/* Compliance Overview */}
      <div>
        <h3 className="mb-3 text-sm font-medium text-white">Compliance Status</h3>
        {complianceLoading ? (
          <PageWithTableSkeleton cols={3} rows={4} />
        ) : (
          <>
            {summary && (
              <div className="mb-4 flex gap-3">
                <div className="flex items-center gap-1.5 rounded-lg bg-emerald-500/10 px-3 py-1.5 text-xs text-emerald-400">
                  <CheckCircle2 className="h-3.5 w-3.5" /> {summary.pass} passed
                </div>
                {summary.fail > 0 && (
                  <div className="flex items-center gap-1.5 rounded-lg bg-red-500/10 px-3 py-1.5 text-xs text-red-400">
                    <XCircle className="h-3.5 w-3.5" /> {summary.fail} failed
                  </div>
                )}
                {summary.na > 0 && (
                  <div className="flex items-center gap-1.5 rounded-lg bg-neutral-500/10 px-3 py-1.5 text-xs text-neutral-400">
                    <MinusCircle className="h-3.5 w-3.5" /> {summary.na} N/A
                  </div>
                )}
              </div>
            )}
            <div className="space-y-2">
              {checks.map((check, i) => (
                <div key={i} className="flex items-center justify-between rounded-lg border border-border bg-surface-100 px-4 py-3">
                  <div className="flex items-center gap-3">
                    {statusIcon(check.status)}
                    <div>
                      <p className="text-sm text-white">{check.item}</p>
                      <p className="text-[11px] text-neutral-500">{check.description}</p>
                    </div>
                  </div>
                  <span className="rounded bg-surface-300 px-1.5 py-0.5 text-[10px] text-neutral-400">
                    {check.category}
                  </span>
                </div>
              ))}
            </div>
          </>
        )}
      </div>

      {/* IP Whitelisting */}
      <div>
        <div className="mb-3 flex items-center justify-between">
          <div>
            <h3 className="text-sm font-medium text-white">IP Whitelisting</h3>
            <p className="text-[11px] text-neutral-500">Restrict access to specific IP ranges. Enterprise only.</p>
          </div>
          <button
            onClick={() => setShowAddIP(true)}
            className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
          >
            <Plus className="h-4 w-4" />
            Add IP
          </button>
        </div>

        {showAddIP && (
          <div className="mb-3 rounded-lg border border-border bg-surface-100 p-4">
            <div className="space-y-3">
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">CIDR Range</label>
                <input
                  type="text"
                  value={cidr}
                  onChange={(e) => setCidr(e.target.value)}
                  placeholder="192.168.1.0/24"
                  className="w-full max-w-sm rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none font-mono"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">Description</label>
                <input
                  type="text"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Office network"
                  className="w-full max-w-sm rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                />
              </div>
              <div className="flex gap-2">
                <button onClick={() => setShowAddIP(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
                <button
                  onClick={handleAddIP}
                  disabled={!cidr.trim() || adding}
                  className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
                >
                  {adding ? "Adding..." : "Add"}
                </button>
              </div>
            </div>
          </div>
        )}

        {ipLoading ? (
          <PageWithTableSkeleton cols={3} rows={2} />
        ) : ipEntries.length === 0 ? (
          <EmptyState
            title="No IP restrictions"
            description="All IPs are currently allowed. Add CIDR ranges to restrict access."
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">CIDR</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Description</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Added</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500 w-20">Actions</th>
                </tr>
              </thead>
              <tbody>
                {ipEntries.map((entry) => (
                  <tr key={entry.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="whitespace-nowrap px-4 py-3 font-mono text-sm text-white">{entry.cidr}</td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-neutral-400">{entry.description || "—"}</td>
                    <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-500">{new Date(entry.created_at).toLocaleDateString()}</td>
                    <td className="whitespace-nowrap px-4 py-3">
                      <button
                        onClick={() => handleDeleteIP(entry.id)}
                        disabled={deletingId === entry.id}
                        className="rounded p-1 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 transition-colors disabled:opacity-50"
                      >
                        {deletingId === entry.id ? (
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

function InfrastructureTab() {
  const { data: statusData, loading: statusLoading } = useApi(() => autoscaler.getStatus(), []);
  const { data: nodesData, loading: nodesLoading } = useApi(() => autoscaler.listNodes(), []);
  const { data: eventsData, loading: eventsLoading } = useApi(() => autoscaler.listEvents(20), []);

  const status: AutoscalerStatus | null = statusData || null;
  const nodes: HetznerNode[] = nodesData?.items || [];
  const events: AutoscaleEvent[] = eventsData?.items || [];

  if (statusLoading || nodesLoading) return <PageWithTableSkeleton cols={5} rows={3} />;

  const budgetPct = status ? Math.min(100, (status.budget_used_eur / status.budget_cap_eur) * 100) : 0;

  return (
    <div className="space-y-6">
      <p className="text-xs text-neutral-500">
        Hetzner autoscaler manages worker nodes based on cluster resource utilization.
      </p>

      {/* Status Overview */}
      {status && (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-[10px] font-medium uppercase text-neutral-500">Nodes</p>
            <p className="mt-1 text-xl font-semibold text-white">{status.node_count} <span className="text-sm font-normal text-neutral-500">/ {status.max_nodes}</span></p>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-[10px] font-medium uppercase text-neutral-500">CPU</p>
            <p className={`mt-1 text-xl font-semibold ${status.cpu_percent > 80 ? "text-red-400" : status.cpu_percent > 60 ? "text-amber-400" : "text-emerald-400"}`}>
              {status.cpu_percent.toFixed(0)}%
            </p>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-[10px] font-medium uppercase text-neutral-500">RAM</p>
            <p className={`mt-1 text-xl font-semibold ${status.ram_percent > 80 ? "text-red-400" : status.ram_percent > 60 ? "text-amber-400" : "text-emerald-400"}`}>
              {status.ram_percent.toFixed(0)}%
            </p>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-[10px] font-medium uppercase text-neutral-500">Budget</p>
            <p className="mt-1 text-xl font-semibold text-white">&euro;{status.budget_used_eur.toFixed(0)} <span className="text-sm font-normal text-neutral-500">/ &euro;{status.budget_cap_eur}</span></p>
            <div className="mt-2 h-1.5 w-full rounded-full bg-surface-300">
              <div
                className={`h-full rounded-full transition-all ${budgetPct > 80 ? "bg-red-400" : budgetPct > 60 ? "bg-amber-400" : "bg-accent-500"}`}
                style={{ width: `${budgetPct}%` }}
              />
            </div>
          </div>
        </div>
      )}

      {/* Autoscaler Indicator */}
      {status && (
        <div className="flex items-center gap-3 rounded-lg border border-border bg-surface-100 px-4 py-3">
          <Activity className={`h-4 w-4 ${status.enabled ? "text-emerald-400" : "text-neutral-500"}`} />
          <span className="text-sm text-white">{status.enabled ? "Autoscaler active" : "Autoscaler disabled"}</span>
          {status.last_check_at && (
            <span className="text-[11px] text-neutral-500 ml-auto">
              Last check: {new Date(status.last_check_at).toLocaleString()}
            </span>
          )}
        </div>
      )}

      {/* Node List */}
      <div>
        <h3 className="mb-3 text-sm font-medium text-white">Worker Nodes</h3>
        {nodes.length === 0 ? (
          <EmptyState title="No managed nodes" description="The autoscaler will provision Hetzner servers when cluster utilization exceeds thresholds." />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Name</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">IP</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Type</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">CPU</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">RAM</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Cost/mo</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Status</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Created</th>
                </tr>
              </thead>
              <tbody>
                {nodes.map((node) => (
                  <tr key={node.server_id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="whitespace-nowrap px-4 py-3 font-medium text-white">{node.name}</td>
                    <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">{node.ip}</td>
                    <td className="whitespace-nowrap px-4 py-3 text-neutral-400">{node.server_type}</td>
                    <td className="whitespace-nowrap px-4 py-3 text-neutral-400">{node.cpu_cores} cores</td>
                    <td className="whitespace-nowrap px-4 py-3 text-neutral-400">{(node.ram_mb / 1024).toFixed(0)} GB</td>
                    <td className="whitespace-nowrap px-4 py-3 text-neutral-400">&euro;{node.monthly_cost.toFixed(2)}</td>
                    <td className="whitespace-nowrap px-4 py-3">
                      <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${
                        node.status === "running" ? "bg-emerald-500/20 text-emerald-400" :
                        node.status === "provisioning" ? "bg-amber-500/20 text-amber-400" :
                        "bg-neutral-500/20 text-neutral-400"
                      }`}>{node.status}</span>
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-500">
                      {new Date(node.created_at).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Scale History */}
      <div>
        <h3 className="mb-3 text-sm font-medium text-white">Scale History</h3>
        {eventsLoading ? (
          <PageWithTableSkeleton cols={4} rows={3} />
        ) : events.length === 0 ? (
          <EmptyState title="No scale events" description="Scale events will appear here when the autoscaler adjusts node count." />
        ) : (
          <div className="space-y-2">
            {events.map((event) => (
              <div key={event.id} className="flex items-center gap-3 rounded-lg border border-border bg-surface-100 px-4 py-3">
                {event.action === "scale_up" ? (
                  <ArrowUpCircle className="h-4 w-4 shrink-0 text-emerald-400" />
                ) : (
                  <ArrowDownCircle className="h-4 w-4 shrink-0 text-amber-400" />
                )}
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-white">
                      {event.action === "scale_up" ? "Scaled up" : "Scaled down"}: {event.old_count} → {event.new_count} nodes
                    </span>
                    <span className="rounded bg-surface-300 px-1.5 py-0.5 text-[10px] text-neutral-400">
                      {event.server_name}
                    </span>
                  </div>
                  <p className="mt-0.5 truncate text-[11px] text-neutral-500">{event.reason}</p>
                </div>
                <span className="shrink-0 text-[10px] text-neutral-500">
                  {new Date(event.timestamp).toLocaleString()}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function GeneralTab() {
  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-border bg-surface-100 p-5 space-y-4">
        <div>
          <label className="mb-1.5 block text-xs font-medium text-neutral-500">Email</label>
          <input
            type="text"
            readOnly
            value="demo@zenith.dev"
            className="w-full max-w-sm rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-neutral-300 outline-none cursor-not-allowed"
          />
        </div>
        <div>
          <label className="mb-1.5 block text-xs font-medium text-neutral-500">Name</label>
          <input
            type="text"
            readOnly
            value="Demo User"
            className="w-full max-w-sm rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-neutral-300 outline-none cursor-not-allowed"
          />
        </div>
      </div>

      {/* Danger Zone */}
      <div className="rounded-lg border border-red-500/30 bg-red-500/5 p-5">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm font-medium text-white">Delete Account</p>
            <p className="mt-0.5 text-xs text-neutral-500">
              Permanently delete your account and all associated resources.
            </p>
          </div>
          <button className="rounded-lg bg-red-500/30 px-3 py-1.5 text-sm font-medium text-red-400 cursor-not-allowed opacity-50">
            Delete Account
          </button>
        </div>
      </div>
    </div>
  );
}
