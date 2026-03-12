"use client";

import { Zap } from "lucide-react";
import Link from "next/link";

interface UpgradeNudgeProps {
  resource: string;
  current: number;
  limit: number;
  className?: string;
}

export function UpgradeNudge({ resource, current, limit, className = "" }: UpgradeNudgeProps) {
  if (current < limit) return null;

  return (
    <div className={`rounded-lg border border-amber-500/30 bg-amber-500/5 px-4 py-3 ${className}`}>
      <div className="flex items-start gap-3">
        <Zap className="mt-0.5 h-5 w-5 flex-shrink-0 text-amber-400" />
        <div className="flex-1">
          <p className="text-sm font-medium text-amber-300">
            You&apos;re using {current}/{limit} {resource} on Free
          </p>
          <p className="mt-1 text-xs text-neutral-400">
            Upgrade to Pro for more {resource}, custom domains, and always-on deployments.
          </p>
          <div className="mt-3 flex items-center gap-3">
            <Link
              href="/billing"
              className="rounded-lg bg-accent-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-600 transition-colors"
            >
              Upgrade to Pro
            </Link>
            <Link
              href="/billing"
              className="text-xs text-neutral-400 hover:text-neutral-300 transition-colors"
            >
              Start 7-day Free Trial
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}

interface PlanLimitBarProps {
  label: string;
  current: number;
  limit: number;
  className?: string;
}

export function PlanLimitBar({ label, current, limit, className = "" }: PlanLimitBarProps) {
  if (limit <= 0) return null;

  const percent = Math.round((current / limit) * 100);
  const barColor =
    percent >= 100
      ? "bg-red-500"
      : percent >= 80
        ? "bg-amber-500"
        : "bg-accent-500";

  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-1">
        <span className="text-xs text-neutral-500">{label}</span>
        <span className="text-xs text-neutral-300">
          {current}/{limit}
          {percent >= 100 && (
            <Link href="/billing" className="ml-2 text-amber-400 hover:text-amber-300">
              Upgrade
            </Link>
          )}
        </span>
      </div>
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-surface-200">
        <div
          className={`h-full rounded-full transition-all ${barColor}`}
          style={{ width: `${Math.min(percent, 100)}%` }}
        />
      </div>
    </div>
  );
}
