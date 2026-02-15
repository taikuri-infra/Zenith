import { Shell } from "@/components/shell";
import { mockPlatformUpdate } from "@/lib/mock-data";

export default function UpdatesPage() {
  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Platform Updates</h1>

        {/* Available update */}
        <div className="rounded-lg border border-accent-600/30 bg-accent-600/5 p-5">
          <div className="flex items-start justify-between">
            <div>
              <div className="flex items-center gap-2">
                <span className="rounded bg-accent-600/20 px-1.5 py-0.5 text-[10px] font-medium text-accent-400">
                  NEW
                </span>
                <h2 className="text-base font-semibold text-white">
                  Zenith {mockPlatformUpdate.version}
                </h2>
              </div>
              <p className="mt-1 text-sm text-neutral-500">
                Released {mockPlatformUpdate.releasedAt} &middot; Current version: {mockPlatformUpdate.current}
              </p>
              {mockPlatformUpdate.breakingChanges && (
                <p className="mt-1 text-xs text-amber-400">Contains breaking changes</p>
              )}
            </div>
            <button className="rounded-lg bg-accent-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-500">
              Upgrade to {mockPlatformUpdate.version}
            </button>
          </div>

          <div className="mt-4">
            <h3 className="text-sm font-medium text-white">New Features</h3>
            <ul className="mt-2 space-y-1.5">
              {mockPlatformUpdate.features.map((feature, i) => (
                <li key={i} className="flex items-start gap-2 text-sm text-neutral-300">
                  <span className="mt-1.5 h-1 w-1 flex-shrink-0 rounded-full bg-accent-400" />
                  {feature}
                </li>
              ))}
            </ul>
          </div>
        </div>

        {/* Update history */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Update History</h2>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Version</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Date</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                </tr>
              </thead>
              <tbody>
                <tr className="border-b border-border transition-colors hover:bg-surface-200">
                  <td className="px-4 py-3 font-mono text-xs text-white">v1.2.1</td>
                  <td className="px-4 py-3 text-neutral-400">January 28, 2026</td>
                  <td className="px-4 py-3 text-xs text-emerald-400">Installed (current)</td>
                </tr>
                <tr className="border-b border-border transition-colors hover:bg-surface-200">
                  <td className="px-4 py-3 font-mono text-xs text-white">v1.2.0</td>
                  <td className="px-4 py-3 text-neutral-400">January 12, 2026</td>
                  <td className="px-4 py-3 text-xs text-neutral-500">Superseded</td>
                </tr>
                <tr className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                  <td className="px-4 py-3 font-mono text-xs text-white">v1.1.0</td>
                  <td className="px-4 py-3 text-neutral-400">December 5, 2025</td>
                  <td className="px-4 py-3 text-xs text-neutral-500">Superseded</td>
                </tr>
                <tr className="transition-colors hover:bg-surface-200">
                  <td className="px-4 py-3 font-mono text-xs text-white">v1.0.0</td>
                  <td className="px-4 py-3 text-neutral-400">November 1, 2025</td>
                  <td className="px-4 py-3 text-xs text-neutral-500">Superseded</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </Shell>
  );
}
