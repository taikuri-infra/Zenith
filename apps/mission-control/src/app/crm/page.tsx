"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { Skeleton } from "@/components/loading-skeleton";
import { getApi } from "@/lib/get-api";
import type { CrmPipeline, CrmCustomerCard } from "@/lib/api";
import { useApi } from "@/hooks/use-api";
import Link from "next/link";
import { Users, ArrowRight } from "lucide-react";

const stageLabels: Record<string, string> = {
  trial: "Trial",
  active: "Active",
  at_risk: "At Risk",
  churned: "Churned",
};

const stageColors: Record<string, string> = {
  trial: "border-blue-500/30",
  active: "border-emerald-500/30",
  at_risk: "border-amber-500/30",
  churned: "border-red-500/30",
};

const stageHeaderColors: Record<string, string> = {
  trial: "text-blue-400",
  active: "text-emerald-400",
  at_risk: "text-amber-400",
  churned: "text-red-400",
};

function healthBadge(score: number) {
  if (score >= 80) return <StatusBadge status="healthy" label={`${score}`} />;
  if (score >= 50) return <StatusBadge status="warning" label={`${score}`} />;
  return <StatusBadge status="error" label={`${score}`} />;
}

export default function CrmPage() {
  const apiClient = getApi();
  const pipeline = useApi<CrmPipeline>(() => apiClient.crm.pipeline());

  const stages = ["trial", "active", "at_risk", "churned"];

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-white">CRM Pipeline</h1>
          <Link
            href="/customers"
            className="flex items-center gap-1 text-xs text-neutral-500 hover:text-white transition-colors"
          >
            All customers <ArrowRight className="h-3 w-3" />
          </Link>
        </div>

        {pipeline.loading ? (
          <div className="grid grid-cols-4 gap-4">
            {stages.map((stage) => (
              <div key={stage} className="rounded-lg border border-border bg-surface-100 p-4">
                <Skeleton className="mb-4 h-4 w-20" />
                <div className="space-y-3">
                  {Array.from({ length: 3 }).map((_, i) => (
                    <Skeleton key={i} className="h-24 w-full rounded-lg" />
                  ))}
                </div>
              </div>
            ))}
          </div>
        ) : pipeline.error ? (
          <ErrorState error={pipeline.error} onRetry={pipeline.refetch} />
        ) : pipeline.data ? (
          <div className="grid grid-cols-4 gap-4">
            {stages.map((stage) => {
              const customers: CrmCustomerCard[] = pipeline.data?.stages[stage] ?? [];
              return (
                <div key={stage} className={`rounded-lg border ${stageColors[stage]} bg-surface-100 p-4`}>
                  <div className="mb-4 flex items-center justify-between">
                    <h2 className={`text-sm font-medium ${stageHeaderColors[stage]}`}>
                      {stageLabels[stage]}
                    </h2>
                    <span className="rounded-full bg-surface-300 px-2 py-0.5 text-xs text-neutral-400">
                      {customers.length}
                    </span>
                  </div>

                  {customers.length === 0 ? (
                    <p className="text-xs text-neutral-500">No customers in this stage.</p>
                  ) : (
                    <div className="space-y-3">
                      {customers.map((customer) => (
                        <Link
                          key={customer.id}
                          href={`/customers/${customer.id}`}
                          className="block rounded-lg border border-border bg-surface-200 p-3 hover:bg-surface-300 transition-colors"
                        >
                          <div className="flex items-center justify-between">
                            <span className="text-sm font-medium text-white truncate">{customer.name}</span>
                            {healthBadge(customer.healthScore)}
                          </div>
                          <p className="mt-1 text-xs text-neutral-500 truncate">{customer.email}</p>
                          <div className="mt-2 flex items-center justify-between">
                            <span className="rounded-full bg-surface-100 px-2 py-0.5 text-[10px] text-neutral-400 capitalize">
                              {customer.plan}
                            </span>
                            <span className="text-[10px] text-neutral-500">
                              €{customer.mrr}/mo
                            </span>
                          </div>
                        </Link>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        ) : (
          <EmptyState
            title="No pipeline data"
            description="CRM pipeline data is not available yet."
            icon={Users}
          />
        )}
      </div>
    </Shell>
  );
}
