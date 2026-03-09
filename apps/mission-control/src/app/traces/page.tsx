"use client";

import { useState } from "react";
import { Shell } from "@/components/shell";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { Trace } from "@/lib/api";
import { useMutation } from "@/hooks/use-api";
import { Search, Activity } from "lucide-react";

export default function TracesPage() {
  const apiClient = getApi();
  const [serviceName, setServiceName] = useState("");
  const [minDuration, setMinDuration] = useState("");
  const [limit, setLimit] = useState(20);
  const [traces, setTraces] = useState<Trace[] | null>(null);
  const [searchError, setSearchError] = useState<Error | null>(null);

  const mutation = useMutation<{ service: string; minDuration?: string; limit: number }, Trace[]>(
    (input) => apiClient.traces.search(input.service, input.minDuration, input.limit)
  );

  const handleSearch = async () => {
    setSearchError(null);
    try {
      const result = await mutation.execute({
        service: serviceName,
        minDuration: minDuration || undefined,
        limit,
      });
      setTraces(result);
    } catch (err) {
      setSearchError(err instanceof Error ? err : new Error(String(err)));
    }
  };

  function durationColor(ms: number) {
    if (ms < 100) return "text-emerald-400";
    if (ms < 500) return "text-amber-400";
    return "text-red-400";
  }

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Traces</h1>

        {/* Search Form */}
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <div className="grid grid-cols-4 gap-4">
            <div className="col-span-2">
              <label className="mb-1 block text-xs font-medium text-neutral-400">Service Name</label>
              <input
                type="text"
                value={serviceName}
                onChange={(e) => setServiceName(e.target.value)}
                placeholder="e.g. zenith-api"
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Min Duration</label>
              <input
                type="text"
                value={minDuration}
                onChange={(e) => setMinDuration(e.target.value)}
                placeholder="e.g. 100ms"
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Limit</label>
              <select
                value={limit}
                onChange={(e) => setLimit(Number(e.target.value))}
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
              >
                <option value={10}>10</option>
                <option value={20}>20</option>
                <option value={50}>50</option>
                <option value={100}>100</option>
              </select>
            </div>
          </div>
          <div className="mt-3 flex justify-end">
            <button
              onClick={handleSearch}
              disabled={mutation.loading}
              className="flex items-center gap-1.5 rounded-lg bg-accent-600 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {mutation.loading ? (
                <>
                  <div className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                  Searching...
                </>
              ) : (
                <>
                  <Search className="h-3.5 w-3.5" />
                  Search Traces
                </>
              )}
            </button>
          </div>
        </div>

        {/* Error */}
        {searchError && (
          <ErrorState error={searchError} onRetry={handleSearch} />
        )}

        {/* Results */}
        {mutation.loading ? (
          <TableSkeleton columns={5} rows={5} />
        ) : traces ? (
          traces.length === 0 ? (
            <EmptyState
              title="No traces found"
              description="Try adjusting your search criteria."
              icon={Search}
            />
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Trace ID</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Service</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Operation</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Duration</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Spans</th>
                  </tr>
                </thead>
                <tbody>
                  {traces.map((trace) => (
                    <tr key={trace.traceId} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                      <td className="px-4 py-3">
                        <span className="font-mono text-xs text-accent-400">{trace.traceId.slice(0, 16)}...</span>
                      </td>
                      <td className="px-4 py-3 text-white">{trace.service}</td>
                      <td className="px-4 py-3 text-neutral-300">{trace.operationName}</td>
                      <td className="px-4 py-3">
                        <span className={`font-mono text-xs ${durationColor(trace.durationMs)}`}>
                          {trace.durationMs}ms
                        </span>
                      </td>
                      <td className="px-4 py-3 text-neutral-400">{trace.spanCount}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        ) : !searchError ? (
          <EmptyState
            title="Search for traces"
            description="Enter a service name and click Search to find distributed traces."
            icon={Activity}
          />
        ) : null}
      </div>
    </Shell>
  );
}
