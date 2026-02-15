/**
 * Loading skeleton components matching the dark Mission Control theme.
 * All skeletons use the same surface colors and border styles as the real UI.
 */

interface SkeletonProps {
  className?: string;
}

/** Generic pulsing skeleton block */
export function Skeleton({ className = "" }: SkeletonProps) {
  return (
    <div
      className={`animate-pulse rounded bg-surface-300 ${className}`}
    />
  );
}

/** Skeleton for a StatCard */
export function StatCardSkeleton() {
  return (
    <div className="rounded-lg border border-border bg-surface-100 p-4">
      <Skeleton className="h-3 w-16" />
      <Skeleton className="mt-2 h-7 w-20" />
      <Skeleton className="mt-2 h-3 w-24" />
    </div>
  );
}

/** Row of 4 stat card skeletons */
export function StatCardRowSkeleton({ count = 4 }: { count?: number }) {
  return (
    <div className="grid grid-cols-4 gap-4">
      {Array.from({ length: count }).map((_, i) => (
        <StatCardSkeleton key={i} />
      ))}
    </div>
  );
}

/** Skeleton for a table with header and N rows */
export function TableSkeleton({
  columns = 6,
  rows = 3,
}: {
  columns?: number;
  rows?: number;
}) {
  return (
    <div className="overflow-hidden rounded-lg border border-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border bg-surface-100">
            {Array.from({ length: columns }).map((_, i) => (
              <th key={i} className="px-4 py-2.5 text-left">
                <Skeleton className="h-3 w-16" />
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {Array.from({ length: rows }).map((_, rowIdx) => (
            <tr
              key={rowIdx}
              className="border-b border-border last:border-0"
            >
              {Array.from({ length: columns }).map((_, colIdx) => (
                <td key={colIdx} className="px-4 py-3">
                  <Skeleton
                    className={`h-4 ${colIdx === 0 ? "w-28" : "w-16"}`}
                  />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

/** Skeleton for a section heading + table */
export function SectionSkeleton({
  columns = 6,
  rows = 3,
}: {
  columns?: number;
  rows?: number;
}) {
  return (
    <section>
      <Skeleton className="mb-3 h-4 w-32" />
      <TableSkeleton columns={columns} rows={rows} />
    </section>
  );
}

/** Skeleton for the updates / platform card */
export function CardSkeleton() {
  return (
    <div className="rounded-lg border border-border bg-surface-100 p-5">
      <div className="flex items-start justify-between">
        <div className="space-y-2">
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-3 w-56" />
        </div>
        <Skeleton className="h-9 w-28 rounded-lg" />
      </div>
      <div className="mt-4 space-y-2">
        <Skeleton className="h-4 w-24" />
        <Skeleton className="h-3 w-64" />
        <Skeleton className="h-3 w-56" />
        <Skeleton className="h-3 w-48" />
      </div>
    </div>
  );
}

/** Skeleton for a list of activity entries */
export function ActivityListSkeleton({ rows = 4 }: { rows?: number }) {
  return (
    <div className="space-y-0 rounded-lg border border-border bg-surface-100">
      {Array.from({ length: rows }).map((_, i) => (
        <div
          key={i}
          className="flex items-start gap-3 border-b border-border px-3 py-2.5 last:border-0"
        >
          <Skeleton className="mt-px h-3 w-10" />
          <div className="min-w-0 flex-1 space-y-1">
            <Skeleton className="h-4 w-3/4" />
          </div>
        </div>
      ))}
    </div>
  );
}

/** Skeleton for settings sections */
export function SettingsSectionSkeleton() {
  return (
    <section className="rounded-lg border border-border bg-surface-100 p-5">
      <Skeleton className="h-4 w-24" />
      <Skeleton className="mt-1 h-3 w-48" />
      <div className="mt-4 space-y-4">
        {Array.from({ length: 2 }).map((_, i) => (
          <div key={i} className="flex items-center justify-between">
            <div className="space-y-1">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-3 w-48" />
            </div>
            <Skeleton className="h-8 w-40 rounded-lg" />
          </div>
        ))}
      </div>
    </section>
  );
}

/** Skeleton for the cluster detail page */
export function ClusterDetailSkeleton() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-4 w-56" />
        </div>
        <Skeleton className="h-6 w-20 rounded-full" />
      </div>

      {/* Overview cards */}
      <StatCardRowSkeleton count={4} />

      {/* Resource usage */}
      <section>
        <Skeleton className="mb-3 h-4 w-32" />
        <div className="grid grid-cols-2 gap-4">
          {Array.from({ length: 2 }).map((_, i) => (
            <div
              key={i}
              className="rounded-lg border border-border bg-surface-100 p-4"
            >
              <div className="mb-2 flex items-center justify-between">
                <Skeleton className="h-4 w-12" />
                <Skeleton className="h-4 w-10" />
              </div>
              <Skeleton className="h-2.5 w-full rounded-full" />
            </div>
          ))}
        </div>
      </section>

      {/* Actions */}
      <section>
        <Skeleton className="mb-3 h-4 w-16" />
        <div className="flex gap-3">
          <Skeleton className="h-9 w-40 rounded-lg" />
          <Skeleton className="h-9 w-28 rounded-lg" />
        </div>
      </section>
    </div>
  );
}
