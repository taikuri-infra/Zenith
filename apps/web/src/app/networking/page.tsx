import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { mockDomains } from "@/lib/mock-data";

export default function NetworkingPage() {
  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Networking</h1>
          <p className="text-sm text-neutral-500">Domains, firewalls, and load balancers</p>
        </div>

        {/* Domains */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Domains</h2>
            <button className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
              + Add Domain
            </button>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Domain</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Linked App</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">SSL</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                </tr>
              </thead>
              <tbody>
                {mockDomains.map((d) => (
                  <tr key={d.domain} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="px-4 py-3 font-medium text-white">{d.domain}</td>
                    <td className="px-4 py-3 text-neutral-300">{d.app}</td>
                    <td className="px-4 py-3">
                      {d.ssl ? (
                        <span className="inline-flex items-center gap-1.5 text-xs text-emerald-400">
                          <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                          </svg>
                          Active
                        </span>
                      ) : (
                        <span className="text-xs text-neutral-500">None</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={d.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
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
                  <svg className="h-4.5 w-4.5 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                  </svg>
                </div>
                <div>
                  <p className="text-sm font-medium text-white">Default firewall active</p>
                  <p className="text-xs text-neutral-500">Ports 80, 443 open. All other inbound traffic blocked.</p>
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
                  <svg className="h-4.5 w-4.5 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                </div>
                <div>
                  <p className="text-sm font-medium text-white">1 load balancer active</p>
                  <p className="text-xs text-neutral-500">Distributing traffic across 5 planets</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-400">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                  Healthy
                </span>
              </div>
            </div>
          </div>
        </section>
      </div>
    </Shell>
  );
}
