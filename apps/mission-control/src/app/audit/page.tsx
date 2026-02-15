import { Shell } from "@/components/shell";
import { mockAuditLog } from "@/lib/mock-data";

export default function AuditPage() {
  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Audit Log</h1>

        {/* Filter dropdowns (non-functional placeholders) */}
        <div className="flex gap-3">
          <select className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-300 outline-none">
            <option>All Actors</option>
            <option>admin</option>
            <option>system</option>
            <option>CAPI</option>
          </select>
          <select className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-300 outline-none">
            <option>All Clusters</option>
            <option>zenith-shared</option>
            <option>pro-startup-a</option>
            <option>pro-enterprise</option>
          </select>
          <select className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-300 outline-none">
            <option>Today</option>
            <option>Last 7 days</option>
            <option>Last 30 days</option>
            <option>All time</option>
          </select>
        </div>

        {/* Audit log table */}
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-100">
                <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Time</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Actor</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Action</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Cluster</th>
              </tr>
            </thead>
            <tbody>
              {mockAuditLog.map((entry, i) => (
                <tr
                  key={i}
                  className="border-b border-border last:border-0 transition-colors hover:bg-surface-200"
                >
                  <td className="px-4 py-3 font-mono text-xs text-neutral-500">{entry.time}</td>
                  <td className="px-4 py-3 font-medium text-white">{entry.actor}</td>
                  <td className="px-4 py-3 text-neutral-300">{entry.action}</td>
                  <td className="px-4 py-3 text-neutral-400">{entry.cluster || "--"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </Shell>
  );
}
