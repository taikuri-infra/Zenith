"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { StatCardRowSkeleton, TableSkeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { KyvernoPolicy, WafStats } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import {
  ShieldAlert,
  ExternalLink,
  LineChart,
  ScrollText,
  Boxes,
  Container,
  KeyRound,
  Globe,
  Shield,
  Activity,
} from "lucide-react";

const adminServices = [
  {
    name: "Grafana",
    description: "Metrics dashboards and visualization",
    url: "https://grafana.stage.freezenith.com",
    icon: LineChart,
    color: "text-orange-400 bg-orange-500/10 border-orange-500/20",
  },
  {
    name: "ArgoCD",
    description: "GitOps continuous delivery",
    url: "https://argocd.stage.freezenith.com",
    icon: Boxes,
    color: "text-sky-400 bg-sky-500/10 border-sky-500/20",
  },
  {
    name: "Harbor Registry",
    description: "Container image registry",
    url: "https://registry.stage.freezenith.com",
    icon: Container,
    color: "text-teal-400 bg-teal-500/10 border-teal-500/20",
  },
  {
    name: "Keycloak",
    description: "Identity and access management",
    url: "https://keycloak.stage.freezenith.com",
    icon: KeyRound,
    color: "text-violet-400 bg-violet-500/10 border-violet-500/20",
  },
  {
    name: "Prometheus",
    description: "Metrics collection and alerting",
    url: "https://prometheus.stage.freezenith.com",
    icon: Activity,
    color: "text-red-400 bg-red-500/10 border-red-500/20",
  },
  {
    name: "APISIX Dashboard",
    description: "API gateway management",
    url: "https://apisix-dashboard.stage.freezenith.com",
    icon: Globe,
    color: "text-rose-400 bg-rose-500/10 border-rose-500/20",
  },
  {
    name: "Temporal",
    description: "Workflow orchestration",
    url: "https://temporal.stage.freezenith.com",
    icon: ScrollText,
    color: "text-emerald-400 bg-emerald-500/10 border-emerald-500/20",
  },
  {
    name: "Kyverno",
    description: "Kubernetes policy engine",
    url: "#",
    icon: Shield,
    color: "text-amber-400 bg-amber-500/10 border-amber-500/20",
  },
];

function actionBadge(action: string) {
  switch (action) {
    case "enforce":
      return <span className="rounded-full bg-red-500/15 px-2 py-0.5 text-xs font-medium text-red-400">Enforce</span>;
    case "audit":
      return <span className="rounded-full bg-amber-500/15 px-2 py-0.5 text-xs font-medium text-amber-400">Audit</span>;
    default:
      return <span className="rounded-full bg-neutral-500/10 px-2 py-0.5 text-xs font-medium text-neutral-400">{action}</span>;
  }
}

export default function WafPage() {
  const apiClient = getApi();
  const stats = useApi<WafStats>(() => apiClient.security.wafStats());
  const { data: policies, loading, error, refetch } = useApi<KyvernoPolicy[]>(
    () => apiClient.security.policies()
  );

  return (
    <Shell>
      <div className="space-y-8">
        <div>
          <h1 className="text-lg font-semibold text-white">WAF & Admin Services</h1>
          <p className="mt-1 text-sm text-neutral-500">
            Security policies and quick access to all admin service portals
          </p>
        </div>

        {/* Admin Service Portal */}
        <section>
          <h2 className="mb-4 text-sm font-medium text-white">Admin Service Portal</h2>
          <div className="grid grid-cols-4 gap-4">
            {adminServices.map((svc) => (
              <a
                key={svc.name}
                href={svc.url}
                target="_blank"
                rel="noopener noreferrer"
                className={`group rounded-xl border p-4 transition-all hover:scale-[1.02] hover:shadow-lg ${svc.color}`}
              >
                <div className="flex items-center justify-between mb-3">
                  <svc.icon className="h-5 w-5" />
                  <ExternalLink className="h-3.5 w-3.5 opacity-0 group-hover:opacity-100 transition-opacity text-neutral-400" />
                </div>
                <h3 className="text-sm font-semibold text-white">{svc.name}</h3>
                <p className="mt-1 text-xs text-neutral-500">{svc.description}</p>
              </a>
            ))}
          </div>
        </section>

        {/* Policy Stats */}
        <section>
          <h2 className="mb-4 text-sm font-medium text-white">Security Policies</h2>
          {stats.loading ? (
            <StatCardRowSkeleton />
          ) : stats.data ? (
            <div className="grid grid-cols-4 gap-4">
              <StatCard label="Total Policies" value={stats.data.totalPolicies} sub="Kyverno policies" />
              <StatCard label="Enforcing" value={stats.data.enforcing} sub="blocking violations" />
              <StatCard label="Auditing" value={stats.data.auditing} sub="logging only" />
              <StatCard label="Total Violations" value={stats.data.totalViolations} sub="all time" alert={stats.data.totalViolations > 0} />
            </div>
          ) : null}
        </section>

        {/* Policies Table */}
        {loading ? (
          <TableSkeleton columns={6} rows={5} />
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : !policies || policies.length === 0 ? (
          <div className="rounded-xl border border-border bg-surface-100 py-8 text-center">
            <ShieldAlert className="mx-auto h-8 w-8 text-neutral-600 mb-2" />
            <p className="text-sm text-neutral-500">No Kyverno policies configured</p>
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Kind</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Action</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Violations</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Updated</th>
                </tr>
              </thead>
              <tbody>
                {policies.map((policy) => (
                  <tr key={policy.name} className="border-b border-border last:border-0 transition-colors hover:bg-surface-200">
                    <td className="px-4 py-3 font-medium text-white">{policy.name}</td>
                    <td className="px-4 py-3">
                      <span className="rounded bg-surface-300 px-1.5 py-0.5 text-xs text-neutral-300">
                        {policy.kind}
                      </span>
                    </td>
                    <td className="px-4 py-3">{actionBadge(policy.action)}</td>
                    <td className="px-4 py-3">
                      <StatusBadge status={policy.ready ? "healthy" : "error"} label={policy.ready ? "Ready" : "Error"} />
                    </td>
                    <td className="px-4 py-3">
                      <span className={`text-sm ${policy.violations > 0 ? "text-red-400 font-medium" : "text-neutral-400"}`}>
                        {policy.violations}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{policy.updatedAt}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Shell>
  );
}
