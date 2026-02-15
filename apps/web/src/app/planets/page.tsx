"use client";

import { Shell } from "@/components/shell";
import { EmptyState } from "@/components/empty-state";

/**
 * Planets page - compute nodes.
 *
 * There is no planets API endpoint yet so this page shows an empty state
 * indicating the feature is coming soon. When the API is implemented,
 * this page will use useApi + a planets.list() call.
 */
export default function PlanetsPage() {
  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Planets</h1>
            <p className="text-sm text-neutral-500">
              Compute nodes powering your project
            </p>
          </div>
          <button className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
            + Add Planet
          </button>
        </div>

        <EmptyState
          title="No planets available"
          description="Planets (compute nodes) will appear here once the infrastructure API is connected."
        />
      </div>
    </Shell>
  );
}
