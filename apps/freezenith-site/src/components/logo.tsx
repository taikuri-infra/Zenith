import { cn } from "@/lib/utils";

export function Logo({ className }: { className?: string }) {
  return (
    <span className={cn("flex items-center gap-2.5", className)}>
      <span className="relative flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-accent-400 to-accent-600 shadow-lg shadow-accent-500/20">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden>
          <path d="M8 1L14 5V11L8 15L2 11V5L8 1Z" fill="white" fillOpacity="0.9" />
          <path d="M8 1L14 5L8 9L2 5L8 1Z" fill="white" />
        </svg>
      </span>
      <span className="text-lg font-bold tracking-tight text-white">
        Free<span className="text-accent-400">Zenith</span>
      </span>
    </span>
  );
}
