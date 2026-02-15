import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { mockClusters, mockModules, mockPlatformUpdate } from "@/lib/mock-data";

export default function StatePage() {
  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Platform State</h1>

        {/* Platform overview */}
        <div className="grid grid-cols-4 gap-4">
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-xs font-medium text-neutral-500">Platform Version</p>
            <p className="mt-1 font-mono text-lg font-semibold text-white">{mockPlatformUpdate.current}</p>
            {mockPlatformUpdate.version !== mockPlatformUpdate.current && (
              <p className="mt-0.5 text-xs text-amber-400">Update available: {mockPlatformUpdate.version}</p>
            )}
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-xs font-medium text-neutral-500">Installed Date</p>
            <p className="mt-1 text-lg font-semibold text-white">Nov 1, 2025</p>
            <p className="mt-0.5 text-xs text-neutral-500">106 days ago</p>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-xs font-medium text-neutral-500">Management K8s</p>
            <p className="mt-1 font-mono text-lg font-semibold text-white">v1.30.2</p>
            <p className="mt-0.5 text-xs text-neutral-500">Up to date</p>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-xs font-medium text-neutral-500">Domain</p>
            <p className="mt-1 text-lg font-semibold text-white">zenith.local</p>
            <p className="mt-0.5 text-xs text-neutral-500">Wildcard TLS active</p>
          </div>
        </div>

        {/* Clusters table */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Clusters</h2>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Type</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">K8s Version</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Region</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Nodes</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                </tr>
              </thead>
              <tbody>
                {mockClusters.map((cluster) => (
                  <tr
                    key={cluster.name}
                    className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                  >
                    <td className="px-4 py-3 font-medium text-white">{cluster.name}</td>
                    <td className="px-4 py-3 text-neutral-400 capitalize">{cluster.type}</td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">{cluster.k8sVersion}</td>
                    <td className="px-4 py-3 text-neutral-400">{cluster.region}</td>
                    <td className="px-4 py-3 text-neutral-300">{cluster.nodes}</td>
                    <td className="px-4 py-3">
                      <StatusBadge status={cluster.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Modules per cluster - version matrix */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Module Versions</h2>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Module</th>
                  {mockClusters.map((c) => (
                    <th key={c.name} className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                      {c.name}
                    </th>
                  ))}
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Latest</th>
                </tr>
              </thead>
              <tbody>
                {mockModules.map((mod) => (
                  <tr
                    key={mod.name}
                    className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                  >
                    <td className="px-4 py-3 font-medium text-white">{mod.name}</td>
                    {mockClusters.map((c) => (
                      <td key={c.name} className="px-4 py-3 font-mono text-xs text-neutral-400">
                        {mod.installed}
                      </td>
                    ))}
                    <td className="px-4 py-3 font-mono text-xs">
                      <span className={mod.status === "update_available" ? "text-amber-400" : "text-neutral-400"}>
                        {mod.latest}
                      </span>
                    </td>
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
