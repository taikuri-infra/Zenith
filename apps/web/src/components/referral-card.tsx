"use client";

import { useState } from "react";
import { Gift, Copy, Check, ExternalLink } from "lucide-react";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";

export function ReferralCard() {
  const { referral } = getApi();
  const { data, loading } = useApi(() => referral.getSummary(), []);
  const [copied, setCopied] = useState(false);

  const copyLink = async () => {
    if (!data?.link) return;
    await navigator.clipboard.writeText(data.link);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
    try {
      await referral.trackShare();
    } catch {
      // non-blocking
    }
  };

  if (loading) {
    return (
      <div className="rounded-lg border border-border bg-surface-100 p-6 animate-pulse">
        <div className="h-4 w-48 rounded bg-surface-200" />
        <div className="mt-3 h-3 w-64 rounded bg-surface-200" />
      </div>
    );
  }

  if (!data) return null;

  return (
    <div className="rounded-lg border border-accent-500/20 bg-surface-100 p-6">
      <div className="flex items-start gap-3">
        <Gift className="mt-0.5 h-5 w-5 flex-shrink-0 text-accent-400" />
        <div className="flex-1">
          <h3 className="text-sm font-semibold text-white">Refer a friend, get 1 month Pro free</h3>
          <p className="mt-1 text-xs text-neutral-400">
            Share your referral link. When someone signs up and deploys their first app, you both get rewarded.
          </p>

          <div className="mt-4 flex items-center gap-2">
            <div className="flex-1 rounded-lg border border-border bg-surface-200 px-3 py-2">
              <p className="truncate font-mono text-xs text-accent-400">{data.link}</p>
            </div>
            <button
              onClick={copyLink}
              className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-3 py-2 text-xs font-medium text-white hover:bg-accent-600 transition-colors"
            >
              {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
              {copied ? "Copied" : "Copy"}
            </button>
          </div>

          <div className="mt-4 flex items-center gap-4 text-xs text-neutral-400">
            <span>{data.total_referrals} total</span>
            <span className="text-emerald-400">{data.credited} credited</span>
            <span className="text-amber-400">{data.pending} pending</span>
          </div>

          <a
            href={`https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(data.link)}`}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-3 inline-flex items-center gap-1 text-xs text-neutral-500 hover:text-neutral-300"
          >
            Share on LinkedIn <ExternalLink className="h-3 w-3" />
          </a>
        </div>
      </div>
    </div>
  );
}
