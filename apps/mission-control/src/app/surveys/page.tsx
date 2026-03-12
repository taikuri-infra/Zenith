"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { ErrorState } from "@/components/error-state";
import { StatCardRowSkeleton, Skeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { SurveyInsights } from "@/lib/api";
import { useApi } from "@/hooks/use-api";

const FIELD_LABELS: Record<string, string> = {
  use_case: "Use Case",
  role: "Role",
  team_size: "Team Size",
  current_provider: "Current Provider",
  monthly_spend: "Monthly Spend",
  biggest_pain: "Biggest Pain Point",
  expected_traffic: "Expected Traffic",
  timeline: "Timeline to Deploy",
  most_important: "Most Important Feature",
  stack: "Tech Stack",
  discovery: "Discovery Channel",
};

function topEntry(breakdown: Record<string, number> | undefined): string {
  if (!breakdown) return "-";
  const entries = Object.entries(breakdown);
  if (entries.length === 0) return "-";
  entries.sort((a, b) => b[1] - a[1]);
  return entries[0][0] || "-";
}

function BreakdownCard({
  field,
  data,
}: {
  field: string;
  data: Record<string, number>;
}) {
  const entries = Object.entries(data).sort((a, b) => b[1] - a[1]);
  const maxCount = entries.length > 0 ? entries[0][1] : 1;
  const total = entries.reduce((sum, [, count]) => sum + count, 0);

  return (
    <div className="rounded-lg border border-border bg-surface-100 p-4">
      <h3 className="mb-3 text-sm font-medium text-white">
        {FIELD_LABELS[field] || field}
      </h3>
      <div className="space-y-2">
        {entries.map(([label, count]) => {
          const pct = maxCount > 0 ? (count / maxCount) * 100 : 0;
          const pctOfTotal =
            total > 0 ? ((count / total) * 100).toFixed(0) : "0";
          return (
            <div key={label}>
              <div className="mb-1 flex items-center justify-between text-xs">
                <span className="truncate text-neutral-300">{label}</span>
                <span className="ml-2 flex-shrink-0 text-neutral-500">
                  {count} ({pctOfTotal}%)
                </span>
              </div>
              <div className="h-2 w-full overflow-hidden rounded-full bg-surface-300">
                <div
                  className="h-full rounded-full bg-accent-500 transition-all"
                  style={{ width: `${pct}%` }}
                />
              </div>
            </div>
          );
        })}
        {entries.length === 0 && (
          <p className="text-xs text-neutral-500">No data</p>
        )}
      </div>
    </div>
  );
}

function BreakdownSkeletons() {
  return (
    <div className="grid grid-cols-2 gap-4">
      {Array.from({ length: 6 }).map((_, i) => (
        <div
          key={i}
          className="rounded-lg border border-border bg-surface-100 p-4"
        >
          <Skeleton className="mb-3 h-4 w-32" />
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, j) => (
              <div key={j}>
                <div className="mb-1 flex justify-between">
                  <Skeleton className="h-3 w-24" />
                  <Skeleton className="h-3 w-12" />
                </div>
                <Skeleton className="h-2 w-full rounded-full" />
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

export default function SurveysPage() {
  const apiClient = getApi();
  const { data, loading, error, refetch } = useApi<SurveyInsights>(
    () => apiClient.surveys.insights()
  );

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Survey Insights</h1>

        {/* KPI Row */}
        {loading ? (
          <StatCardRowSkeleton />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : data ? (
          <>
            <div className="grid grid-cols-4 gap-4">
              <StatCard
                label="Total Responses"
                value={data.total_responses}
                sub="onboarding surveys"
              />
              <StatCard
                label="Top Use Case"
                value={topEntry(data.breakdowns?.use_case)}
                sub="most common"
              />
              <StatCard
                label="Top Provider"
                value={topEntry(data.breakdowns?.current_provider)}
                sub="current provider"
              />
              <StatCard
                label="Top Pain Point"
                value={topEntry(data.breakdowns?.biggest_pain)}
                sub="biggest pain"
              />
            </div>

            {/* Breakdown Charts */}
            <section>
              <h2 className="mb-3 text-sm font-medium text-white">
                Response Breakdowns
              </h2>
              <div className="grid grid-cols-2 gap-4">
                {Object.entries(data.breakdowns || {}).map(([field, dist]) => (
                  <BreakdownCard key={field} field={field} data={dist} />
                ))}
              </div>
            </section>

            {/* Recent Responses Table */}
            {data.responses && data.responses.length > 0 && (
              <section>
                <h2 className="mb-3 text-sm font-medium text-white">
                  Recent Responses
                </h2>
                <div className="overflow-hidden rounded-lg border border-border">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border bg-surface-100">
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                          User
                        </th>
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                          Date
                        </th>
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                          Use Case
                        </th>
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                          Role
                        </th>
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                          Provider
                        </th>
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                          Spend
                        </th>
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                          Pain Point
                        </th>
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">
                          Timeline
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.responses.map((r, idx) => (
                        <tr
                          key={`${r.user_id}-${idx}`}
                          className="border-b border-border last:border-0"
                        >
                          <td
                            className="px-4 py-3 font-mono text-xs text-neutral-300"
                            title={r.user_id}
                          >
                            {r.user_id.length > 8
                              ? `${r.user_id.slice(0, 8)}...`
                              : r.user_id}
                          </td>
                          <td className="px-4 py-3 text-neutral-300">
                            {new Date(r.created_at).toLocaleDateString()}
                          </td>
                          <td className="px-4 py-3 text-white">
                            {r.use_case}
                          </td>
                          <td className="px-4 py-3 text-neutral-300">
                            {r.role}
                          </td>
                          <td className="px-4 py-3 text-neutral-300">
                            {r.current_provider}
                          </td>
                          <td className="px-4 py-3 text-neutral-300">
                            {r.monthly_spend}
                          </td>
                          <td className="px-4 py-3 text-neutral-300">
                            {r.biggest_pain}
                          </td>
                          <td className="px-4 py-3 text-neutral-300">
                            {r.timeline}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </section>
            )}
          </>
        ) : null}

        {/* Loading skeletons for breakdowns and table */}
        {loading && (
          <>
            <BreakdownSkeletons />
          </>
        )}
      </div>
    </Shell>
  );
}
