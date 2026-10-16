"use client";
import type { CustomDomain } from "@/lib/api";

const COLORS: Record<string, string> = {
  active: "text-emerald-400",
  verified: "text-blue-400",
  pending: "text-amber-400",
};

export function DomainStatus({ domain, onCheck, checking }: { domain: CustomDomain; onCheck: () => void; checking: boolean }) {
  const color = COLORS[domain.status] ?? "text-red-400";
  return (
    <div className="space-y-1">
      <div className="flex items-center gap-3">
        <span className={`inline-flex items-center gap-1.5 text-xs ${color}`}>
          <span className={`h-1.5 w-1.5 rounded-full ${domain.status === "pending" ? "animate-pulse" : ""} bg-current`} />
          {domain.status}
        </span>
        {domain.tls_ready && <span className="text-xs text-emerald-400">HTTPS ready</span>}
        <button
          type="button"
          onClick={onCheck}
          disabled={checking}
          className="rounded border border-border px-2 py-0.5 text-[11px] text-neutral-400 hover:text-white disabled:opacity-50 transition-colors"
        >
          {checking ? "Checking…" : "Check now"}
        </button>
      </div>
      {domain.status_message && <p className="text-xs text-neutral-500">{domain.status_message}</p>}
      {domain.last_checked_at && (
        <p className="text-[11px] text-neutral-600">checked {new Date(domain.last_checked_at).toLocaleTimeString()}</p>
      )}
    </div>
  );
}
