/**
 * Reusable empty state component for the Zenith web platform.
 * Shown when a list has no items (no apps, no databases, etc.).
 */

interface EmptyStateProps {
  /** Title text, e.g. "No apps yet" */
  title: string;
  /** Description text */
  description?: string;
  /** Optional action button label */
  actionLabel?: string;
  /** Optional action button callback */
  onAction?: () => void;
}

export function EmptyState({
  title,
  description,
  actionLabel,
  onAction,
}: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-border bg-surface-100 px-6 py-16">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-surface-300">
        <svg
          className="h-6 w-6 text-neutral-500"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={1.5}
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
          />
        </svg>
      </div>
      <p className="mb-1 text-sm font-medium text-white">{title}</p>
      {description && (
        <p className="mb-4 max-w-sm text-center text-sm text-neutral-500">
          {description}
        </p>
      )}
      {actionLabel && onAction && (
        <button
          onClick={onAction}
          className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-600"
        >
          {actionLabel}
        </button>
      )}
    </div>
  );
}
