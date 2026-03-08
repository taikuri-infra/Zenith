"use client";

import { useState, useEffect } from "react";
import { ShieldAlert, X } from "lucide-react";
import Link from "next/link";
import { getApi, isDemoMode } from "@/lib/get-api";

const MFA_BANNER_DISMISSED_KEY = "zenith_mfa_banner_dismissed";

/**
 * Shows a security banner prompting Pro+ users to set up MFA
 * if they haven't done so yet. Dismissible with localStorage persistence.
 */
export function MFABanner() {
  const [show, setShow] = useState(false);

  useEffect(() => {
    if (isDemoMode()) return;

    // Check if already dismissed this session
    const dismissed = sessionStorage.getItem(MFA_BANNER_DISMISSED_KEY);
    if (dismissed) return;

    const checkMFA = async () => {
      try {
        const api = getApi();
        const [plan, mfaStatus] = await Promise.all([
          api.userPlan.get(),
          api.mfa.getStatus(),
        ]);

        // Only show for Pro+ users without MFA enabled
        const proPlus = ["pro", "team", "business", "enterprise"];
        if (proPlus.includes(plan.tier) && mfaStatus.status !== "enabled") {
          setShow(true);
        }
      } catch {
        // Silently ignore — don't block the UI
      }
    };

    checkMFA();
  }, []);

  if (!show) return null;

  return (
    <div className="sticky top-0 z-50 flex items-center justify-between gap-3 bg-amber-600/10 border-b border-amber-600/20 px-4 py-2 text-xs text-amber-400 backdrop-blur-sm">
      <div className="flex items-center gap-2">
        <ShieldAlert className="h-3.5 w-3.5 shrink-0" />
        <span>
          <span className="font-semibold">Secure your account</span> — Enable
          two-factor authentication for additional security.{" "}
          <Link
            href="/settings"
            className="underline underline-offset-2 hover:text-amber-300 transition-colors"
          >
            Set up MFA
          </Link>
        </span>
      </div>
      <button
        onClick={() => {
          setShow(false);
          sessionStorage.setItem(MFA_BANNER_DISMISSED_KEY, "1");
        }}
        className="shrink-0 p-0.5 hover:text-amber-300 transition-colors"
        aria-label="Dismiss"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
