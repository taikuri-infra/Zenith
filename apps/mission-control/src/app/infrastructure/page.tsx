import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";

export default function InfrastructurePage() {
  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Infrastructure</h1>

        {/* Stat cards */}
        <div className="grid grid-cols-4 gap-4">
          <StatCard label="Servers" value={25} sub="Hetzner Cloud" />
          <StatCard label="Volumes" value={89} sub="2.4 TB total" />
          <StatCard label="Load Balancers" value={4} sub="2 public, 2 internal" />
          <StatCard label="Monthly Cost" value="&euro;289.02" sub="Current billing period" />
        </div>

        {/* Resource breakdown */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Resource Breakdown</h2>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Resource</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Type</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Count</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Cluster</th>
                  <th className="px-4 py-2.5 text-right text-xs font-medium text-neutral-500">Monthly Cost</th>
                </tr>
              </thead>
              <tbody>
                <tr className="border-b border-border transition-colors hover:bg-surface-200">
                  <td className="px-4 py-3 font-medium text-white">CX31 Nodes</td>
                  <td className="px-4 py-3 text-neutral-400">Compute</td>
                  <td className="px-4 py-3 text-neutral-300">8</td>
                  <td className="px-4 py-3 text-neutral-400">zenith-shared</td>
                  <td className="px-4 py-3 text-right font-mono text-xs text-neutral-300">&euro;79.20</td>
                </tr>
                <tr className="border-b border-border transition-colors hover:bg-surface-200">
                  <td className="px-4 py-3 font-medium text-white">CX41 Nodes</td>
                  <td className="px-4 py-3 text-neutral-400">Compute</td>
                  <td className="px-4 py-3 text-neutral-300">4</td>
                  <td className="px-4 py-3 text-neutral-400">pro-startup-a</td>
                  <td className="px-4 py-3 text-right font-mono text-xs text-neutral-300">&euro;59.60</td>
                </tr>
                <tr className="border-b border-border transition-colors hover:bg-surface-200">
                  <td className="px-4 py-3 font-medium text-white">CX51 Nodes</td>
                  <td className="px-4 py-3 text-neutral-400">Compute</td>
                  <td className="px-4 py-3 text-neutral-300">12</td>
                  <td className="px-4 py-3 text-neutral-400">pro-enterprise</td>
                  <td className="px-4 py-3 text-right font-mono text-xs text-neutral-300">&euro;95.40</td>
                </tr>
                <tr className="border-b border-border transition-colors hover:bg-surface-200">
                  <td className="px-4 py-3 font-medium text-white">Block Volumes</td>
                  <td className="px-4 py-3 text-neutral-400">Storage</td>
                  <td className="px-4 py-3 text-neutral-300">89</td>
                  <td className="px-4 py-3 text-neutral-400">All clusters</td>
                  <td className="px-4 py-3 text-right font-mono text-xs text-neutral-300">&euro;38.82</td>
                </tr>
                <tr className="border-b border-border transition-colors hover:bg-surface-200">
                  <td className="px-4 py-3 font-medium text-white">Load Balancers</td>
                  <td className="px-4 py-3 text-neutral-400">Network</td>
                  <td className="px-4 py-3 text-neutral-300">4</td>
                  <td className="px-4 py-3 text-neutral-400">All clusters</td>
                  <td className="px-4 py-3 text-right font-mono text-xs text-neutral-300">&euro;16.00</td>
                </tr>
                <tr className="transition-colors hover:bg-surface-200">
                  <td className="px-4 py-3 font-medium text-white">Managed DNS</td>
                  <td className="px-4 py-3 text-neutral-400">Network</td>
                  <td className="px-4 py-3 text-neutral-300">1</td>
                  <td className="px-4 py-3 text-neutral-400">--</td>
                  <td className="px-4 py-3 text-right font-mono text-xs text-neutral-300">&euro;0.00</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </Shell>
  );
}
