import { Shell } from "@/components/shell";
import { mockStorage } from "@/lib/mock-data";
import Link from "next/link";

function parseSizeGB(s: string): number {
  return parseFloat(s.replace(/[^0-9.]/g, ""));
}

export default function StoragePage() {
  const totalSize = mockStorage.reduce((sum, b) => sum + parseSizeGB(b.used), 0);
  const totalObjects = mockStorage.reduce((sum, b) => sum + b.objects, 0);

  /* Mock metadata for each bucket */
  const bucketMeta: Record<string, { access: "Private" | "Public"; versioning: "Enabled" | "Suspended"; created: string }> = {
    uploads: { access: "Private", versioning: "Enabled", created: "2025-09-14" },
    backups: { access: "Private", versioning: "Suspended", created: "2025-08-02" },
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Storage</h1>
            <p className="text-sm text-neutral-500">
              {mockStorage.length} buckets, {totalSize.toFixed(1)} GB total, {totalObjects.toLocaleString()} objects
            </p>
          </div>
          <button className="rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-3 py-1.5 text-sm transition-colors">
            + Create Bucket
          </button>
        </div>

        {/* Filter bar */}
        <div className="flex items-center gap-3">
          <div className="relative flex-1">
            <svg
              className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-500"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-4.35-4.35M11 19a8 8 0 100-16 8 8 0 000 16z" />
            </svg>
            <input
              type="text"
              placeholder="Filter buckets..."
              className="w-full rounded-lg border border-border bg-surface-100 py-1.5 pl-9 pr-3 text-sm text-white placeholder:text-neutral-500 focus:border-accent-500 focus:outline-none"
            />
          </div>
          <select className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 focus:border-accent-500 focus:outline-none">
            <option value="">All access</option>
            <option value="private">Private</option>
            <option value="public">Public</option>
          </select>
        </div>

        {/* Table */}
        <div className="overflow-hidden rounded-lg border border-border">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Bucket Name</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Objects</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Size</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Access</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Versioning</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Created</th>
                  <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Region</th>
                </tr>
              </thead>
              <tbody>
                {mockStorage.map((bucket) => {
                  const meta = bucketMeta[bucket.name] ?? { access: "Private" as const, versioning: "Suspended" as const, created: "---" };

                  return (
                    <tr
                      key={bucket.name}
                      className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors"
                    >
                      <td className="whitespace-nowrap px-4 py-3">
                        <Link
                          href={`/storage/${bucket.name}`}
                          className="font-medium text-white hover:text-accent-400 transition-colors"
                        >
                          {bucket.name}
                        </Link>
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 font-mono text-xs tabular-nums text-neutral-300">
                        {bucket.objects.toLocaleString()}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">
                        {bucket.used}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3">
                        {meta.access === "Private" ? (
                          <span className="inline-flex items-center rounded-full bg-neutral-500/10 px-2 py-0.5 text-xs font-medium text-neutral-400">
                            Private
                          </span>
                        ) : (
                          <span className="inline-flex items-center rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-400">
                            Public
                          </span>
                        )}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-xs">
                        {meta.versioning === "Enabled" ? (
                          <span className="text-emerald-400">Enabled</span>
                        ) : (
                          <span className="text-neutral-500">Suspended</span>
                        )}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                        {meta.created}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                        eu-central-1
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </Shell>
  );
}
