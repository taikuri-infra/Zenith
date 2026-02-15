"use client";

import { isDemoMode } from "@/lib/get-api";
import { Eye } from "lucide-react";

/**
 * Renders a subtle banner at the top of the page when demo mode is active.
 * Completely hidden in production / real-API mode.
 */
export function DemoBanner() {
  if (!isDemoMode()) return null;

  return (
    <div className="sticky top-0 z-50 flex items-center justify-center gap-2 bg-accent-600/10 border-b border-accent-600/20 px-4 py-1.5 text-xs text-accent-400 backdrop-blur-sm">
      <Eye className="h-3.5 w-3.5" />
      <span>
        <span className="font-semibold">Demo Mode</span> &mdash; Viewing with
        sample data.{" "}
        <a
          href="https://freezenith.com"
          target="_blank"
          rel="noopener noreferrer"
          className="underline underline-offset-2 hover:text-accent-300 transition-colors"
        >
          Install Zenith
        </a>{" "}
        to deploy your own apps.
      </span>
    </div>
  );
}
