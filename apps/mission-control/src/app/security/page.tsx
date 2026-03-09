"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { ErrorState } from "@/components/error-state";
import { StatCardRowSkeleton, Skeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { SecurityOverview } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import Link from "next/link";
import { ArrowUpRight, Shield, ShieldAlert, Key, Eye, Bug } from "lucide-react";

function scoreColor(score: number): string {
  if (score >= 90) return "text-emerald-400";
  if (score >= 70) return "text-amber-400";
  return "text-red-400";
}

function scoreRingColor(score: number): string {
  if (score >= 90) return "stroke-emerald-400";
  if (score >= 70) return "stroke-amber-400";
  return "stroke-red-400";
}

export default function SecurityPage() {
  const apiClient = getApi();
  const { data, loading, error, refetch } = useApi<SecurityOverview>(
    () => apiClient.security.overview()
  );

  return (
    <Shell>
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-white">Security</h1>

        {loading ? (
          <>
            <div className="flex items-center justify-center py-12">
              <Skeleton className="h-40 w-40 rounded-full" />
            </div>
            <StatCardRowSkeleton />
          </>
        ) : error ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : data ? (
          <>
            {/* Security Score */}
            <div className="flex items-center justify-center py-8">
              <div className="relative flex h-44 w-44 items-center justify-center">
                <svg className="absolute inset-0 -rotate-90" viewBox="0 0 160 160">
                  <circle
                    cx="80"
                    cy="80"
                    r="70"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="8"
                    className="text-surface-300"
                  />
                  <circle
                    cx="80"
                    cy="80"
                    r="70"
                    fill="none"
                    strokeWidth="8"
                    strokeLinecap="round"
                    strokeDasharray={`${(data.score / 100) * 440} 440`}
                    className={scoreRingColor(data.score)}
                  />
                </svg>
                <div className="text-center">
                  <span className={`text-4xl font-bold ${scoreColor(data.score)}`}>
                    {data.score}
                  </span>
                  <p className="text-xs text-neutral-500">Security Score</p>
                </div>
              </div>
            </div>

            {/* KPI Cards */}
            <div className="grid grid-cols-4 gap-4">
              <StatCard
                label="MFA Adoption"
                value={`${data.mfaAdoption.toFixed(0)}%`}
                sub="of admin users"
                alert={data.mfaAdoption < 100}
              />
              <StatCard
                label="Vulnerabilities"
                value={data.vulnerabilities}
                sub="open CVEs"
                alert={data.vulnerabilities > 0}
              />
              <StatCard
                label="Policy Violations"
                value={data.policyViolations}
                sub="Kyverno violations"
                alert={data.policyViolations > 0}
              />
              <StatCard
                label="Failed Logins (24h)"
                value={data.failedLogins}
                sub="last 24 hours"
                alert={data.failedLogins > 10}
              />
            </div>

            {/* Additional Metrics */}
            <div className="grid grid-cols-3 gap-4">
              <StatCard
                label="Active Sessions"
                value={data.activeSessions}
                sub="currently active"
              />
              <StatCard
                label="API Keys"
                value={data.activeApiKeys}
                sub="active keys"
              />
              <StatCard
                label="Certificates Expiring"
                value={data.certsExpiringSoon}
                sub="within 30 days"
                alert={data.certsExpiringSoon > 0}
              />
            </div>

            {/* Quick Links */}
            <div className="grid grid-cols-3 gap-4">
              {[
                { label: "WAF & Policies", href: "/security/waf", icon: ShieldAlert, desc: "Kyverno policies and WAF rules" },
                { label: "Image Scanning", href: "/security/images", icon: Bug, desc: "Container vulnerability scanning" },
                { label: "Active Sessions", href: "/security/sessions", icon: Eye, desc: "Monitor and terminate sessions" },
              ].map((link) => (
                <Link
                  key={link.href}
                  href={link.href}
                  className="group flex items-start gap-3 rounded-lg border border-border bg-surface-100 p-4 hover:bg-surface-200 transition-colors"
                >
                  <link.icon className="h-5 w-5 text-neutral-500 group-hover:text-white transition-colors" />
                  <div className="flex-1">
                    <div className="flex items-center justify-between">
                      <h3 className="text-sm font-medium text-white">{link.label}</h3>
                      <ArrowUpRight className="h-3 w-3 text-neutral-500 group-hover:text-white transition-colors" />
                    </div>
                    <p className="mt-0.5 text-xs text-neutral-500">{link.desc}</p>
                  </div>
                </Link>
              ))}
            </div>
          </>
        ) : null}
      </div>
    </Shell>
  );
}
