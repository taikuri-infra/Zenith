"use client";

import { Shell } from "@/components/shell";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { Modal } from "@/components/modal";
import { useApi } from "@/hooks/use-api";
import { type SSOConfig } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import Link from "next/link";
import { useState } from "react";
import { Shield, Plus, Trash2, Key, Lock } from "lucide-react";

function providerBadge(provider: string) {
  switch (provider) {
    case "saml":
      return "bg-purple-500/15 text-purple-400";
    case "oidc":
      return "bg-blue-500/15 text-blue-400";
    default:
      return "bg-neutral-500/15 text-neutral-400";
  }
}

function statusBadge(enabled: boolean) {
  return enabled
    ? "bg-green-500/15 text-green-400"
    : "bg-neutral-500/15 text-neutral-400";
}

export default function SSOPage() {
  const { sso, userPlan } = getApi();

  const {
    data: configs,
    loading,
    error,
    refetch,
  } = useApi(() => sso.list(), []);

  const { data: planData, loading: planLoading } = useApi(
    () => userPlan.get(),
    []
  );

  const [showCreate, setShowCreate] = useState(false);
  const [activeTab, setActiveTab] = useState<"saml" | "oidc">("saml");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState("");

  // SAML fields
  const [samlEntityId, setSamlEntityId] = useState("");
  const [samlSsoUrl, setSamlSsoUrl] = useState("");
  const [samlCertificate, setSamlCertificate] = useState("");

  // OIDC fields
  const [oidcClientId, setOidcClientId] = useState("");
  const [oidcClientSecret, setOidcClientSecret] = useState("");
  const [oidcDiscoveryUrl, setOidcDiscoveryUrl] = useState("");

  // Delete confirmation
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  const tier = planData?.tier ?? "free";
  const isAllowed = tier === "team" || tier === "business" || tier === "enterprise";
  const configList: SSOConfig[] = configs?.items ?? [];

  const samlCount = configList.filter((c) => c.provider === "saml").length;
  const oidcCount = configList.filter((c) => c.provider === "oidc").length;
  const enabledCount = configList.filter((c) => c.enabled).length;

  const resetForm = () => {
    setSamlEntityId("");
    setSamlSsoUrl("");
    setSamlCertificate("");
    setOidcClientId("");
    setOidcClientSecret("");
    setOidcDiscoveryUrl("");
    setCreateError("");
    setActiveTab("saml");
  };

  const handleCreate = async () => {
    if (creating) return;
    setCreating(true);
    setCreateError("");
    try {
      if (activeTab === "saml") {
        if (!samlEntityId.trim() || !samlSsoUrl.trim() || !samlCertificate.trim()) {
          setCreateError("All SAML fields are required.");
          setCreating(false);
          return;
        }
        await sso.configureSAML(
          samlEntityId.trim(),
          samlSsoUrl.trim(),
          samlCertificate.trim()
        );
      } else {
        if (!oidcClientId.trim() || !oidcClientSecret.trim() || !oidcDiscoveryUrl.trim()) {
          setCreateError("All OIDC fields are required.");
          setCreating(false);
          return;
        }
        await sso.configureOIDC(
          oidcClientId.trim(),
          oidcClientSecret.trim(),
          oidcDiscoveryUrl.trim()
        );
      }
      setShowCreate(false);
      resetForm();
      refetch();
    } catch (err: unknown) {
      const status = (err as { status?: number }).status;
      if (status === 403) {
        setCreateError("SSO requires a Team plan or higher.");
      } else {
        setCreateError(
          err instanceof Error ? err.message : "Failed to configure SSO provider"
        );
      }
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteId || deleting) return;
    setDeleting(true);
    try {
      await sso.delete(deleteId);
      setDeleteId(null);
      refetch();
    } catch (err: unknown) {
      console.error("Failed to delete SSO config:", err);
    } finally {
      setDeleting(false);
    }
  };

  if (loading || planLoading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={5} rows={3} />
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

  // Plan gate — require Team or higher
  if (!isAllowed) {
    return (
      <Shell>
        <div className="space-y-6">
          <div>
            <h1 className="text-lg font-semibold text-white">Single Sign-On</h1>
            <p className="text-sm text-neutral-500">
              Configure SAML and OIDC identity providers
            </p>
          </div>

          <div className="flex flex-col items-center justify-center rounded-xl border border-border bg-surface-100 py-16 px-6">
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-accent-500/10 mb-5">
              <Shield className="h-8 w-8 text-accent-400" />
            </div>
            <h2 className="text-xl font-semibold text-white mb-2">
              Requires Team Plan or Higher
            </h2>
            <p className="text-sm text-neutral-400 text-center max-w-md mb-6">
              Single Sign-On lets your team authenticate through your
              organization&apos;s identity provider using SAML or OIDC protocols.
            </p>
            <div className="flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-xs text-neutral-500 mb-8">
              <span className="flex items-center gap-1.5">
                <svg className="h-4 w-4 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                SAML 2.0 support
              </span>
              <span className="flex items-center gap-1.5">
                <svg className="h-4 w-4 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                OIDC support
              </span>
              <span className="flex items-center gap-1.5">
                <svg className="h-4 w-4 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                Multiple providers
              </span>
              <span className="flex items-center gap-1.5">
                <svg className="h-4 w-4 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                Centralized access control
              </span>
            </div>
            <Link
              href="/billing"
              className="rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-6 py-2.5 text-sm font-medium transition-colors"
            >
              Upgrade to Team
            </Link>
          </div>
        </div>
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Single Sign-On</h1>
            <p className="text-sm text-neutral-500">
              Configure SAML and OIDC identity providers
            </p>
          </div>
          <button
            onClick={() => {
              resetForm();
              setShowCreate(true);
            }}
            className="flex items-center gap-2 rounded-lg bg-accent-500 hover:bg-accent-600 px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            <Plus className="h-4 w-4" />
            Add SSO Provider
          </button>
        </div>

        {/* Stat cards */}
        <div className="grid grid-cols-3 gap-4">
          {[
            { label: "SAML Providers", value: samlCount, color: "text-purple-400" },
            { label: "OIDC Providers", value: oidcCount, color: "text-blue-400" },
            { label: "Enabled", value: enabledCount, color: "text-green-400" },
          ].map((stat) => (
            <div
              key={stat.label}
              className="rounded-xl border border-border bg-surface-100 p-4"
            >
              <p className="text-xs text-neutral-500">{stat.label}</p>
              <p className={`text-2xl font-semibold ${stat.color}`}>
                {stat.value}
              </p>
            </div>
          ))}
        </div>

        {/* Provider list */}
        {configList.length === 0 ? (
          <EmptyState
            title="No SSO providers"
            description="You haven't configured any SSO providers yet. Add a SAML or OIDC provider to get started."
          />
        ) : (
          <div className="space-y-3">
            {configList.map((config) => (
              <div
                key={config.id}
                className="flex items-center justify-between rounded-xl border border-border bg-surface-100 p-4 transition-colors hover:bg-surface-200"
              >
                <div className="flex items-center gap-4">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-surface-200">
                    {config.provider === "saml" ? (
                      <Key className="h-5 w-5 text-purple-400" />
                    ) : (
                      <Lock className="h-5 w-5 text-blue-400" />
                    )}
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-white">
                        {config.provider === "saml"
                          ? config.entity_id || "SAML Provider"
                          : config.client_id || "OIDC Provider"}
                      </span>
                      <span
                        className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium uppercase ${providerBadge(config.provider)}`}
                      >
                        {config.provider}
                      </span>
                      <span
                        className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusBadge(config.enabled)}`}
                      >
                        {config.enabled ? "Enabled" : "Disabled"}
                      </span>
                    </div>
                    <div className="mt-1 flex items-center gap-3 text-xs text-neutral-500">
                      {config.provider === "saml" && config.sso_url && (
                        <span className="truncate max-w-xs">
                          SSO URL: {config.sso_url}
                        </span>
                      )}
                      {config.provider === "oidc" && config.discovery_url && (
                        <span className="truncate max-w-xs">
                          Discovery: {config.discovery_url}
                        </span>
                      )}
                      <span>
                        Created {new Date(config.created_at).toLocaleDateString()}
                      </span>
                    </div>
                  </div>
                </div>
                <button
                  onClick={() => setDeleteId(config.id)}
                  className="rounded-lg border border-border p-2 text-neutral-500 hover:border-red-500/50 hover:text-red-400 transition-colors"
                  title="Delete provider"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Create SSO provider modal */}
      {showCreate && (
        <Modal title="Add SSO Provider" onClose={() => setShowCreate(false)}>
          <div className="space-y-4">
            {/* Tab switcher */}
            <div className="flex rounded-lg border border-border bg-surface-100 p-1">
              <button
                onClick={() => setActiveTab("saml")}
                className={`flex-1 flex items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                  activeTab === "saml"
                    ? "bg-accent-500/15 text-accent-400"
                    : "text-neutral-400 hover:text-white"
                }`}
              >
                <Key className="h-4 w-4" />
                SAML
              </button>
              <button
                onClick={() => setActiveTab("oidc")}
                className={`flex-1 flex items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                  activeTab === "oidc"
                    ? "bg-accent-500/15 text-accent-400"
                    : "text-neutral-400 hover:text-white"
                }`}
              >
                <Lock className="h-4 w-4" />
                OIDC
              </button>
            </div>

            {/* SAML form */}
            {activeTab === "saml" && (
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-1">
                    Entity ID
                  </label>
                  <input
                    value={samlEntityId}
                    onChange={(e) => setSamlEntityId(e.target.value)}
                    placeholder="https://idp.example.com/metadata"
                    className="w-full rounded-lg border border-border bg-surface-50 px-3 py-2 text-sm text-white placeholder-neutral-500 outline-none focus:border-accent-500 focus:ring-1 focus:ring-accent-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-1">
                    SSO URL
                  </label>
                  <input
                    value={samlSsoUrl}
                    onChange={(e) => setSamlSsoUrl(e.target.value)}
                    placeholder="https://idp.example.com/sso/saml"
                    className="w-full rounded-lg border border-border bg-surface-50 px-3 py-2 text-sm text-white placeholder-neutral-500 outline-none focus:border-accent-500 focus:ring-1 focus:ring-accent-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-1">
                    Certificate
                  </label>
                  <textarea
                    value={samlCertificate}
                    onChange={(e) => setSamlCertificate(e.target.value)}
                    placeholder="-----BEGIN CERTIFICATE-----&#10;MIICmzCCAYMCBgF...&#10;-----END CERTIFICATE-----"
                    rows={5}
                    className="w-full rounded-lg border border-border bg-surface-50 px-3 py-2 text-sm text-white placeholder-neutral-500 outline-none focus:border-accent-500 focus:ring-1 focus:ring-accent-500 resize-none font-mono"
                  />
                </div>
              </div>
            )}

            {/* OIDC form */}
            {activeTab === "oidc" && (
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-1">
                    Client ID
                  </label>
                  <input
                    value={oidcClientId}
                    onChange={(e) => setOidcClientId(e.target.value)}
                    placeholder="your-client-id"
                    className="w-full rounded-lg border border-border bg-surface-50 px-3 py-2 text-sm text-white placeholder-neutral-500 outline-none focus:border-accent-500 focus:ring-1 focus:ring-accent-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-1">
                    Client Secret
                  </label>
                  <input
                    type="password"
                    value={oidcClientSecret}
                    onChange={(e) => setOidcClientSecret(e.target.value)}
                    placeholder="your-client-secret"
                    className="w-full rounded-lg border border-border bg-surface-50 px-3 py-2 text-sm text-white placeholder-neutral-500 outline-none focus:border-accent-500 focus:ring-1 focus:ring-accent-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-1">
                    Discovery URL
                  </label>
                  <input
                    value={oidcDiscoveryUrl}
                    onChange={(e) => setOidcDiscoveryUrl(e.target.value)}
                    placeholder="https://accounts.google.com/.well-known/openid-configuration"
                    className="w-full rounded-lg border border-border bg-surface-50 px-3 py-2 text-sm text-white placeholder-neutral-500 outline-none focus:border-accent-500 focus:ring-1 focus:ring-accent-500"
                  />
                </div>
              </div>
            )}

            {createError && (
              <p className="text-sm text-red-400">{createError}</p>
            )}
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowCreate(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={creating}
                className="rounded-lg bg-accent-500 hover:bg-accent-600 px-4 py-2 text-sm font-medium text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {creating ? "Configuring..." : "Configure Provider"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete confirmation modal */}
      {deleteId && (
        <Modal title="Delete SSO Provider" onClose={() => setDeleteId(null)}>
          <div className="space-y-4">
            <p className="text-sm text-neutral-400">
              Are you sure you want to delete this SSO provider? Users who
              authenticate through this provider will no longer be able to sign
              in with SSO. This action cannot be undone.
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setDeleteId(null)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                disabled={deleting}
                className="rounded-lg bg-red-600 hover:bg-red-700 px-4 py-2 text-sm font-medium text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {deleting ? "Deleting..." : "Delete Provider"}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </Shell>
  );
}
