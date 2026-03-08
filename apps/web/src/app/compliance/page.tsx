"use client";

import { Shell } from "@/components/shell";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import { type ComplianceCheck } from "@/lib/api";
import { useMemo, useState } from "react";
import {
  Shield,
  CheckCircle2,
  XCircle,
  AlertCircle,
  MinusCircle,
  ShieldCheck,
  ChevronDown,
  ChevronRight,
} from "lucide-react";

const CATEGORIES = [
  "Authentication",
  "Encryption",
  "Audit",
  "Access Control",
  "GDPR",
] as const;

function statusIcon(status: string) {
  switch (status) {
    case "pass":
      return <CheckCircle2 className="h-5 w-5 text-green-400 shrink-0" />;
    case "fail":
      return <XCircle className="h-5 w-5 text-red-400 shrink-0" />;
    case "partial":
      return <AlertCircle className="h-5 w-5 text-yellow-400 shrink-0" />;
    case "na":
      return <MinusCircle className="h-5 w-5 text-neutral-500 shrink-0" />;
    default:
      return <MinusCircle className="h-5 w-5 text-neutral-500 shrink-0" />;
  }
}

function statusLabel(status: string): string {
  switch (status) {
    case "pass":
      return "Pass";
    case "fail":
      return "Fail";
    case "partial":
      return "Partial";
    case "na":
      return "N/A";
    default:
      return status;
  }
}

function statusBadgeColor(status: string): string {
  switch (status) {
    case "pass":
      return "bg-green-500/15 text-green-400";
    case "fail":
      return "bg-red-500/15 text-red-400";
    case "partial":
      return "bg-yellow-500/15 text-yellow-400";
    case "na":
      return "bg-neutral-500/15 text-neutral-400";
    default:
      return "bg-neutral-500/15 text-neutral-400";
  }
}

export default function CompliancePage() {
  const { compliance } = getApi();

  const {
    data: complianceData,
    loading,
    error,
    refetch,
  } = useApi(() => compliance.getStatus(), []);

  const [collapsedCategories, setCollapsedCategories] = useState<Set<string>>(
    new Set()
  );

  const toggleCategory = (cat: string) => {
    setCollapsedCategories((prev) => {
      const next = new Set(prev);
      if (next.has(cat)) {
        next.delete(cat);
      } else {
        next.add(cat);
      }
      return next;
    });
  };

  const grouped = useMemo(() => {
    if (!complianceData?.checks) return new Map<string, ComplianceCheck[]>();
    const map = new Map<string, ComplianceCheck[]>();
    for (const cat of CATEGORIES) {
      map.set(cat, []);
    }
    for (const check of complianceData.checks) {
      const existing = map.get(check.category);
      if (existing) {
        existing.push(check);
      } else {
        map.set(check.category, [check]);
      }
    }
    return map;
  }, [complianceData]);

  const score = useMemo(() => {
    if (!complianceData?.summary) return 0;
    const { total, pass, na } = complianceData.summary;
    const applicable = total - na;
    if (applicable <= 0) return 100;
    return Math.round((pass / applicable) * 100);
  }, [complianceData]);

  const soc2Label = score >= 80 ? "On Track" : "Needs Attention";
  const isoLabel = score >= 80 ? "On Track" : "Needs Attention";
  const readinessColor =
    score >= 80
      ? "bg-green-500/15 text-green-400"
      : "bg-yellow-500/15 text-yellow-400";

  if (loading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={4} rows={6} />
      </Shell>
    );
  }

  if (error) {
    return (
      <Shell>
        <ErrorState message={error} onRetry={refetch} />
      </Shell>
    );
  }

  const summary = complianceData?.summary ?? {
    total: 0,
    pass: 0,
    fail: 0,
    partial: 0,
    na: 0,
  };

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <div className="flex items-center gap-2">
            <Shield className="h-5 w-5 text-accent-400" />
            <h1 className="text-lg font-semibold text-white">
              Compliance Dashboard
            </h1>
          </div>
          <p className="text-sm text-neutral-500 mt-1">
            Review your compliance readiness across security and privacy
            standards
          </p>
        </div>

        {/* Score + Readiness labels */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Compliance Score Ring */}
          <div className="rounded-xl border border-border bg-surface-100 p-6 flex flex-col items-center justify-center">
            <div className="relative h-28 w-28 mb-3">
              <svg className="h-28 w-28 -rotate-90" viewBox="0 0 120 120">
                <circle
                  cx="60"
                  cy="60"
                  r="52"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="8"
                  className="text-surface-300"
                />
                <circle
                  cx="60"
                  cy="60"
                  r="52"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="8"
                  strokeLinecap="round"
                  strokeDasharray={`${(score / 100) * 2 * Math.PI * 52} ${2 * Math.PI * 52}`}
                  className={
                    score >= 80
                      ? "text-green-400"
                      : score >= 50
                        ? "text-yellow-400"
                        : "text-red-400"
                  }
                />
              </svg>
              <div className="absolute inset-0 flex flex-col items-center justify-center">
                <span
                  className={`text-2xl font-bold ${
                    score >= 80
                      ? "text-green-400"
                      : score >= 50
                        ? "text-yellow-400"
                        : "text-red-400"
                  }`}
                >
                  {score}%
                </span>
                <span className="text-[10px] text-neutral-500 uppercase tracking-wide">
                  Score
                </span>
              </div>
            </div>
            <p className="text-sm font-medium text-neutral-300">
              Compliance Score
            </p>
          </div>

          {/* SOC 2 Readiness */}
          <div className="rounded-xl border border-border bg-surface-100 p-6 flex flex-col items-center justify-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-accent-500/10 mb-3">
              <ShieldCheck className="h-7 w-7 text-accent-400" />
            </div>
            <p className="text-sm font-medium text-neutral-300 mb-2">
              SOC 2 Readiness
            </p>
            <span
              className={`inline-flex rounded-full px-3 py-1 text-xs font-medium ${readinessColor}`}
            >
              {soc2Label}
            </span>
          </div>

          {/* ISO 27001 Readiness */}
          <div className="rounded-xl border border-border bg-surface-100 p-6 flex flex-col items-center justify-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-accent-500/10 mb-3">
              <Shield className="h-7 w-7 text-accent-400" />
            </div>
            <p className="text-sm font-medium text-neutral-300 mb-2">
              ISO 27001 Readiness
            </p>
            <span
              className={`inline-flex rounded-full px-3 py-1 text-xs font-medium ${readinessColor}`}
            >
              {isoLabel}
            </span>
          </div>
        </div>

        {/* Summary stat cards */}
        <div className="grid grid-cols-2 sm:grid-cols-5 gap-4">
          {[
            {
              label: "Total Checks",
              value: summary.total,
              color: "text-white",
            },
            {
              label: "Passed",
              value: summary.pass,
              color: "text-green-400",
            },
            {
              label: "Failed",
              value: summary.fail,
              color: "text-red-400",
            },
            {
              label: "Partial",
              value: summary.partial,
              color: "text-yellow-400",
            },
            {
              label: "N/A",
              value: summary.na,
              color: "text-neutral-400",
            },
          ].map((stat) => (
            <div
              key={stat.label}
              className="rounded-xl border border-border bg-surface-100 p-4"
            >
              <p className="text-xs text-neutral-500">{stat.label}</p>
              <p className={`text-2xl font-semibold ${stat.color}`}>
                {stat.value}
              </p>
            </div>
          ))}
        </div>

        {/* Category sections */}
        <div className="space-y-3">
          {Array.from(grouped.entries()).map(([category, checks]) => {
            const isCollapsed = collapsedCategories.has(category);
            const catPass = checks.filter((c) => c.status === "pass").length;
            const catTotal = checks.length;

            return (
              <div
                key={category}
                className="rounded-xl border border-border bg-surface-100 overflow-hidden"
              >
                {/* Category header */}
                <button
                  onClick={() => toggleCategory(category)}
                  className="flex w-full items-center justify-between px-4 py-3 text-left transition-colors hover:bg-surface-200"
                >
                  <div className="flex items-center gap-3">
                    {isCollapsed ? (
                      <ChevronRight className="h-4 w-4 text-neutral-500" />
                    ) : (
                      <ChevronDown className="h-4 w-4 text-neutral-500" />
                    )}
                    <span className="text-sm font-medium text-white">
                      {category}
                    </span>
                    <span className="text-xs text-neutral-500">
                      {catPass}/{catTotal} passed
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    {/* Mini progress bar */}
                    <div className="hidden sm:block w-24 h-1.5 rounded-full bg-surface-300 overflow-hidden">
                      <div
                        className={`h-full rounded-full transition-all ${
                          catTotal > 0 && catPass === catTotal
                            ? "bg-green-400"
                            : catPass > 0
                              ? "bg-yellow-400"
                              : "bg-red-400"
                        }`}
                        style={{
                          width:
                            catTotal > 0
                              ? `${(catPass / catTotal) * 100}%`
                              : "0%",
                        }}
                      />
                    </div>
                  </div>
                </button>

                {/* Check items */}
                {!isCollapsed && checks.length > 0 && (
                  <div className="border-t border-border divide-y divide-border">
                    {checks.map((check, idx) => (
                      <div
                        key={`${check.category}-${check.item}-${idx}`}
                        className="flex items-start gap-3 px-4 py-3 hover:bg-surface-50 transition-colors"
                      >
                        {statusIcon(check.status)}
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium text-neutral-200">
                              {check.item}
                            </span>
                            <span
                              className={`inline-flex rounded-full px-2 py-0.5 text-[10px] font-medium ${statusBadgeColor(check.status)}`}
                            >
                              {statusLabel(check.status)}
                            </span>
                          </div>
                          <p className="text-xs text-neutral-500 mt-0.5">
                            {check.description}
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                )}

                {/* Empty category */}
                {!isCollapsed && checks.length === 0 && (
                  <div className="border-t border-border px-4 py-6 text-center">
                    <p className="text-sm text-neutral-500">
                      No checks in this category
                    </p>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </Shell>
  );
}
