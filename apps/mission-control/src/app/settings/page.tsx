import { Shell } from "@/components/shell";

export default function SettingsPage() {
  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Settings</h1>

        <div className="space-y-4">
          {/* General section */}
          <section className="rounded-lg border border-border bg-surface-100 p-5">
            <h2 className="text-sm font-medium text-white">General</h2>
            <p className="mt-1 text-xs text-neutral-500">Platform-wide configuration options.</p>
            <div className="mt-4 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-neutral-300">Platform Name</p>
                  <p className="text-xs text-neutral-500">Display name for this Zenith instance</p>
                </div>
                <input
                  type="text"
                  defaultValue="Zenith Platform"
                  readOnly
                  className="rounded-lg border border-border bg-surface-200 px-3 py-1.5 text-sm text-neutral-300 outline-none"
                />
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-neutral-300">Base Domain</p>
                  <p className="text-xs text-neutral-500">Root domain for all tenant applications</p>
                </div>
                <input
                  type="text"
                  defaultValue="zenith.local"
                  readOnly
                  className="rounded-lg border border-border bg-surface-200 px-3 py-1.5 text-sm text-neutral-300 outline-none"
                />
              </div>
            </div>
          </section>

          {/* Cloud Provider section */}
          <section className="rounded-lg border border-border bg-surface-100 p-5">
            <h2 className="text-sm font-medium text-white">Cloud Provider</h2>
            <p className="mt-1 text-xs text-neutral-500">Infrastructure provider settings.</p>
            <div className="mt-4 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-neutral-300">Provider</p>
                  <p className="text-xs text-neutral-500">Current cloud infrastructure provider</p>
                </div>
                <span className="rounded-full border border-border bg-surface-200 px-3 py-1 text-sm text-neutral-300">
                  Hetzner Cloud
                </span>
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-neutral-300">Default Region</p>
                  <p className="text-xs text-neutral-500">Default datacenter for new clusters</p>
                </div>
                <span className="rounded-full border border-border bg-surface-200 px-3 py-1 text-sm text-neutral-300">
                  fsn1 (Falkenstein)
                </span>
              </div>
            </div>
          </section>

          {/* Backup section */}
          <section className="rounded-lg border border-border bg-surface-100 p-5">
            <h2 className="text-sm font-medium text-white">Backups</h2>
            <p className="mt-1 text-xs text-neutral-500">Database and volume backup configuration.</p>
            <div className="mt-4 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-neutral-300">Automatic Backups</p>
                  <p className="text-xs text-neutral-500">Nightly backups of all tenant databases</p>
                </div>
                <span className="text-sm text-emerald-400">Enabled</span>
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-neutral-300">Retention Period</p>
                  <p className="text-xs text-neutral-500">How long backups are kept</p>
                </div>
                <span className="text-sm text-neutral-300">30 days</span>
              </div>
            </div>
          </section>
        </div>
      </div>
    </Shell>
  );
}
