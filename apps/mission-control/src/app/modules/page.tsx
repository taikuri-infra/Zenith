import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { mockModules } from "@/lib/mock-data";

const updatesAvailable = mockModules.filter(
  (m) => m.status === "update_available"
).length;

export default function ModulesPage() {
  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Modules</h1>
            <p className="mt-1 text-sm text-neutral-500">
              {mockModules.length} installed &middot; {updatesAvailable} update{updatesAvailable !== 1 ? "s" : ""} available
            </p>
          </div>
          {updatesAvailable > 0 && (
            <button className="rounded-lg bg-accent-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500">
              Update All
            </button>
          )}
        </div>

        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-100">
                <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Module</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Description</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Installed</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Latest</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
              </tr>
            </thead>
            <tbody>
              {mockModules.map((mod) => (
                <tr
                  key={mod.name}
                  className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                >
                  <td className="px-4 py-3 font-medium text-white">{mod.name}</td>
                  <td className="px-4 py-3 text-neutral-400">{mod.description}</td>
                  <td className="px-4 py-3 font-mono text-xs text-neutral-400">{mod.installed}</td>
                  <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                    {mod.latest}
                    {mod.status === "update_available" && (
                      <span className="ml-1.5 text-amber-400">&#9888;</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={mod.status} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </Shell>
  );
}
