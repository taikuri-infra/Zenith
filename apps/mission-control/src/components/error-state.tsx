"use client";

import { AlertTriangle, RefreshCw } from "lucide-react";

interface ErrorStateProps {
  /** The error to display */
  error: Error;
  /** Callback to retry the failed operation */
  onRetry?: () => void;
  /** Optional title override */
  title?: string;
}

export function ErrorState({
  error,
  onRetry,
  title = "Something went wrong",
}: ErrorStateProps) {
  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-red-500/20 bg-red-500/5 px-6 py-12">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10">
        <AlertTriangle className="h-6 w-6 text-red-400" />
      </div>
      <h3 className="mt-4 text-sm font-medium text-white">{title}</h3>
      <p className="mt-1 max-w-md text-center text-xs text-neutral-500">
        {error.message}
      </p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="mt-4 flex items-center gap-1.5 rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm font-medium text-neutral-300 transition-colors hover:bg-surface-200 hover:text-white"
        >
          <RefreshCw className="h-3.5 w-3.5" />
          Retry
        </button>
      )}
    </div>
  );
}
