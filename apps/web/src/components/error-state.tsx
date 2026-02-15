/**
 * Reusable error state component for the Zenith web platform.
 */

interface ErrorStateProps {
  /** The error message to display. */
  message?: string;
  /** Optional retry callback. When provided, a retry button is shown. */
  onRetry?: () => void;
}

export function ErrorState({
  message = "Something went wrong",
  onRetry,
}: ErrorStateProps) {
  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-red-500/20 bg-red-500/5 px-6 py-12">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10">
        <svg
          className="h-6 w-6 text-red-400"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4.5c-.77-.833-2.694-.833-3.464 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z"
          />
        </svg>
      </div>
      <p className="mb-1 text-sm font-medium text-white">Error</p>
      <p className="mb-4 max-w-sm text-center text-sm text-neutral-400">
        {message}
      </p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-600"
        >
          Try Again
        </button>
      )}
    </div>
  );
}
