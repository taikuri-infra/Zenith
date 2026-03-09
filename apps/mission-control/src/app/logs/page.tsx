"use client";

import { useState } from "react";
import { Shell } from "@/components/shell";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { Skeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { LogQueryResult } from "@/lib/api";
import { useApi, useMutation } from "@/hooks/use-api";
import { FileText, Search, Play } from "lucide-react";

const defaultQuery = '{namespace="zenith-platform"} |= ""';

const presetQueries = [
  { label: "All errors", query: '{namespace="zenith-platform"} |= "error"' },
  { label: "API logs", query: '{app="zenith-api"} | json' },
  { label: "Auth failures", query: '{app="zenith-api"} |= "authentication failed"' },
  { label: "Slow queries", query: '{app="zenith-api"} |= "slow query"' },
  { label: "Build logs", query: '{namespace="zenith-builds"}' },
];

export default function LogsPage() {
  const apiClient = getApi();
  const [query, setQuery] = useState(defaultQuery);
  const [limit, setLimit] = useState(100);
  const [results, setResults] = useState<LogQueryResult | null>(null);
  const [queryError, setQueryError] = useState<Error | null>(null);

  const mutation = useMutation<{ query: string; limit: number }, LogQueryResult>(
    (input) => apiClient.logs.query(input.query, input.limit)
  );

  const handleQuery = async () => {
    setQueryError(null);
    try {
      const result = await mutation.execute({ query, limit });
      setResults(result);
    } catch (err) {
      setQueryError(err instanceof Error ? err : new Error(String(err)));
    }
  };

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Logs</h1>

        {/* Query Input */}
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <label className="mb-2 block text-xs font-medium text-neutral-400">
            LogQL Query
          </label>
          <textarea
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            rows={3}
            className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2 font-mono text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
            placeholder='Enter LogQL query, e.g. {namespace="zenith-platform"}'
          />

          <div className="mt-3 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <label className="text-xs text-neutral-500">Limit</label>
              <select
                value={limit}
                onChange={(e) => setLimit(Number(e.target.value))}
                className="rounded border border-border bg-surface-200 px-2 py-1 text-xs text-white focus:border-accent-500 focus:outline-none"
              >
                <option value={50}>50</option>
                <option value={100}>100</option>
                <option value={250}>250</option>
                <option value={500}>500</option>
                <option value={1000}>1000</option>
              </select>
            </div>
            <button
              onClick={handleQuery}
              disabled={mutation.loading || !query.trim()}
              className="flex items-center gap-1.5 rounded-lg bg-accent-600 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {mutation.loading ? (
                <>
                  <div className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                  Querying...
                </>
              ) : (
                <>
                  <Play className="h-3.5 w-3.5" />
                  Run Query
                </>
              )}
            </button>
          </div>

          {/* Preset Queries */}
          <div className="mt-3 flex flex-wrap gap-2">
            {presetQueries.map((preset) => (
              <button
                key={preset.label}
                onClick={() => setQuery(preset.query)}
                className="rounded-md border border-border bg-surface-200 px-2.5 py-1 text-[11px] text-neutral-400 hover:bg-surface-300 hover:text-white transition-colors"
              >
                {preset.label}
              </button>
            ))}
          </div>
        </div>

        {/* Error */}
        {queryError && (
          <ErrorState error={queryError} onRetry={handleQuery} />
        )}

        {/* Results */}
        {results ? (
          <section>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-medium text-white">
                Results <span className="text-neutral-500">({results.lines.length} lines)</span>
              </h2>
              <span className="text-xs text-neutral-500">
                Executed in {results.executionTimeMs}ms
              </span>
            </div>
            {results.lines.length === 0 ? (
              <EmptyState
                title="No results"
                description="The query returned no log lines."
                icon={Search}
              />
            ) : (
              <div className="overflow-hidden rounded-lg border border-border bg-surface-100">
                <div className="max-h-[600px] overflow-y-auto">
                  {results.lines.map((line, i) => (
                    <div
                      key={i}
                      className="flex border-b border-border px-3 py-1.5 last:border-0 hover:bg-surface-200 transition-colors"
                    >
                      <span className="mr-3 flex-shrink-0 font-mono text-[10px] text-neutral-600 leading-5 select-none">
                        {line.timestamp}
                      </span>
                      <span className="mr-3 flex-shrink-0 rounded bg-surface-300 px-1 py-0.5 text-[10px] text-neutral-400">
                        {line.labels}
                      </span>
                      <span className="font-mono text-xs text-neutral-300 whitespace-pre-wrap break-all leading-5">
                        {line.message}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </section>
        ) : !mutation.loading && !queryError ? (
          <EmptyState
            title="Run a query"
            description="Enter a LogQL query above and click Run Query to search logs."
            icon={FileText}
          />
        ) : null}
      </div>
    </Shell>
  );
}
