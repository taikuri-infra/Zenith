"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import { demoApi } from "@/lib/demo-api";
import type { Route, Certificate, DnsRecord } from "@/lib/api";
import { useApiWithFallback } from "@/hooks/use-api";
import { Globe, Network, Shield } from "lucide-react";

export default function NetworkingPage() {
  const apiClient = getApi();
  const routes = useApiWithFallback<Route[]>(
    () => apiClient.networking.routes(),
    () => demoApi.networking.routes()
  );
  const certs = useApiWithFallback<Certificate[]>(
    () => apiClient.networking.certificates(),
    () => demoApi.networking.certificates()
  );
  const dns = useApiWithFallback<DnsRecord[]>(
    () => apiClient.networking.dnsRecords(),
    () => demoApi.networking.dnsRecords()
  );

  const anyDemo = routes.isDemo || certs.isDemo || dns.isDemo;

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Networking</h1>
          {anyDemo && (
            <p className="mt-1 text-xs text-amber-400/70">Showing sample data where live data unavailable</p>
          )}
        </div>

        {/* Routes */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">Ingress Routes</h2>
          {routes.loading ? (
            <TableSkeleton columns={5} rows={4} />
          ) : routes.error ? (
            <ErrorState error={routes.error} onRetry={routes.refetch} />
          ) : !routes.data || routes.data.length === 0 ? (
            <EmptyState title="No routes" description="No ingress routes configured." icon={Network} />
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Host</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Service</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">TLS</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Namespace</th>
                  </tr>
                </thead>
                <tbody>
                  {routes.data.map((route) => (
                    <tr key={`${route.namespace}-${route.name}`} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                      <td className="px-4 py-3 font-medium text-white">{route.name}</td>
                      <td className="px-4 py-3 font-mono text-xs text-accent-400">{route.host}</td>
                      <td className="px-4 py-3 text-neutral-300">{route.service}:{route.port}</td>
                      <td className="px-4 py-3">
                        {route.tls ? (
                          <span className="text-emerald-400 text-xs">Enabled</span>
                        ) : (
                          <span className="text-neutral-500 text-xs">None</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-neutral-400">{route.namespace}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        {/* Certificates */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">TLS Certificates</h2>
          {certs.loading ? (
            <TableSkeleton columns={5} rows={3} />
          ) : certs.error ? (
            <ErrorState error={certs.error} onRetry={certs.refetch} />
          ) : !certs.data || certs.data.length === 0 ? (
            <EmptyState title="No certificates" description="No TLS certificates found." icon={Shield} />
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Domains</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Issuer</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Expires</th>
                  </tr>
                </thead>
                <tbody>
                  {certs.data.map((cert) => (
                    <tr key={cert.name} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                      <td className="px-4 py-3 font-medium text-white">{cert.name}</td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap gap-1">
                          {cert.domains.map((d) => (
                            <span key={d} className="rounded bg-surface-300 px-1.5 py-0.5 font-mono text-[10px] text-neutral-300">{d}</span>
                          ))}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-neutral-400 text-xs">{cert.issuer}</td>
                      <td className="px-4 py-3">
                        <StatusBadge status={cert.ready ? "healthy" : "error"} label={cert.ready ? "Ready" : "Not Ready"} />
                      </td>
                      <td className="px-4 py-3 text-xs text-neutral-500">{cert.expiresAt}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        {/* DNS Records */}
        <section>
          <h2 className="mb-3 text-sm font-medium text-white">DNS Records</h2>
          {dns.loading ? (
            <TableSkeleton columns={5} rows={4} />
          ) : dns.error ? (
            <ErrorState error={dns.error} onRetry={dns.refetch} />
          ) : !dns.data || dns.data.length === 0 ? (
            <EmptyState title="No DNS records" description="No external DNS records configured." icon={Globe} />
          ) : (
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Type</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Value</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">TTL</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Managed By</th>
                  </tr>
                </thead>
                <tbody>
                  {dns.data.map((record, i) => (
                    <tr key={`${record.name}-${i}`} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                      <td className="px-4 py-3 font-mono text-xs text-white">{record.name}</td>
                      <td className="px-4 py-3">
                        <span className="rounded bg-surface-300 px-1.5 py-0.5 text-xs text-neutral-300">{record.type}</span>
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-300">{record.value}</td>
                      <td className="px-4 py-3 text-neutral-500 text-xs">{record.ttl}s</td>
                      <td className="px-4 py-3 text-neutral-500 text-xs">{record.managedBy}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </Shell>
  );
}
