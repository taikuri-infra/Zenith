/**
 * Reusable loading skeleton components for the Zenith web platform.
 * These match the dark theme with emerald accents used throughout the app.
 */

function Bone({ className = "" }: { className?: string }) {
  return (
    <div
      className={`animate-pulse rounded bg-surface-300 ${className}`}
    />
  );
}

/** A skeleton row for table layouts. */
export function TableRowSkeleton({ cols = 6 }: { cols?: number }) {
  return (
    <tr className="border-b border-border last:border-0">
      {Array.from({ length: cols }).map((_, i) => (
        <td key={i} className="px-4 py-3">
          <Bone className="h-4 w-full max-w-[120px]" />
        </td>
      ))}
    </tr>
  );
}

/** Skeleton for a full table with header and rows. */
export function TableSkeleton({
  cols = 6,
  rows = 4,
}: {
  cols?: number;
  rows?: number;
}) {
  return (
    <div className="overflow-hidden rounded-lg border border-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border bg-surface-100">
            {Array.from({ length: cols }).map((_, i) => (
              <th key={i} className="px-4 py-2.5">
                <Bone className="h-3 w-20" />
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {Array.from({ length: rows }).map((_, i) => (
            <TableRowSkeleton key={i} cols={cols} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

/** Skeleton for a stat card. */
export function StatCardSkeleton() {
  return (
    <div className="rounded-lg border border-border bg-surface-100 p-4">
      <Bone className="mb-2 h-3 w-16" />
      <Bone className="mb-1 h-7 w-20" />
      <Bone className="h-3 w-24" />
    </div>
  );
}

/** Skeleton for the dashboard / overview page. */
export function DashboardSkeleton() {
  return (
    <div className="space-y-6">
      <div>
        <Bone className="mb-2 h-5 w-32" />
        <Bone className="h-3 w-24" />
      </div>
      <div className="grid grid-cols-4 gap-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <StatCardSkeleton key={i} />
        ))}
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <Bone className="mb-2 h-3 w-20" />
          <Bone className="h-2.5 w-full" />
        </div>
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <Bone className="mb-2 h-3 w-20" />
          <Bone className="h-2.5 w-full" />
        </div>
      </div>
      <TableSkeleton cols={6} rows={5} />
    </div>
  );
}

/** Skeleton for a list of cards (apps, planets, etc.). */
export function CardListSkeleton({ count = 3 }: { count?: number }) {
  return (
    <div className="space-y-2">
      {Array.from({ length: count }).map((_, i) => (
        <div
          key={i}
          className="rounded-lg border border-border bg-surface-100 p-4"
        >
          <div className="flex items-center justify-between">
            <div>
              <Bone className="mb-2 h-4 w-28" />
              <Bone className="h-3 w-20" />
            </div>
            <Bone className="h-4 w-16" />
          </div>
        </div>
      ))}
    </div>
  );
}

/** Skeleton for a page header with title and subtitle. */
export function PageHeaderSkeleton() {
  return (
    <div className="flex items-center justify-between">
      <div>
        <Bone className="mb-2 h-5 w-32" />
        <Bone className="h-3 w-48" />
      </div>
      <Bone className="h-8 w-28 rounded-lg" />
    </div>
  );
}

/** Full page loading skeleton with header, filter bar, and table. */
export function PageWithTableSkeleton({
  cols = 6,
  rows = 4,
}: {
  cols?: number;
  rows?: number;
}) {
  return (
    <div className="space-y-6">
      <PageHeaderSkeleton />
      <div className="flex items-center gap-3">
        <Bone className="h-8 flex-1 rounded-lg" />
        <Bone className="h-8 w-32 rounded-lg" />
      </div>
      <TableSkeleton cols={cols} rows={rows} />
    </div>
  );
}

/** App detail page skeleton. */
export function AppDetailSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2">
            <Bone className="h-5 w-32" />
            <Bone className="h-5 w-16 rounded-full" />
          </div>
          <Bone className="mt-1 h-3 w-40" />
        </div>
        <div className="flex items-center gap-2">
          <Bone className="h-8 w-24 rounded-lg" />
          <Bone className="h-8 w-20 rounded-lg" />
        </div>
      </div>
      <div className="grid grid-cols-4 gap-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <StatCardSkeleton key={i} />
        ))}
      </div>
      <Bone className="h-10 w-full rounded-lg" />
      <div className="grid grid-cols-2 gap-6">
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <Bone className="mb-3 h-4 w-16" />
          <div className="space-y-2">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-center justify-between">
                <Bone className="h-3 w-20" />
                <Bone className="h-3 w-32" />
              </div>
            ))}
          </div>
        </div>
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <Bone className="mb-3 h-4 w-20" />
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex items-center justify-between">
                <Bone className="h-3 w-20" />
                <Bone className="h-3 w-24" />
              </div>
            ))}
          </div>
        </div>
      </div>
      <TableSkeleton cols={5} rows={4} />
    </div>
  );
}

/** Database detail page skeleton. */
export function DatabaseDetailSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Bone className="h-10 w-10 rounded-lg" />
        <div>
          <div className="flex items-center gap-2">
            <Bone className="h-5 w-28" />
            <Bone className="h-5 w-20 rounded-md" />
            <Bone className="h-5 w-16 rounded-full" />
          </div>
          <Bone className="mt-1 h-3 w-24" />
        </div>
      </div>
      <div className="rounded-lg border border-border bg-surface-100 p-4">
        <Bone className="mb-2 h-4 w-32" />
        <Bone className="h-10 w-full rounded-lg" />
      </div>
      <div className="grid grid-cols-4 gap-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <StatCardSkeleton key={i} />
        ))}
      </div>
      <TableSkeleton cols={5} rows={4} />
    </div>
  );
}
