"use client";

import { useState, useEffect } from "react";
import { Shell } from "@/components/shell";
import { ErrorState } from "@/components/error-state";
import { SettingsSectionSkeleton } from "@/components/loading-skeleton";
import { DemoButton } from "@/components/demo-button";
import { getApi, isDemoMode } from "@/lib/get-api";
import type { PlatformSettings } from "@/lib/api";
import { useApi, useMutation } from "@/hooks/use-api";

export default function SettingsPage() {
  const apiClient = getApi();
  const demo = isDemoMode();

  const {
    data: settings,
    loading,
    error,
    refetch,
  } = useApi<PlatformSettings>(() => apiClient.settings.get());

  const [platformName, setPlatformName] = useState("");
  const [baseDomain, setBaseDomain] = useState("");
  const [dirty, setDirty] = useState(false);

  // Sync local state when settings are loaded
  useEffect(() => {
    if (settings) {
      setPlatformName(settings.platformName);
      setBaseDomain(settings.baseDomain);
      setDirty(false);
    }
  }, [settings]);

  const saveMutation = useMutation((data: Partial<PlatformSettings>) =>
    apiClient.settings.update(data)
  );

  const handleSave = async () => {
    if (demo) return;
    try {
      await saveMutation.execute({ platformName, baseDomain });
      setDirty(false);
      refetch();
    } catch {
      // error is set in the mutation hook
    }
  };

  const handlePlatformNameChange = (value: string) => {
    setPlatformName(value);
    setDirty(true);
  };

  const handleBaseDomainChange = (value: string) => {
    setBaseDomain(value);
    setDirty(true);
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-white">Settings</h1>
          {dirty && (
            <DemoButton
              onClick={handleSave}
              disabled={saveMutation.loading}
              className="rounded-lg bg-accent-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500 disabled:opacity-50"
            >
              {saveMutation.loading ? "Saving..." : "Save Changes"}
            </DemoButton>
          )}
        </div>

        {saveMutation.error && (
          <div className="rounded-lg border border-red-500/20 bg-red-500/5 px-4 py-2 text-xs text-red-400">
            Failed to save settings: {saveMutation.error.message}
          </div>
        )}

        {loading ? (
          <div className="space-y-4">
            <SettingsSectionSkeleton />
            <SettingsSectionSkeleton />
            <SettingsSectionSkeleton />
          </div>
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : settings ? (
          <div className="space-y-4">
            {/* General section */}
            <section className="rounded-lg border border-border bg-surface-100 p-5">
              <h2 className="text-sm font-medium text-white">General</h2>
              <p className="mt-1 text-xs text-neutral-500">
                Platform-wide configuration options.
              </p>
              <div className="mt-4 space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-neutral-300">Platform Name</p>
                    <p className="text-xs text-neutral-500">
                      Display name for this Zenith instance
                    </p>
                  </div>
                  <input
                    type="text"
                    value={platformName}
                    onChange={(e) =>
                      handlePlatformNameChange(e.target.value)
                    }
                    readOnly={demo}
                    className={`rounded-lg border border-border bg-surface-200 px-3 py-1.5 text-sm text-neutral-300 outline-none focus:border-accent-600 ${demo ? "cursor-not-allowed opacity-60" : ""}`}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-neutral-300">Base Domain</p>
                    <p className="text-xs text-neutral-500">
                      Root domain for all tenant applications
                    </p>
                  </div>
                  <input
                    type="text"
                    value={baseDomain}
                    onChange={(e) =>
                      handleBaseDomainChange(e.target.value)
                    }
                    readOnly={demo}
                    className={`rounded-lg border border-border bg-surface-200 px-3 py-1.5 text-sm text-neutral-300 outline-none focus:border-accent-600 ${demo ? "cursor-not-allowed opacity-60" : ""}`}
                  />
                </div>
              </div>
            </section>

            {/* Cloud Provider section */}
            <section className="rounded-lg border border-border bg-surface-100 p-5">
              <h2 className="text-sm font-medium text-white">
                Cloud Provider
              </h2>
              <p className="mt-1 text-xs text-neutral-500">
                Infrastructure provider settings.
              </p>
              <div className="mt-4 space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-neutral-300">Provider</p>
                    <p className="text-xs text-neutral-500">
                      Current cloud infrastructure provider
                    </p>
                  </div>
                  <span className="rounded-full border border-border bg-surface-200 px-3 py-1 text-sm text-neutral-300">
                    {settings.provider}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-neutral-300">Default Region</p>
                    <p className="text-xs text-neutral-500">
                      Default datacenter for new clusters
                    </p>
                  </div>
                  <span className="rounded-full border border-border bg-surface-200 px-3 py-1 text-sm text-neutral-300">
                    {settings.defaultRegion} ({settings.regionLabel})
                  </span>
                </div>
              </div>
            </section>

            {/* Backup section */}
            <section className="rounded-lg border border-border bg-surface-100 p-5">
              <h2 className="text-sm font-medium text-white">Backups</h2>
              <p className="mt-1 text-xs text-neutral-500">
                Database and volume backup configuration.
              </p>
              <div className="mt-4 space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-neutral-300">
                      Automatic Backups
                    </p>
                    <p className="text-xs text-neutral-500">
                      Nightly backups of all tenant databases
                    </p>
                  </div>
                  <span
                    className={`text-sm ${
                      settings.autoBackups
                        ? "text-emerald-400"
                        : "text-neutral-500"
                    }`}
                  >
                    {settings.autoBackups ? "Enabled" : "Disabled"}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-neutral-300">
                      Retention Period
                    </p>
                    <p className="text-xs text-neutral-500">
                      How long backups are kept
                    </p>
                  </div>
                  <span className="text-sm text-neutral-300">
                    {settings.retentionDays} days
                  </span>
                </div>
              </div>
            </section>
          </div>
        ) : null}
      </div>
    </Shell>
  );
}
