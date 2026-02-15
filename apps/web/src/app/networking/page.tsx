"use client";

import { Shell } from "@/components/shell";
import { EmptyState } from "@/components/empty-state";

/**
 * Networking page - domains, firewalls, and load balancers.
 *
 * There is no networking / domains API endpoint yet so this page shows
 * placeholder content. When the API is implemented, this page will use
 * useApi + the relevant API calls.
 */
export default function NetworkingPage() {
  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Networking</h1>
          <p className="text-sm text-neutral-500">
            Domains, firewalls, and load balancers
          </p>
        </div>

        {/* Domains */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Domains</h2>
            <button className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
              + Add Domain
            </button>
          </div>
          <EmptyState
            title="No domains configured"
            description="Custom domains will appear here once the networking API is connected."
          />
        </section>

        {/* Firewalls */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Firewalls</h2>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-emerald-500/10">
                  <svg
                    className="h-4.5 w-4.5 text-emerald-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
                    />
                  </svg>
                </div>
                <div>
                  <p className="text-sm font-medium text-white">
                    Default firewall active
                  </p>
                  <p className="text-xs text-neutral-500">
                    Ports 80, 443 open. All other inbound traffic blocked.
                  </p>
                </div>
              </div>
              <button className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
                Configure
              </button>
            </div>
          </div>
        </section>

        {/* Load Balancers */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Load Balancers</h2>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-accent-500/10">
                  <svg
                    className="h-4.5 w-4.5 text-accent-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                    />
                  </svg>
                </div>
                <div>
                  <p className="text-sm font-medium text-white">
                    Load balancer status
                  </p>
                  <p className="text-xs text-neutral-500">
                    Load balancer information will be available once connected to
                    the API.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </Shell>
  );
}
