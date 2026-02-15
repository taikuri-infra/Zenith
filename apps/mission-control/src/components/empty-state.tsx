"use client";

import { Inbox } from "lucide-react";

interface EmptyStateProps {
  /** Title text */
  title?: string;
  /** Description below the title */
  description?: string;
  /** Optional icon override -- pass a Lucide icon component */
  icon?: React.ComponentType<{ className?: string }>;
  /** Optional action button */
  action?: {
    label: string;
    onClick: () => void;
  };
}

export function EmptyState({
  title = "No data",
  description = "There is nothing to display yet.",
  icon: Icon = Inbox,
  action,
}: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-border bg-surface-100 px-6 py-12">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-surface-300">
        <Icon className="h-6 w-6 text-neutral-500" />
      </div>
      <h3 className="mt-4 text-sm font-medium text-white">{title}</h3>
      <p className="mt-1 text-xs text-neutral-500">{description}</p>
      {action && (
        <button
          onClick={action.onClick}
          className="mt-4 rounded-lg bg-accent-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-500"
        >
          {action.label}
        </button>
      )}
    </div>
  );
}
